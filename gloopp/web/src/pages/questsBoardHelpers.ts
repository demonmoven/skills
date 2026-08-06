import type { QuestMeta, QuestStatus, QuestType } from '../api/types'
import type { GroupBy } from '../components/GroupPopover'
import type { SortBy, SortDir } from '../components/SortPopover'
import { hasApplyFailed, fmtBlockedReason, humanizeMilliseconds, typeLabel, statusLabel } from '../components/util'
import { questPhaseView, phaseDisplayName } from '../domain/questSelectors'
import { compareStuckDiagnoses, getStuckDiagnosis, type StuckDiagnosisKind } from '../features/quest/stuckDiagnosisMetrics'
import i18n from '../i18n'

export type GroupDef = {
  key: string
  title: string
  dot?: string
  attention?: boolean
  items: QuestMeta[]
}

export const STATUS_BOARD_ORDER: { key: string; status: QuestStatus; wide?: boolean; applyFailed?: boolean }[] = [
  { key: 'pending', status: 'pending' },
  { key: 'running', status: 'running' },
  { key: 'reviewing', status: 'reviewing' },
  { key: 'waiting_input', status: 'waiting_input', wide: true },
  { key: 'user_review', status: 'user_review', wide: true },
  { key: 'apply_failed', status: 'success', wide: true, applyFailed: true },
  { key: 'blocked', status: 'blocked' },
  { key: 'success', status: 'success' },
  { key: 'failed', status: 'failed' },
  { key: 'cancelled', status: 'cancelled' },
]

export const STATUS_LIST_ORDER: QuestStatus[] = [
  'user_review', 'waiting_input', 'blocked', 'reviewing', 'running', 'success', 'failed', 'cancelled', 'pending',
]

export const MOBILE_ORDER: Record<QuestStatus, number> = {
  user_review: 1,
  waiting_input: 2,
  blocked: 3,
  reviewing: 4,
  running: 5,
  pending: 6,
  success: 7,
  failed: 8,
  cancelled: 9,
}

export const GROUP_MOBILE_ORDER: Record<string, number> = {
  user_review: 1,
  apply_failed: 2,
  blocked: 3,
  waiting_input: 4,
}

export type AttentionSection = {
  key: string
  title: string
  icon: string
  description: string
  items: QuestMeta[]
  tone: 'amber' | 'red'
  diagnosisKind?: StuckDiagnosisKind
}

export type PhaseColumn = {
  key: string
  title: string
  role?: string
  items: QuestMeta[]
}

export type PhaseLane = {
  key: string
  title: string
  subtitle: string
  columns: PhaseColumn[]
  legacy?: boolean
}

export function questActivityTime(q: QuestMeta): number {
  return Number(q.updated_at_ms ?? q.completed_at_ms ?? q.started_at_ms ?? q.created_at_ms) || 0
}

export function sortQuests(list: QuestMeta[], sortBy: SortBy, sortDir: SortDir): QuestMeta[] {
  return [...list].sort((a, b) => {
    let va: number, vb: number
    switch (sortBy) {
      case 'created_at_ms':
        va = a.created_at_ms || 0; vb = b.created_at_ms || 0; break
      case 'updated_at_ms':
        va = questActivityTime(a)
        vb = questActivityTime(b); break
      case 'mage_score':
        va = a.mage_score ?? -1; vb = b.mage_score ?? -1; break
      default:
        va = a.created_at_ms || 0; vb = b.created_at_ms || 0
    }
    return sortDir === 'desc' ? vb - va : va - vb
  })
}

export function buildGroups(quests: QuestMeta[], groupBy: GroupBy, viewMode: 'feed' | 'board' | 'list'): GroupDef[] {
  if (groupBy === 'none') {
    return [{ key: 'all', title: i18n.t('common.action.all'), items: quests }]
  }
  if (groupBy === 'type') {
    return (['execute', 'design'] as QuestType[]).map(t => ({
      key: t,
      title: typeLabel(t),
      items: quests.filter(q => q.type === t),
    })).filter(g => g.items.length > 0)
  }
  // groupBy === 'status'
  if (viewMode === 'board') {
    const known = new Set(STATUS_BOARD_ORDER.map((col) => col.status))
    const groups: GroupDef[] = STATUS_BOARD_ORDER.map((col): GroupDef => ({
      key: col.key,
      title: col.applyFailed ? i18n.t('quest.board.applyFailed') : statusLabel(col.status),
      dot: col.status,
      attention: col.status === 'user_review' || col.status === 'waiting_input' || col.status === 'blocked' || col.applyFailed === true,
      items: quests.filter(q => q.status === col.status && (col.applyFailed ? hasApplyFailed(q) : !(col.status === 'success' && hasApplyFailed(q)))),
    }))
    const fallbackItems = quests.filter(q => !known.has(q.status))
    if (fallbackItems.length > 0) {
      groups.push({
        key: 'unknown',
        title: i18n.t('quest.board.other'),
        items: fallbackItems,
      })
    }
    return groups
  }
  const known = new Set(STATUS_LIST_ORDER)
  const groups: GroupDef[] = STATUS_LIST_ORDER.map((s): GroupDef => ({
    key: s,
    title: statusLabel(s),
    dot: s,
    attention: s === 'user_review' || s === 'waiting_input' || s === 'blocked',
    items: quests.filter(q => q.status === s && !(s === 'success' && hasApplyFailed(q))),
  })).filter(g => g.items.length > 0)
  const fallbackItems = quests.filter(q => !known.has(q.status))
  if (fallbackItems.length > 0) {
    groups.push({ key: 'unknown', title: i18n.t('quest.board.other'), items: fallbackItems })
  }
  return groups
}

