import type { AgentSessionState, QuestMeta } from '../../api/types'
import i18n from '../../i18n'

export type StuckDiagnosisKind =
  | 'blocked'
  | 'waiting_input'
  | 'apply_failed'
  | 'runtime.idle_warning'
  | 'runtime.protocol_warning'

export type StuckDiagnosisSource = 'human_exception' | 'runtime'
type RuntimeWarningKind = Extract<StuckDiagnosisKind, 'runtime.idle_warning' | 'runtime.protocol_warning'>
type HumanExceptionKind = Extract<StuckDiagnosisKind, 'blocked' | 'waiting_input' | 'apply_failed'>

export type StuckDiagnosis = {
  kind: StuckDiagnosisKind
  source: StuckDiagnosisSource
  title: string
  reason: string
  recommendedAction: string
  riskLevel: string
  availableActions: string[]
  evidenceSummary?: string
  auditRef?: string
  sourceStatus?: string
  sessionId?: string
  runtimeHealth?: string
  hasRecommendedAction: boolean
  hasEvidenceJump: boolean
}

export type StuckDiagnosisEntrySurface = 'feed' | 'inbox' | 'quest_detail'
export type StuckDiagnosisResolvedSurface = 'quest_detail' | 'trace' | 'evidence' | 'action'

export type StuckDiagnosisPathEvent = {
  entry_surface: StuckDiagnosisEntrySurface
  quest_id: string
  exception_kind: StuckDiagnosisKind
  click_count: number
  resolved_surface: StuckDiagnosisResolvedSurface
  has_recommended_action: boolean
  has_evidence_jump: boolean
}

type StorageLike = Pick<Storage, 'getItem' | 'setItem'>

const STORAGE_KEY = 'gloop.ui.stuck_diagnosis_path.v1'
const MAX_STORED_EVENTS = 200

const RUNTIME_WARNING_HEALTH: Record<string, RuntimeWarningKind | undefined> = {
  idle: 'runtime.idle_warning',
  protocol_risk: 'runtime.protocol_warning',
}

function kindTitle(kind: StuckDiagnosisKind): string {
  const map: Record<StuckDiagnosisKind, string> = {
    blocked: i18n.t('quest.diagnosis.kind.blocked'),
    waiting_input: i18n.t('quest.diagnosis.kind.waiting_input'),
    apply_failed: i18n.t('quest.diagnosis.kind.apply_failed'),
    'runtime.idle_warning': i18n.t('quest.diagnosis.kind.idleWarning'),
    'runtime.protocol_warning': i18n.t('quest.diagnosis.kind.protocolWarning'),
  }
  return map[kind]
}

function runtimeReason(kind: RuntimeWarningKind): string {
  const map: Record<RuntimeWarningKind, string> = {
    'runtime.idle_warning': i18n.t('quest.diagnosis.runtimeReason.idleWarning'),
    'runtime.protocol_warning': i18n.t('quest.diagnosis.runtimeReason.protocolWarning'),
  }
  return map[kind]
}

function runtimeAction(kind: RuntimeWarningKind): string {
  const map: Record<RuntimeWarningKind, string> = {
    'runtime.idle_warning': i18n.t('quest.diagnosis.runtimeAction.idleWarning'),
    'runtime.protocol_warning': i18n.t('quest.diagnosis.runtimeAction.protocolWarning'),
  }
  return map[kind]
}

function latestRuntimeWarningSession(quest: QuestMeta): { session: AgentSessionState; kind: RuntimeWarningKind } | null {
  const sessions = Array.isArray(quest.agent_sessions) ? quest.agent_sessions : []
  let latest: { session: AgentSessionState; kind: RuntimeWarningKind } | null = null
  for (const session of sessions) {
    const kind = RUNTIME_WARNING_HEALTH[String(session.health_status || '')]
    if (!kind) continue
    if (!latest || (session.updated_at_ms || 0) > (latest.session.updated_at_ms || 0)) {
      latest = { session, kind }
    }
  }
  return latest
}

export function humanExceptionSourceKind(sourceStatus: string | undefined): HumanExceptionKind | null {
  if (sourceStatus === 'apply_failed' || sourceStatus === 'blocked' || sourceStatus === 'waiting_input') {
    return sourceStatus
  }
  return null
}

function humanExceptionKind(quest: QuestMeta): HumanExceptionKind | null {
  return humanExceptionSourceKind(quest.human_exception?.source_status)
}

