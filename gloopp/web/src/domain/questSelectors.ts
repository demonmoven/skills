import type { QuestArtifact, QuestEvent, QuestMeta, PhaseStatus, QuestPhaseTask, Verdict, QuestStatus } from '../api/types'
import { fmtBlockedReason, fmtDuration, fmtTime, statusLabel } from '../components/util'
import { getStuckDiagnosis } from '../features/quest/stuckDiagnosisMetrics'
import i18n from '../i18n'

export type QuestActionState = {
  canReview: boolean
  applyFailed: boolean
  hasWorkspaceDiffPending: boolean
  canApply: boolean
  canDiscard: boolean
  canSpawn: boolean
  canStart: boolean
  canStop: boolean
  isQueued: boolean
  isStarting: boolean
  isBlocked: boolean
  isEnded: boolean
  isRunning: boolean
}

export type ReviewActionDraft = {
  verdict: Verdict
  apply?: boolean
  label: string
  tone: 'primary' | 'danger' | 'default'
}

export type QuestResourceUsage = {
  turns: number
  currentPhaseTurns: number
  durationMs: number
  maxTurnsPerPhase?: number
  maxDurationMs?: number
}

export type EffectTypeInfo = {
  key: string
  label: string
  icon: string
  tone: 'muted' | 'blue' | 'amber' | 'violet'
}

export type QuestPhaseView = {
  available: boolean
  currentIdx: number
  count: number
  name: string
  role: string
  status?: string
  startedAtMs?: number
  reworkCount: number
  pipelineName: string
  pipelineHash: string
}

export type PhaseNode = {
  phaseIdx: number
  name: string
  displayName: string
  role: string
  class_?: string
  status: PhaseStatus
  turns: number
  reworkCount: number
  startedAtMs?: number
  endedAtMs?: number
  sessionId?: string
  artifactKinds: string[]
}

export type PolicyChip = {
  key: string
  label: string
  icon: string
  tone: 'muted' | 'blue' | 'amber' | 'violet'
  title?: string
}

export type PhaseEventGroup = {
  phaseIdx: number
  events: QuestEvent[]
}

export type GroupedPhaseEvents = {
  byPhase: Map<number, QuestEvent[]>
  unassigned: QuestEvent[]
}

export type TimelineVariant = 'warrior' | 'mage' | 'user' | 'success' | 'system' | 'note' | 'comment'

export type TimelineEntry = {
  variant: TimelineVariant
  icon: string
  title: string
  meta: string
  body?: string
  status?: 'pass' | 'blocked' | ''
  chips?: string[]
}

function isMagePhase(phase: PhaseNode): boolean {
  return phase.class_ === 'mage' || phase.role === 'review'
}

function isQuickMode(quest: Pick<QuestMeta, 'intensity'> | null | undefined): boolean {
  return String(quest?.intensity || '').toLowerCase() === 'quick'
}

export function hasApplyFailed(quest: { apply_status?: string; apply_failed?: boolean } | null | undefined): boolean {
  return quest?.apply_status === 'failed' || quest?.apply_failed === true
}

export function questResourceUsage(quest: QuestMeta | null | undefined, now = Date.now()): QuestResourceUsage {
  const phases = Array.isArray(quest?.phases) ? quest.phases : []
  const turns = phases.reduce((sum, phase) => sum + (Number(phase.turns) || 0), 0)
  const currentPhaseIdx = typeof quest?.current_phase_idx === 'number' ? quest.current_phase_idx : -1
  const currentPhaseTurns =
    currentPhaseIdx >= 0 && currentPhaseIdx < phases.length
      ? Number(phases[currentPhaseIdx]?.turns) || 0
      : 0
  const started = Number(quest?.started_at_ms) || 0
  const completed = Number(quest?.completed_at_ms) || 0
  const durationMs = started > 0 ? Math.max(0, (completed > 0 ? completed : now) - started) : 0
  const maxTurnsPerPhase = positiveNumber(quest?.max_turns_per_phase_override)
  const maxDurationMs = positiveNumber(quest?.max_duration_per_quest_ms_override)
  return {
    turns,
    currentPhaseTurns,
    durationMs,
    maxTurnsPerPhase,
    maxDurationMs,
  }
}

export function questTurnUsageLabel(quest: QuestMeta | null | undefined): string {
  const usage = questResourceUsage(quest)
  const labelTurns = usage.currentPhaseTurns || usage.turns
  const scope = usage.currentPhaseTurns ? i18n.t('quest.turnUsage.scopeCurrent') : i18n.t('quest.turnUsage.scopeTotal')
  if (usage.maxTurnsPerPhase) return i18n.t('quest.turnUsage.label', { count: `${labelTurns}/${usage.maxTurnsPerPhase}`, scope })
  return i18n.t('quest.turnUsage.label', { count: labelTurns, scope })
}

export function questDurationUsageLabel(quest: QuestMeta | null | undefined, now = Date.now()): string {
  const usage = questResourceUsage(quest, now)
  if (usage.maxDurationMs) return `${fmtDuration(usage.durationMs)}/${fmtDuration(usage.maxDurationMs)}`
  return fmtDuration(usage.durationMs)
}

export function questWaitingInputAgeMs(quest: QuestMeta | null | undefined, now = Date.now()): number {
  const askedAt = quest?.waiting_input?.asked_at ? Date.parse(quest.waiting_input.asked_at) : NaN
  const since = Number.isFinite(askedAt) && askedAt > 0 ? askedAt : Number(quest?.updated_at_ms ?? quest?.started_at_ms ?? quest?.created_at_ms) || now
  return Math.max(0, now - since)
}