export function failureSummary(quest: QuestMeta): { title: string; meta: string; recoverable?: boolean } | null {
  const failure = quest.failure_attribution
  if (!failure) {
    return quest.blocked_reason ? { title: fmtBlockedReason(quest), meta: 'blocked' } : null
  }
  const meta = [failure.stage, failure.category].filter(Boolean).join(' · ') || 'blocked'
  return {
    title: humanizeMilliseconds(String(failure.reason || fmtBlockedReason(quest) || i18n.t('quest.failure.needHandle'))),
    meta,
    recoverable: failure.recoverable,
  }
}

export function waitDuration(quest: QuestMeta): { text: string; urgency: 'normal' | 'amber' | 'red' } {
  const now = Date.now()
  const since = questActivityTime(quest) || quest.created_at_ms || now
  const mins = Math.floor((now - since) / 60000)
  if (mins < 10) return { text: '<10m', urgency: 'normal' }
  if (mins < 60) return { text: `${mins}m`, urgency: 'amber' }
  const hours = Math.floor(mins / 60)
  if (hours < 24) return { text: `${hours}h`, urgency: 'red' }
  const days = Math.floor(hours / 24)
  return { text: `${days}d`, urgency: 'red' }
}

export function isRunningTooLong(quest: QuestMeta): boolean {
  if (quest.status !== 'running' && quest.status !== 'reviewing') return false
  const now = Date.now()
  const started = quest.started_at_ms || quest.created_at_ms || now
  const elapsed = now - started
  const maxTurns = quest.max_turns_per_phase_override || 20
  const expectedMs = maxTurns * 2 * 60000
  return elapsed > expectedMs
}

export function focusPriority(quest: QuestMeta): number {
  if (quest.status === 'user_review') return 0
  if (hasApplyFailed(quest)) return 0
  if (quest.status === 'blocked') return 0
  if (quest.status === 'failed') return 1
  if (quest.status === 'waiting_input') return 1
  if (quest.status === 'running' || quest.status === 'reviewing') return 2
  if (quest.final_verdict === 'reject') return 3
  if (quest.final_verdict === 'request_changes') return 4
  if (quest.final_verdict === 'pass') return 5
  return 6
}

export function focusPrimaryLabel(quest: QuestMeta | undefined): string {
  if (!quest) return i18n.t('quest.focus.create')
  if (hasApplyFailed(quest)) return i18n.t('quest.focus.applyFailed')
  if (quest.status === 'blocked') return i18n.t('quest.focus.blocked')
  if (quest.status === 'failed') return i18n.t('quest.focus.failed')
  if (quest.status === 'waiting_input') return i18n.t('quest.focus.waitingInput')
  if (quest.status === 'running' || quest.status === 'reviewing') return i18n.t('quest.focus.timeout')
  if (quest.final_verdict === 'pass') return i18n.t('quest.focus.result')
  return i18n.t('quest.focus.default')
}