export function getStuckDiagnosis(quest: QuestMeta | null | undefined): StuckDiagnosis | null {
  if (!quest) return null
  const humanException = quest.human_exception
  const kind = humanExceptionKind(quest)
  if (kind && humanException) {
    return {
      kind,
      source: 'human_exception',
      title: kindTitle(kind),
      reason: humanException.reason || kindTitle(kind),
      recommendedAction: humanException.recommended_action || fallbackAction(quest, kind),
      riskLevel: humanException.risk_level || (kind === 'waiting_input' ? 'medium' : 'high'),
      availableActions: Array.isArray(humanException.available_actions) ? humanException.available_actions : [],
      evidenceSummary: humanException.evidence_summary,
      auditRef: humanException.audit_ref,
      sourceStatus: humanException.source_status,
      hasRecommendedAction: Boolean(humanException.recommended_action || fallbackAction(quest, kind)),
      hasEvidenceJump: Boolean(humanException.evidence_summary || humanException.audit_ref),
    }
  }
  const runtimeWarning = latestRuntimeWarningSession(quest)
  if (!runtimeWarning) return null
  const { session, kind: runtimeKind } = runtimeWarning
  return {
    kind: runtimeKind,
    source: 'runtime',
    title: kindTitle(runtimeKind),
    reason: runtimeReason(runtimeKind),
    recommendedAction: runtimeAction(runtimeKind),
    riskLevel: 'medium',
    availableActions: ['open_trace'],
    evidenceSummary: [
      session.session_id ? `session ${session.session_id}` : '',
      session.health_status ? `health ${session.health_status}` : '',
    ].filter(Boolean).join(' · ') || undefined,
    sourceStatus: session.health_status,
    sessionId: session.session_id,
    runtimeHealth: session.health_status,
    hasRecommendedAction: true,
    hasEvidenceJump: true,
  }
}

export function compareStuckDiagnoses(a: QuestMeta, b: QuestMeta): number {
  const ad = getStuckDiagnosis(a)
  const bd = getStuckDiagnosis(b)
  const ah = typeof a.human_exception?.priority === 'number' ? a.human_exception.priority : 999
  const bh = typeof b.human_exception?.priority === 'number' ? b.human_exception.priority : 999
  if (ah !== bh) return ah - bh
  const ap = diagnosisPriority(ad)
  const bp = diagnosisPriority(bd)
  if (ap !== bp) return ap - bp
  const at = Number(a.updated_at_ms ?? a.created_at_ms) || 0
  const bt = Number(b.updated_at_ms ?? b.created_at_ms) || 0
  return bt - at
}

export function emitStuckDiagnosisPath(event: StuckDiagnosisPathEvent, storage: StorageLike | null = browserStorage()): boolean {
  if (!storage || !event.quest_id || !event.exception_kind) return false
  const row = {
    event: 'ui.stuck_diagnosis_path',
    ts: Date.now(),
    ...event,
  }
  try {
    const current = JSON.parse(storage.getItem(STORAGE_KEY) || '[]')
    const rows = Array.isArray(current) ? current : []
    rows.push(row)
    storage.setItem(STORAGE_KEY, JSON.stringify(rows.slice(-MAX_STORED_EVENTS)))
    return true
  } catch {
    return false
  }
}

function diagnosisPriority(diagnosis: StuckDiagnosis | null): number {
  if (!diagnosis) return 99
  if (diagnosis.source === 'human_exception' && diagnosis.riskLevel === 'high') return 0
  if (diagnosis.kind === 'apply_failed') return 1
  if (diagnosis.kind === 'blocked') return 2
  if (diagnosis.kind === 'waiting_input') return 3
  if (diagnosis.kind === 'runtime.protocol_warning') return 4
  if (diagnosis.kind === 'runtime.idle_warning') return 5
  return 9
}

function fallbackAction(quest: QuestMeta, kind: StuckDiagnosisKind): string {
  if (kind === 'blocked') return i18n.t('quest.diagnosis.fallbackAction.blocked')
  if (kind === 'waiting_input') return i18n.t('quest.diagnosis.fallbackAction.waitingInput')
  if (kind === 'apply_failed') return i18n.t('quest.diagnosis.fallbackAction.applyFailed')
  if (kind === 'runtime.idle_warning' || kind === 'runtime.protocol_warning') return runtimeAction(kind)
  return i18n.t('quest.diagnosis.fallbackAction.default')
}

function browserStorage(): StorageLike | null {
  if (typeof window === 'undefined') return null
  try {
    return window.localStorage
  } catch {
    return null
  }
}