export function effectTypeInfo(effectType: string | null | undefined): EffectTypeInfo | null {
  switch (effectType) {
    case 'workspace_diff':
      return { key: effectType, label: i18n.t('quest.effect.workspaceDiff'), icon: 'file-diff', tone: 'blue' }
    case 'external_side_effect':
      return { key: effectType, label: i18n.t('quest.effect.external'), icon: 'external-link', tone: 'amber' }
    case 'context_store':
      return { key: effectType, label: i18n.t('quest.effect.contextStore'), icon: 'database', tone: 'violet' }
    default:
      return null
  }
}

export function artifactKindCounts(quest: QuestMeta | null | undefined): { total: number; labels: string[] } {
  const outputs = Array.isArray(quest?.outputs) ? quest.outputs : []
  if (outputs.length === 0) return { total: 0, labels: [] }
  const counts = new Map<string, number>()
  for (const output of outputs) {
    const kind = output.kind || 'output'
    counts.set(kind, (counts.get(kind) || 0) + 1)
  }
  return {
    total: outputs.length,
    labels: [...counts.entries()].map(([kind, count]) => `${kind} ${count}`),
  }
}

export function phaseDisplayName(quest: QuestMeta | null | undefined, idx: number): string {
  if (!quest || idx < 0) return i18n.t('quest.phase.unavailable')
  const phases = Array.isArray(quest.phases) ? quest.phases : []
  const pipelineDef = Array.isArray(quest.pipeline_def) ? quest.pipeline_def : []
  const phase = phases[idx]
  const def = pipelineDef[idx]
  return (
    friendlyPhaseName(def) ||
    friendlyPhaseName(phase) ||
    phaseNameFallback(quest, idx)
  )
}

export function questPhaseNodes(quest: QuestMeta | null | undefined): PhaseNode[] {
  const phases = Array.isArray(quest?.phases) ? quest.phases : []
  if (!quest) return []
  if (phases.length === 0) return legacyPhaseNodes(quest)
  const pipelineDef = Array.isArray(quest.pipeline_def) ? quest.pipeline_def : []
  const outputKindsBySource = artifactKindsBySource(quest.outputs)

  return phases.map((phase, idx) => {
    const def = pipelineDef[idx] || phase
    const role = normalizePhaseRole(def.role || phase.role)
    const className = def.class || phase.class || defaultPhaseClass(role)
    const name = def.name || phase.name || `phase_${idx}`
    return {
      phaseIdx: Number(phase.phase_idx ?? def.phase_idx ?? idx) || idx,
      name,
      displayName: phaseDisplayName(quest, idx),
      role,
      class_: className,
      status: phase.status || 'pending',
      turns: Number(phase.turns) || 0,
      reworkCount: Number(phase.rework_count ?? quest.rework_count ?? 0) || 0,
      startedAtMs: phase.started_at_ms,
      endedAtMs: phase.ended_at_ms,
      sessionId: phase.session_id,
      artifactKinds: [
        ...new Set([
          ...artifactKindsForPhase(outputKindsBySource, phase),
          ...artifactKindsForPhase(outputKindsBySource, def),
        ]),
      ],
    }
  })
}

export function getDesignPlan(quest: QuestMeta | null | undefined): QuestArtifact | null {
  if (!quest) return null
  const outputs = Array.isArray(quest.outputs) ? quest.outputs : []
  const byKind = outputs.find((artifact) => artifact.kind === 'design_plan')
  if (byKind) return byKind

  const phases = Array.isArray(quest.phases) ? quest.phases : []
  const pipelineDef = Array.isArray(quest.pipeline_def) ? quest.pipeline_def : []
  const designPhase = phases.find((phase, idx) => {
    const def = pipelineDef[idx]
    const name = phase.name || def?.name
    return name === 'warrior_design' && normalizePhaseRole(def?.role || phase.role) === 'execute'
  })
  if (!designPhase) return null

  return outputs.find((artifact) => {
    const source = String(artifact.source || '')
    return source === designPhase.session_id || source === designPhase.name || source === 'warrior_design'
  }) || null
}

export function groupEventsByPhase(
  events: QuestEvent[],
  phases: PhaseNode[],
  currentIdx = -1,
): GroupedPhaseEvents {
  const byPhase = new Map<number, QuestEvent[]>()
  const bySessionID = new Map<string, number>()
  for (const phase of phases) {
    byPhase.set(phase.phaseIdx, [])
    if (phase.sessionId) bySessionID.set(phase.sessionId, phase.phaseIdx)
  }
  const fallbackIdx = currentIdx >= 0 ? currentIdx : phases.find((phase) => phase.status === 'running')?.phaseIdx
  const unassigned: QuestEvent[] = []

  for (const event of events) {
    const idx = phaseIndexForEvent(event, bySessionID, phases, fallbackIdx)
    if (idx == null) {
      unassigned.push(event)
      continue
    }
    const list = byPhase.get(idx) || []
    list.push(event)
    byPhase.set(idx, list)
  }

  return { byPhase, unassigned }
}

/**
 * Resolve the current phase index, falling back to the first running phase
 * when current_phase_idx is absent — keeps questPhaseView() and questActionState()
 * in lockstep so they never disagree on which phase is active.
 */
function resolveCurrentPhaseIdx(quest: Pick<QuestMeta, 'current_phase_idx' | 'phases'>): number {
  const phases = Array.isArray(quest.phases) ? quest.phases : []
  if (typeof quest.current_phase_idx === 'number') return quest.current_phase_idx
  return phases.findIndex((phase) => phase.status === 'running')
}