export function buildAttentionSections(sorted: QuestMeta[]): AttentionSection[] {
  const stuckSorted = [...sorted].filter((q) => getStuckDiagnosis(q)).sort(compareStuckDiagnoses)
  const byDiagnosis = (kind: StuckDiagnosisKind) => stuckSorted.filter((q) => getStuckDiagnosis(q)?.kind === kind)
  const userReview = sorted.filter((q) => q.status === 'user_review' && !getStuckDiagnosis(q))
  const sections: AttentionSection[] = [
    {
      key: 'user_review',
      title: i18n.t('quest.attention.userReview.title'),
      icon: 'shield',
      description: i18n.t('quest.attention.userReview.desc'),
      tone: 'amber',
      items: userReview,
    },
    {
      key: 'blocked',
      title: i18n.t('quest.attention.blocked.title'),
      icon: 'alert-triangle',
      description: i18n.t('quest.attention.blocked.desc'),
      tone: 'red',
      diagnosisKind: 'blocked',
      items: byDiagnosis('blocked'),
    },
    {
      key: 'apply_failed',
      title: i18n.t('quest.attention.applyFailed.title'),
      icon: 'x',
      description: i18n.t('quest.attention.applyFailed.desc'),
      tone: 'red',
      diagnosisKind: 'apply_failed',
      items: byDiagnosis('apply_failed'),
    },
    {
      key: 'waiting_input',
      title: i18n.t('quest.attention.waitingInput.title'),
      icon: 'message-circle',
      description: i18n.t('quest.attention.waitingInput.desc'),
      tone: 'amber',
      diagnosisKind: 'waiting_input',
      items: byDiagnosis('waiting_input'),
    },
    {
      key: 'runtime_protocol_warning',
      title: i18n.t('quest.attention.protocolWarning.title'),
      icon: 'radio',
      description: i18n.t('quest.attention.protocolWarning.desc'),
      tone: 'amber',
      diagnosisKind: 'runtime.protocol_warning',
      items: byDiagnosis('runtime.protocol_warning'),
    },
    {
      key: 'runtime_idle_warning',
      title: i18n.t('quest.attention.idleWarning.title'),
      icon: 'timer',
      description: i18n.t('quest.attention.idleWarning.desc'),
      tone: 'amber',
      diagnosisKind: 'runtime.idle_warning',
      items: byDiagnosis('runtime.idle_warning'),
    },
    {
      key: 'stuck',
      title: i18n.t('quest.attention.stuck.title'),
      icon: 'timer',
      description: i18n.t('quest.attention.stuck.desc'),
      tone: 'red',
      items: sorted.filter((q) => isRunningTooLong(q) && !getStuckDiagnosis(q)),
    },
  ]
  return sections.filter((section) => section.items.length > 0)
}

function firstPipelineDefs(quests: QuestMeta[]): NonNullable<QuestMeta['pipeline_def']> {
  for (const quest of quests) {
    if (Array.isArray(quest.pipeline_def) && quest.pipeline_def.length > 0) {
      return [...quest.pipeline_def].sort((a, b) => (a.phase_idx ?? 0) - (b.phase_idx ?? 0))
    }
  }
  const phaseCount = Math.max(0, ...quests.map((q) => q.phases?.length || 0))
  return Array.from({ length: phaseCount }, (_, index) => ({ phase_idx: index, display_name: `Phase ${index + 1}` }))
}

export function buildPhaseLanes(quests: QuestMeta[]): PhaseLane[] {
  const lanes = new Map<string, { title: string; hash: string; quests: QuestMeta[] }>()
  const legacy: QuestMeta[] = []
  for (const quest of quests) {
    const view = questPhaseView(quest)
    if (!view.available) {
      legacy.push(quest)
      continue
    }
    const key = view.pipelineName + ':' + view.pipelineHash
    const lane = lanes.get(key) || { title: view.pipelineName, hash: view.pipelineHash, quests: [] }
    lane.quests.push(quest)
    lanes.set(key, lane)
  }
  const out: PhaseLane[] = [...lanes.entries()]
    .sort((a, b) => b[1].quests.length - a[1].quests.length)
    .map(([key, lane]) => {
      const defs = firstPipelineDefs(lane.quests)
      const labelQuest = lane.quests[0]
      const columns: PhaseColumn[] = defs.map((def, index) => ({
        key: String(def.phase_idx ?? index),
        title: phaseDisplayName(labelQuest, index),
        role: def.role,
        items: [],
      }))
      for (const quest of lane.quests) {
        const view = questPhaseView(quest)
        const idx = view.currentIdx >= 0 ? view.currentIdx : 0
        if (!columns[idx]) {
          columns[idx] = { key: String(idx), title: view.name || `Phase ${idx + 1}`, items: [] }
        }
        columns[idx].items.push(quest)
      }
      return {
        key,
        title: lane.title === 'design' ? i18n.t('quest.board.designPipeline') : lane.title,
        subtitle: lane.hash === 'legacy'
          ? i18n.t('quest.board.questCount', { count: lane.quests.length })
          : i18n.t('quest.board.questCountHash', { count: lane.quests.length, hash: lane.hash.slice(0, 8) }),
        columns,
      }
    })
  if (legacy.length > 0) {
    out.push({
      key: 'legacy',
      title: i18n.t('quest.board.legacyTitle'),
      subtitle: i18n.t('quest.board.questCount', { count: legacy.length }),
      legacy: true,
      columns: [{ key: 'legacy', title: i18n.t('quest.board.legacyColumn'), items: legacy }],
    })
  }
  return out
}

export function truncateText(text: string, max: number): string {
  return text.length > max ? text.slice(0, max - 1) + '…' : text
}

export function finalizedByLabel(finalizedBy: string): string {
  switch (finalizedBy) {
    case 'policy':
      return i18n.t('quest.finalizedBy.policy')
    case 'user':
      return i18n.t('quest.finalizedBy.user')
    case 'automation':
      return i18n.t('quest.finalizedBy.automation')
    default:
      return finalizedBy
  }
}
