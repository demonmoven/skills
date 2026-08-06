import type { HumanExceptionItem, QuestMeta } from '../api/types'
import i18n from '../i18n'

export type AutomationCandidateMeta = {
  source: string
  triageMode: string
  holdReason: string
}

export function isAutomationCandidate(item: QuestMeta): boolean {
  return String(item.created_by || '').startsWith('automation:')
}

export function automationCandidateMeta(item: QuestMeta): AutomationCandidateMeta {
  const rawSource = typeof item.candidate_source === 'string' && item.candidate_source.trim()
    ? item.candidate_source
    : String(item.created_by || '')
  const source = rawSource.startsWith('automation:')
    ? rawSource.slice('automation:'.length) || 'automation'
    : rawSource || 'automation'
  const triageMode = typeof item.triage_mode === 'string' && item.triage_mode.trim()
    ? item.triage_mode
    : 'candidate'
  const holdReason = triageMode === 'candidate'
    ? i18n.t('inbox.automationCandidate.holdReason.candidate')
    : i18n.t('inbox.automationCandidate.holdReason.direct')
  return { source, triageMode, holdReason }
}

export function splitInboxWorkQueues(items: QuestMeta[], humanExceptions: HumanExceptionItem[] = []): {
  automationCandidates: QuestMeta[]
  pendingDelegations: QuestMeta[]
} {
  const exceptionQuestIds = new Set(humanExceptions.map((item) => item.quest_id).filter(Boolean))
  const visible = items.filter((item) => !exceptionQuestIds.has(item.id))
  return {
    automationCandidates: visible.filter(isAutomationCandidate),
    pendingDelegations: visible.filter((item) => !isAutomationCandidate(item)),
  }
}