export function questPhaseView(quest: QuestMeta | null | undefined): QuestPhaseView {
  const phases = Array.isArray(quest?.phases) ? quest.phases : []
  const currentIdx = resolveCurrentPhaseIdx(quest || ({} as QuestMeta))
  const phase = currentIdx >= 0 ? phases[currentIdx] : undefined
  const count = positiveNumber(quest?.phase_count) || phases.length
  const name = phaseDisplayName(quest, currentIdx)
  const pipelineDef = Array.isArray(quest?.pipeline_def) ? quest.pipeline_def : []
  const def = currentIdx >= 0 ? pipelineDef[currentIdx] : undefined
  const role = normalizePhaseRole(def?.role || phase?.role || 'execute')
  return {
    available: Boolean(quest?.pipeline_version && quest.pipeline_version > 0 && phases.length > 0 && currentIdx >= 0),
    currentIdx,
    count,
    name,
    role,
    status: phase?.status,
    startedAtMs: phase?.started_at_ms,
    reworkCount: Number(phase?.rework_count ?? quest?.rework_count ?? 0) || 0,
    pipelineName: quest?.pipeline_name || 'default',
    pipelineHash: quest?.pipeline_def_hash || 'legacy',
  }
}

export type QuestStatusBadgeInfo = {
  status: QuestStatus
  label: string
  muted: boolean
  title?: string
}

export function questStatusBadgeInfo(quest: QuestMeta | null | undefined): QuestStatusBadgeInfo {
  if (!quest) {
    return { status: 'pending', label: statusLabel('pending') || 'pending', muted: false }
  }
  const status = quest.status || 'pending'
  const label = statusLabel(status) || status
  const phaseView = questPhaseView(quest)
  const phases = Array.isArray(quest.phases) ? quest.phases : []
  const pipelineDef = Array.isArray(quest.pipeline_def) ? quest.pipeline_def : []
  let muted = false
  if (phaseView.available && (status === 'running' || status === 'reviewing')) {
    // Count same-type phases before current to determine shade
    // (running = execute phases, reviewing = review phases)
    const targetRole = status === 'running' ? 'execute' : 'review'
    let sameTypeCount = 0
    for (let i = 0; i < phaseView.currentIdx; i++) {
      const def = pipelineDef[i] || phases[i]
      const role = normalizePhaseRole(def?.role)
      if (role === targetRole) sameTypeCount++
    }
    muted = sameTypeCount > 0
  }
  const title = phaseView.available ? i18n.t('quest.statusBadge.currentPhase', { phase: phaseView.name }) : undefined
  return { status, label, muted, title }
}

export function questReviewCopy(quest: QuestMeta | null | undefined): { title: string; body: string } {
  if (!quest) {
    return { title: i18n.t('quest.reviewCopy.defaultTitle'), body: i18n.t('quest.reviewCopy.defaultBody') }
  }
  const phase = currentPhaseDef(quest)
  const phaseName = phaseDisplayName(quest, typeof quest.current_phase_idx === 'number' ? quest.current_phase_idx : -1)
  const isDesignReview = phase?.name === 'mage_design_review'
  const hasDiff = quest.workspace_diff_pending ?? false
  if (quest.final_verdict === 'pass') {
    if (isDesignReview) {
      return {
        title: i18n.t('quest.reviewCopy.designPassedTitle'),
        body: i18n.t('quest.reviewCopy.designPassedBody'),
      }
    }
    if (hasDiff) {
      return {
        title: i18n.t('quest.reviewCopy.defaultTitle'),
        body: i18n.t('quest.reviewCopy.hasDiffBody'),
      }
    }
    return {
      title: quest.effect_type === 'context_store' ? i18n.t('quest.reviewCopy.knowledgeTitle') : i18n.t('quest.reviewCopy.defaultTitle'),
      body: i18n.t('quest.reviewCopy.noDiffBody'),
    }
  }
  if (quest.final_verdict === 'request_changes') {
    return {
      title: i18n.t('quest.reviewCopy.requestChangesTitle', { phase: phaseName || i18n.t('quest.timeline.phase.review') }),
      body: i18n.t('quest.reviewCopy.requestChangesBody'),
    }
  }
  if (quest.final_verdict === 'reject') {
    return {
      title: i18n.t('quest.reviewCopy.rejectTitle', { phase: phaseName || i18n.t('quest.timeline.phase.review') }),
      body: i18n.t('quest.reviewCopy.rejectBody'),
    }
  }
  return {
    title: i18n.t('quest.reviewCopy.defaultTitle'),
    body: i18n.t('quest.reviewCopy.submittedBody'),
  }
}

export function policyChips(quest: QuestMeta | null | undefined): PolicyChip[] {
  const view = quest?.policy_view
  if (!view) return []
  const chips: PolicyChip[] = []
  if (view.effective_trust_tier) {
    chips.push({
      key: 'trust',
      label: trustTierLabel(String(view.effective_trust_tier)),
      icon: 'shield',
      tone: 'blue',
      title: i18n.t('questSelectors.trustTitle'),
    })
  }
  if (view.safety_floor_reason) {
    chips.push({
      key: 'safety',
      label: i18n.t('questSelectors.safetyFloor'),
      icon: 'shield-alert',
      tone: 'amber',
      title: view.safety_floor_reason,
    })
  }
  if (view.last_policy_decision?.action || view.last_policy_decision?.policy_name) {
    const autoComplete = view.last_policy_decision?.action === 'auto_complete'
    chips.push({
      key: 'policy',
      label: policyActionLabel(view.last_policy_decision.action, view.last_policy_decision.policy_name),
      icon: autoComplete ? 'zap' : 'circle-check',
      tone: autoComplete ? 'blue' : 'muted',
      title: view.last_policy_decision.reason || view.last_policy_decision.policy_name,
    })
  }
  if (view.recovery_state_summary?.status || view.recovery_state_summary?.strategy) {
    chips.push({
      key: 'recovery',
      label: recoveryLabel(view.recovery_state_summary.status, view.recovery_state_summary.attempt, view.recovery_state_summary.max_attempts),
      icon: 'refresh',
      tone: view.recovery_state_summary.status === 'failed' ? 'amber' : 'violet',
      title: view.recovery_state_summary.policy_name || view.recovery_state_summary.last_error,
    })
  }
  return chips
}

export function questActionState(quest: QuestMeta | null | undefined): QuestActionState {
  const applyFailed = hasApplyFailed(quest)
  const hasWorkspaceDiffPending = quest?.workspace_diff_pending ?? false
  const isReadonly = quest?.workspace_mode === 'readonly'

  // Phase runtime fields — refine action authority alongside quest.status
  const phases = Array.isArray(quest?.phases) ? quest.phases : []
  const currentIdx = resolveCurrentPhaseIdx(quest || ({} as QuestMeta))
  const currentPhase = currentIdx >= 0 ? phases[currentIdx] : undefined
  const currentPhaseStatus = currentPhase?.status
  const hasPipeline = Boolean(quest?.pipeline_version && quest.pipeline_version > 0 && phases.length > 0)

  // readonly 模式无隔离工作区，无需 apply（改动直接在基准目录）
  const canApply = !isReadonly && (hasWorkspaceDiffPending || (applyFailed && quest?.status === 'success' && !quest?.applied))
  const canDiscard = !isReadonly && (hasWorkspaceDiffPending || (applyFailed && quest?.status === 'success' && !quest?.applied))
  // canSpawn: design success 隐含 contract — 后端未提供 allowed_actions 前的 best-effort。
  // TODO(PA-BE-2): 后端提供 explicit can_spawn_execute / phase completion authority 后切过去。
  const canSpawn = quest?.type === 'design' && quest?.status === 'success' && !quest?.child_execute_quest_id
  const isQueued = !!quest?.queued
  const isStarting = !!quest?.starting

  // canStart: pending + not queued/starting + no phase already running
  const firstPhaseRunning = phases[0]?.status === 'running'
  const canStart = quest?.status === 'pending' && !isQueued && !isStarting && !firstPhaseRunning

  // canStop: quest is running/reviewing AND a phase is actually executing
  // When pipeline is available, require current phase status === 'running'
  const statusAllowsStop = quest?.status === 'running' || quest?.status === 'reviewing'
  const phaseAllowsStop = hasPipeline ? currentPhaseStatus === 'running' : statusAllowsStop
  const canStop = statusAllowsStop && phaseAllowsStop

  const isBlocked = quest?.status === 'blocked'
  const isEnded = quest?.status === 'success' || quest?.status === 'failed' || quest?.status === 'cancelled'

  // isRunning: align quest status with phase reality.
  // Ended quests must never be "running" even if a phase stale-carries 'running'.
  const anyPhaseRunning = phases.some((p) => p.status === 'running')
  const isRunning = !isEnded && (quest?.status === 'running' || quest?.status === 'reviewing' || isQueued || isStarting || anyPhaseRunning)

  return {
    canReview: quest?.status === 'user_review',
    applyFailed,
    hasWorkspaceDiffPending,
    canApply,
    canDiscard,
    canSpawn,
    canStart,
    canStop,
    isQueued,
    isStarting,
    isBlocked,
    isEnded,
    isRunning,
  }
}

export function reviewActionDraft(verdict: Verdict, hasWorkspaceDiffPending: boolean, apply?: boolean): ReviewActionDraft {
  let label: string
  if (verdict === 'pass') {
    if (!hasWorkspaceDiffPending) {
      label = i18n.t('quest.reviewAction.pass')
    } else if (apply === false) {
      label = i18n.t('quest.reviewAction.passNoApply')
    } else {
      label = i18n.t('quest.reviewAction.passAndApply')
    }
  } else if (verdict === 'reject') {
    label = i18n.t('quest.reviewAction.reject')
  } else {
    label = i18n.t('quest.reviewAction.requestChanges')
  }
  return {
    verdict,
    apply,
    label,
    tone: verdict === 'reject' ? 'danger' : verdict === 'pass' ? 'primary' : 'default',
  }
}

// ─── Timeline helpers ────────────────────────────────────────────────

export function timelineVariantForPhase(phase: PhaseNode): TimelineVariant {
  if (isMagePhase(phase)) return 'mage'
  if (phase.class_ === 'user') return 'user'
  return 'warrior'
}

export function timelineIconForPhase(phase: PhaseNode): string {
  if (phase.status === 'failed') return 'triangle-alert'
  if (isMagePhase(phase)) return 'wand-sparkles'
  if (phase.class_ === 'user') return 'user'
  return 'swords'
}

export function timelineMetaForPhase(phase: PhaseNode): string {
  if (phase.status === 'running') return i18n.t('quest.timeline.meta.running')
  if (phase.status === 'done') return phase.endedAtMs ? fmtTime(phase.endedAtMs) : i18n.t('quest.timeline.meta.done')
  if (phase.status === 'failed') return phase.endedAtMs ? fmtTime(phase.endedAtMs) : i18n.t('quest.timeline.meta.failed')
  if (phase.startedAtMs) return fmtTime(phase.startedAtMs)
  return i18n.t('quest.timeline.meta.pending')
}

export function timelineBodyForPhase(quest: QuestMeta, phase: PhaseNode): string | undefined {
  if (quest.starting && quest.status === 'pending' && phase.phaseIdx === 0) {
    return i18n.t('quest.timeline.body.starting')
  }
  if (quest.queued && quest.status === 'pending' && phase.phaseIdx === 0) {
    const pos = quest.queue_position ? i18n.t('quest.timeline.body.queuedPosition', { pos: quest.queue_position }) : ''
    return i18n.t('quest.timeline.body.queued', { position: pos })
  }
  if (quest.status === 'pending' && phase.phaseIdx === 0) return i18n.t('quest.timeline.body.pendingFirst')
  if (phase.status === 'running') return phase.role === 'review' ? i18n.t('quest.timeline.body.runningReview') : i18n.t('quest.timeline.body.runningExecute')
  if (phase.status === 'failed') return i18n.t('quest.timeline.body.failed')
  if (isMagePhase(phase) && isQuickMode(quest)) return i18n.t('quest.timeline.body.quickSkipMage')
  if (isMagePhase(phase) && !String(quest.mage_id || '').trim()) return i18n.t('quest.timeline.body.noMage')
  if (phase.status === 'pending' && phase.role === 'review') return i18n.t('quest.timeline.body.pendingReview')
  if (phase.status === 'pending') return i18n.t('quest.timeline.body.pending')
  return undefined
}

function phaseChips(quest: QuestMeta, phase: PhaseNode, isLast: boolean): string[] {
  const chips: string[] = []
  if (isMagePhase(phase) && isQuickMode(quest)) chips.push(i18n.t('quest.timeline.chip.skipped'))
  else if (isMagePhase(phase) && !String(quest.mage_id || '').trim()) chips.push(i18n.t('quest.timeline.chip.noMage'))
  if (phase.turns > 0) chips.push(`${phase.turns} turns`)
  if (phase.artifactKinds.length > 0) chips.push(...phase.artifactKinds)
  if (isLast) {
    if (quest.diff_changed_files) chips.push(`${quest.diff_changed_files} files`)
    if (quest.diff_additions) chips.push(`+${quest.diff_additions}`)
    if (quest.diff_deletions) chips.push(`-${quest.diff_deletions}`)
  }
  // 单节点标签收敛至 3 个，避免徽章墙；优先保留语义性标签。
  return chips.slice(0, 3)
}

function eventBody(ev: QuestEvent): string {
  const p = (ev.payload || {}) as Record<string, unknown>
  const title = String(p.title || '')
  const content = String(p.content || '')
  const comment = String(p.comment || '')
  const summary = String(p.summary || '')
  const message = String(p.message || '')
  const note = String(p.note || '')
  if (title && content) return title + '\n' + content
  return title || content || comment || summary || message || note || compactTimelinePayload(p) || ''
}

function compactTimelinePayload(payload: Record<string, unknown>): string {
  const keys = ['verdict', 'status', 'phase', 'phase_idx', 'from', 'reason', 'queue_pos']
  return keys
    .filter((key) => payload[key] != null && payload[key] !== '')
    .map((key) => `${key}: ${String(payload[key])}`)
    .join(' · ')
}

function eventPhaseEntry(ev: QuestEvent, phase: PhaseNode): TimelineEntry {
  return {
    variant: timelineVariantForPhase(phase),
    icon: timelineIconForPhase(phase),
    title: i18n.t('quest.timeline.event.phaseReport', { phase: phase.displayName }),
    meta: ev.ts ? fmtTime(ev.ts) : '',
    body: eventBody(ev),
  }
}

function eventUnassignedEntry(ev: QuestEvent): TimelineEntry {
  const isMage = (ev.session_id || '').startsWith('mage')
  return {
    variant: isMage ? 'mage' : 'warrior',
    icon: isMage ? 'wand-sparkles' : 'swords',
    title: isMage ? i18n.t('quest.timeline.event.mageReport') : i18n.t('quest.timeline.event.warriorReport'),
    meta: ev.ts ? fmtTime(ev.ts) : '',
    body: eventBody(ev),
  }
}

/**
 * 构建时间线完整条目列表。
 * 按顺序：各阶段节点 + 阶段下事件 → 未归属事件 → 备注 → 评论 → 状态节点
 */
export function questTimelineEntries(
  quest: QuestMeta | null | undefined,
  events: QuestEvent[],
  options: { lastCommentInjectedTs?: number; phaseOutputs?: Map<number, string> } = {},
): TimelineEntry[] {
  if (!quest) return []

  const list: TimelineEntry[] = []
  const nodes = questPhaseNodes(quest)
  const agentEvents = events.filter((e) => e.type === 'quest.agent_update' && !!eventBody(e))
  const visibleEvents = events.filter((e) => !!eventBody(e))
  const phaseView = questPhaseView(quest)
  const eventGroups = groupEventsByPhase(agentEvents, nodes, phaseView.currentIdx)
  const visibleEventGroups = groupEventsByPhase(visibleEvents, nodes, phaseView.currentIdx)
  const { lastCommentInjectedTs = 0, phaseOutputs } = options
  const actionState = questActionState(quest)
  const { canApply, isBlocked, hasWorkspaceDiffPending, applyFailed } = actionState

  // 阶段节点 + 阶段下事件
  for (const phase of nodes) {
    const isLast = phase.phaseIdx === nodes.length - 1
    const chips = phaseChips(quest, phase, isLast)
    list.push({
      variant: timelineVariantForPhase(phase),
      icon: timelineIconForPhase(phase),
      title: phase.displayName,
      meta: timelineMetaForPhase(phase),
      body: phaseOutputs?.get(phase.phaseIdx) || firstEventBody(visibleEventGroups.byPhase.get(phase.phaseIdx)) || timelineBodyForPhase(quest, phase),
      status: phase.status === 'failed' ? 'blocked' : '',
      chips: chips.length > 0 ? chips : undefined,
    })
    for (const ev of eventGroups.byPhase.get(phase.phaseIdx) || []) {
      list.push(eventPhaseEntry(ev, phase))
    }
  }

  // 未归属事件
  for (const ev of eventGroups.unassigned) {
    list.push(eventUnassignedEntry(ev))
  }

  // 备注
  if (quest.notes?.length) {
    for (const note of quest.notes) {
      list.push({
        variant: 'note',
        icon: 'file-text',
        title: i18n.t('quest.timeline.note'),
        meta: note.time ? fmtTime(note.time) : '',
        body: note.content || '',
      })
    }
  }

  // 评论
  if (quest.comments?.length) {
    for (const c of quest.comments) {
      const injected = (c.time || 0) > 0 && (c.time || 0) <= lastCommentInjectedTs
      const chips = injected ? [i18n.t('quest.timeline.commentInjected')] : undefined
      list.push({
        variant: 'comment',
        icon: 'user',
        title: c.author === 'user' ? i18n.t('quest.timeline.commentYour') : i18n.t('quest.timeline.commentOther'),
        meta: c.time ? fmtTime(c.time) : '',
        body: c.content || '',
        chips,
      })
    }
  }

  // 用户评审节点
  const hasUserReviewEvent = events.some((event) => event.type === 'quest.user_review')
  if (quest.status === 'user_review' && !applyFailed && !hasUserReviewEvent) {
    const reviewCopy = questReviewCopy(quest)
    list.push({
      variant: 'user',
      icon: 'user',
      title: reviewCopy.title,
      meta: i18n.t('quest.timeline.pendingAction'),
      body: reviewCopy.body,
    })
  }

  // 应用失败
  if (applyFailed) {
    list.push({
      variant: 'system',
      icon: 'triangle-alert',
      title: i18n.t('quest.timeline.applyFailed.title'),
      meta: quest.apply_failed_at_ms ? fmtTime(quest.apply_failed_at_ms) : i18n.t('quest.timeline.needHandle'),
      body: quest.apply_error || i18n.t('quest.timeline.applyFailed.body'),
      status: 'blocked',
    })
  }

  // 成功/完成状态
  if (quest.status === 'success') {
    const isContextStoreQuest = quest.effect_type === 'context_store'
    const isExternalSideEffectQuest = quest.effect_type === 'external_side_effect'

    let title = i18n.t('quest.timeline.success.noChange')
    if (quest.type === 'design') {
      title = i18n.t('quest.timeline.success.designPassed')
    } else if (isContextStoreQuest) {
      title = i18n.t('quest.timeline.success.knowledgeUpdated')
    } else if (isExternalSideEffectQuest) {
      title = i18n.t('quest.timeline.success.externalDone')
    } else if (applyFailed) {
      title = i18n.t('quest.timeline.applyFailed.title')
    } else if (quest.applied) {
      title = i18n.t('quest.timeline.success.applied')
    } else if (hasWorkspaceDiffPending) {
      title = i18n.t('quest.timeline.success.waitingApply')
    }

    let body = i18n.t('quest.timeline.success.bodyNoChange')
    if (quest.type === 'design') {
      body = i18n.t('quest.timeline.success.bodyDesign')
    } else if (isContextStoreQuest) {
      body = i18n.t('quest.timeline.success.bodyKnowledge')
    } else if (isExternalSideEffectQuest) {
      body = i18n.t('quest.timeline.success.bodyExternal')
    } else if (applyFailed) {
      body = i18n.t('quest.timeline.success.bodyApplyFailed')
    } else if (canApply) {
      body = i18n.t('quest.timeline.success.bodyCanApply')
    }

    list.push({
      variant: 'success',
      icon: 'circle-check',
      title,
      meta: quest.completed_at_ms ? fmtTime(quest.completed_at_ms) : quest.started_at_ms ? fmtTime(quest.started_at_ms) : i18n.t('quest.timeline.success.metaRunning'),
      body,
    })
  }

  // 阻塞状态
  if (isBlocked) {
    list.push({
      variant: 'system',
      icon: 'triangle-alert',
      title: i18n.t('quest.timeline.blockedNode.title'),
      meta: i18n.t('quest.timeline.needHandle'),
      body: fmtBlockedReason(quest) || i18n.t('quest.timeline.blockedNode.body'),
      status: 'blocked',
    })
  }

  return list
}

function firstEventBody(events: QuestEvent[] | undefined): string {
  if (!events) return ''
  for (const event of events) {
    const body = eventBody(event)
    if (body) return body
  }
  return ''
}

function positiveNumber(value: unknown): number | undefined {
  const n = Number(value)
  return Number.isFinite(n) && n > 0 ? n : undefined
}

function phaseNameFallback(quest: QuestMeta, idx: number): string {
  const phases = Array.isArray(quest.phases) ? quest.phases : []
  const pipelineDef = Array.isArray(quest.pipeline_def) ? quest.pipeline_def : []
  const def = pipelineDef[idx]
  const phase = phases[idx]
  const role = normalizePhaseRole(def?.role || phase?.role)
  if (quest.status === 'user_review') return i18n.t('quest.timeline.phase.userReview')
  if (role === 'review') return i18n.t('quest.timeline.phase.numberedReview', { idx: idx + 1 })
  if (role === 'execute') return i18n.t('quest.timeline.phase.numberedExecute', { idx: idx + 1 })
  if (idx >= 0) return i18n.t('quest.timeline.phase.numbered', { idx: idx + 1 })
  return i18n.t('quest.phase.unavailable')
}

function friendlyPhaseName(phase: QuestPhaseTask | undefined): string | undefined {
  if (!phase) return undefined
  const displayName = String(phase.display_name || '').trim()
  if (displayName) return phaseNameFromKey(displayName) || displayName
  const name = String(phase.name || '').trim()
  if (!name) return undefined
  return phaseNameFromKey(name) || name
}

function phaseNameFromKey(key: string): string | undefined {
  const map: Record<string, string> = {
    execute: i18n.t('quest.timeline.phase.execute'),
    review: i18n.t('quest.timeline.phase.review'),
    warrior: i18n.t('quest.timeline.phase.warriorExecute'),
    mage: i18n.t('quest.timeline.phase.mageReview'),
    warrior_execute: i18n.t('quest.timeline.phase.execute'),
    warrior_design: i18n.t('quest.timeline.phase.warriorDesign'),
    mage_review: i18n.t('quest.timeline.phase.mageImplementationReview'),
    mage_design_review: i18n.t('quest.timeline.phase.mageDesignReview'),
    mage_implementation_review: i18n.t('quest.timeline.phase.mageImplementationReview'),
    user_review: i18n.t('quest.timeline.phase.userReview'),
  }
  return map[key]
}

function normalizePhaseRole(role: string | undefined): string {
  if (role === 'review' || role === 'execute') return role
  return role || 'execute'
}

function defaultPhaseClass(role: string): string {
  return role === 'review' ? 'mage' : 'warrior'
}

function artifactKindsBySource(outputs: QuestArtifact[] | undefined): Map<string, string[]> {
  const out = new Map<string, string[]>()
  if (!Array.isArray(outputs)) return out
  for (const artifact of outputs) {
    const source = String(artifact.source || '').trim()
    if (!source || !artifact.kind) continue
    const list = out.get(source) || []
    list.push(artifact.kind)
    out.set(source, list)
  }
  return out
}

function artifactKindsForPhase(kindsBySource: Map<string, string[]>, phase: QuestPhaseTask | undefined): string[] {
  if (!phase) return []
  const keys = [phase.session_id, phase.name].filter(Boolean) as string[]
  return keys.flatMap((key) => kindsBySource.get(key) || [])
}

function phaseIndexForEvent(
  event: QuestEvent,
  bySessionID: Map<string, number>,
  phases: PhaseNode[],
  fallbackIdx: number | undefined,
): number | null {
  if (event.session_id && bySessionID.has(event.session_id)) return bySessionID.get(event.session_id) ?? null
  const payload = event.payload as Record<string, unknown> | undefined
  const rawPhaseIdx = eventPhaseIndex(payload)
  if (rawPhaseIdx != null && phases.some((phase) => phase.phaseIdx === rawPhaseIdx)) return rawPhaseIdx
  if (event.session_id) {
    const legacyIdx = legacyPhaseIndexFromSession(event.session_id, phases)
    if (legacyIdx != null) return legacyIdx
  }
  return fallbackIdx != null && phases.some((phase) => phase.phaseIdx === fallbackIdx) ? fallbackIdx : null
}

function eventPhaseIndex(payload: Record<string, unknown> | undefined): number | null {
  const raw = payload?.phase_idx ?? payload?.phase
  const n = Number(raw)
  return Number.isFinite(n) && n >= 0 ? n : null
}

function legacyPhaseIndexFromSession(sessionID: string, phases: PhaseNode[]): number | null {
  if (sessionID.startsWith('warrior')) return phases.find((phase) => phase.class_ === 'warrior')?.phaseIdx ?? null
  if (sessionID.startsWith('mage')) return phases.find((phase) => phase.class_ === 'mage')?.phaseIdx ?? null
  return null
}

function currentPhaseDef(quest: QuestMeta): QuestPhaseTask | undefined {
  const idx = typeof quest.current_phase_idx === 'number' ? quest.current_phase_idx : -1
  if (idx < 0) return undefined
  const phases = Array.isArray(quest.phases) ? quest.phases : []
  const pipelineDef = Array.isArray(quest.pipeline_def) ? quest.pipeline_def : []
  return pipelineDef[idx] || phases[idx]
}

function legacyPhaseNodes(quest: QuestMeta): PhaseNode[] {
  const executeStatus: PhaseStatus =
    quest.status === 'pending'
      ? 'pending'
      : quest.status === 'running' || quest.status === 'waiting_input' || quest.status === 'blocked'
        ? 'running'
        : quest.status === 'failed' || quest.status === 'cancelled'
          ? 'failed'
          : 'done'
  const nodes: PhaseNode[] = [{
    phaseIdx: 0,
    name: 'warrior',
    displayName: i18n.t('quest.timeline.phase.warriorExecute'),
    role: 'execute',
    class_: 'warrior',
    status: executeStatus,
    turns: questResourceUsage(quest).turns,
    reworkCount: Number(quest.rework_count) || 0,
    startedAtMs: quest.started_at_ms,
    endedAtMs: executeStatus === 'done' ? quest.completed_at_ms : undefined,
    artifactKinds: Array.isArray(quest.outputs) ? [...new Set(quest.outputs.map((out) => out.kind).filter(Boolean))] : [],
  }]

  if (quest.intensity !== 'quick' && quest.status !== 'pending' && quest.status !== 'running') {
    nodes.push({
      phaseIdx: 1,
      name: 'mage_review',
      displayName: i18n.t('quest.timeline.phase.mageReview'),
      role: 'review',
      class_: 'mage',
      status: quest.status === 'reviewing' ? 'running' : quest.status === 'failed' ? 'failed' : 'done',
      turns: 0,
      reworkCount: Number(quest.rework_count) || 0,
      startedAtMs: quest.started_at_ms,
      endedAtMs: quest.status === 'reviewing' ? undefined : quest.completed_at_ms,
      artifactKinds: [],
    })
  }
  return nodes
}

function trustTierLabel(tier: string): string {
  const map: Record<string, string> = {
    tier_0: i18n.t('quest.policy.trust.tier_0'),
    tier_1: i18n.t('quest.policy.trust.tier_1'),
    tier_2: i18n.t('quest.policy.trust.tier_2'),
    tier_3: i18n.t('quest.policy.trust.tier_3'),
  }
  return map[tier] || tier
}

function policyActionLabel(action?: string, policyName?: string): string {
  const map: Record<string, string> = {
    auto_pass: i18n.t('quest.policy.action.auto_pass'),
    auto_complete: i18n.t('quest.policy.action.auto_complete'),
    require_user: i18n.t('quest.policy.action.require_user'),
    retry: i18n.t('quest.policy.action.retry'),
  }
  return map[action || ''] || policyName || action || i18n.t('quest.policy.action.default')
}

function recoveryLabel(status?: string, attempt?: number, maxAttempts?: number): string {
  const suffix = maxAttempts ? ` ${attempt || 0}/${maxAttempts}` : ''
  const map: Record<string, string> = {
    pending: i18n.t('quest.policy.recovery.waiting'),
    waiting: i18n.t('quest.policy.recovery.waiting'),
    running: i18n.t('quest.policy.recovery.running'),
    active: i18n.t('quest.policy.recovery.running'),
    succeeded: i18n.t('quest.policy.recovery.succeeded'),
    failed: i18n.t('quest.policy.recovery.failed'),
    exhausted: i18n.t('quest.policy.recovery.failed'),
  }
  return (map[status || ''] || (status || i18n.t('quest.policy.recovery.default'))) + suffix
}

// ── Attention & Progress (P1a) ──────────────────────────────────────────────

export type QuestAttentionKind =
  | 'apply_failed'
  | 'blocked'
  | 'waiting_input'
  | 'runtime_warning'
  | 'review'
  | 'none'

export type QuestAttentionSource = 'apply' | 'status' | 'runtime' | 'review' | 'human_exception'

export type QuestAttention = {
  kind: QuestAttentionKind
  source: QuestAttentionSource
  priority: number
}

/**
 * Derive the attention level for a quest.
 * Priority (lower = more urgent): apply_failed > blocked > waiting_input > runtime_warning > review > none.
 * Composes getStuckDiagnosis() for runtime_warning detection — does not duplicate runtime logic.
 */
export function deriveQuestAttention(quest: QuestMeta | null | undefined): QuestAttention {
  if (!quest) return { kind: 'none', source: 'status', priority: 999 }
  const actionState = questActionState(quest)
  if (actionState.applyFailed) return { kind: 'apply_failed', source: 'apply', priority: 1 }
  const humanBlocked = quest.human_exception?.source_status === 'blocked'
  if (actionState.isBlocked || humanBlocked) return { kind: 'blocked', source: humanBlocked ? 'human_exception' : 'status', priority: 2 }
  if (quest.status === 'waiting_input') return { kind: 'waiting_input', source: 'status', priority: 3 }
  if (getStuckDiagnosis(quest)?.source === 'runtime') return { kind: 'runtime_warning', source: 'runtime', priority: 4 }
  if (actionState.canReview || quest.status === 'user_review') return { kind: 'review', source: 'review', priority: 5 }
  return { kind: 'none', source: 'status', priority: 999 }
}

export type QuestProgressLine = {
  i18nKey: string
  params?: Record<string, unknown>
}

/**
 * Derive the progress line for a quest. Returns i18n key + params, not display text.
 * Note: server-provided human_exception.recommended_action is NOT handled here;
 * callers should check that first and use it as an override.
 */
export function deriveQuestProgressLine(quest: QuestMeta | null | undefined): QuestProgressLine {
  if (!quest) return { i18nKey: 'quest.progress.default' }
  const actionState = questActionState(quest)
  const { canReview, applyFailed } = actionState
  // Priority must align with deriveQuestAttention() — same quest must tell the same story.
  const attention = deriveQuestAttention(quest)
  if (attention.kind === 'apply_failed') return { i18nKey: 'quest.progress.applyFailed' }
  if (attention.kind === 'blocked') return { i18nKey: 'quest.progress.blocked' }
  if (attention.kind === 'waiting_input') return { i18nKey: 'quest.progress.waitingInput' }
  if (attention.kind === 'review') return { i18nKey: 'quest.progress.waitingReview' }
  if (quest.status === 'reviewing') return { i18nKey: 'quest.progress.mageReviewing' }
  const isQuick = String(quest.intensity || '').toLowerCase() === 'quick'
  if (quest.status === 'running') return { i18nKey: isQuick ? 'quest.progress.warriorRunningQuick' : 'quest.progress.warriorRunning' }
  if (quest.status === 'pending') {
    if (quest.queued) {
      return quest.queue_position
        ? { i18nKey: 'quest.progress.queuedPosition', params: { position: quest.queue_position } }
        : { i18nKey: 'quest.progress.queued' }
    }
    if (quest.starting) return { i18nKey: 'quest.progress.starting' }
    return { i18nKey: 'quest.progress.pending' }
  }
  if (quest.status === 'success') return { i18nKey: 'quest.progress.success' }
  if (quest.status === 'failed') return { i18nKey: 'quest.progress.failed' }
  if (quest.status === 'cancelled') return { i18nKey: 'quest.progress.cancelled' }
  return { i18nKey: 'quest.progress.default' }
}
