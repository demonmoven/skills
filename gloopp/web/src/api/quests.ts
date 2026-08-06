import { api } from './client'
import type { AgentSessionState, ApplyBackupMeta, DesignDoc, DiffResult, FanoutGroup, LoopStateSpine, QuestCheck, QuestMeta, QuestReports, QuestReview, SafetyWarning, ThreadPost, ThreadViewResponse, Verdict } from './types'
import type { QuestTraceItem } from '../features/quest/traceAdapters'

export type QuestDetailResponse = {
  quest: QuestMeta
  design?: DesignDoc
  checks?: QuestCheck[]
  reviews?: QuestReview[]
  reports?: QuestReports
  thread_posts?: ThreadPost[]
  fanout_group?: FanoutGroup
  loop_state_spine?: LoopStateSpine
  agent_sessions?: AgentSessionState[]
}

export type ResolveQuestPayload = {
  verdict: Verdict
  comment: string
  apply?: boolean
}

export type ResolveQuestResponse = {
  ok: boolean
  qid: string
  verdict: Verdict
  applied?: boolean
  warnings?: SafetyWarning[]
  quest?: QuestMeta
}

export type RecoverAgentPayload = {
  action: string
  add_turns: number
  add_duration_minutes: number
}

export type WorkflowModeUpgradePayload = {
  workflow_mode: 'direct' | 'checked' | 'goal'
  reason?: string
}

export type AnswerQuestPayload = {
  question_id?: string
  answer: string
  source?: string
}

export type AnswerQuestResponse = {
  ok: boolean
  qid: string
  answer?: unknown
}

export function getQuestDetail(qid: string): Promise<QuestDetailResponse> {
  return api.get<QuestDetailResponse>('/api/quests/' + qid)
}

export function listQuestBackups(qid: string): Promise<{ items: ApplyBackupMeta[] }> {
  return api.get<{ items: ApplyBackupMeta[] }>('/api/quests/' + qid + '/backups')
}

export function getQuestBackup(qid: string, backupID: string): Promise<{ item: ApplyBackupMeta }> {
  return api.get<{ item: ApplyBackupMeta }>('/api/quests/' + qid + '/backups/' + encodeURIComponent(backupID))
}

export function resolveQuest(qid: string, payload: ResolveQuestPayload): Promise<ResolveQuestResponse> {
  return api.post<ResolveQuestResponse>('/api/quests/' + qid + '/resolve', payload)
}

export function applyQuestChanges(qid: string, force = false): Promise<unknown> {
  return api.post('/api/quests/' + qid + '/apply', force ? { force: true } : {})
}

export function discardQuestChanges(qid: string): Promise<unknown> {
  return api.post('/api/quests/' + qid + '/discard', { reason: 'discarded from dashboard' })
}

export function commentOnQuest(qid: string, comment: string, parentReplyId?: string): Promise<unknown> {
  const payload: Record<string, string> = { comment }
  if (parentReplyId) {
    payload.parent_reply_id = parentReplyId
  }
  return api.post('/api/quests/' + qid + '/comment', payload)
}

export function answerQuest(qid: string, payload: AnswerQuestPayload): Promise<AnswerQuestResponse> {
  return api.post<AnswerQuestResponse>('/api/quests/' + qid + '/answer', {
    ...payload,
    source: payload.source || 'dashboard',
  })
}

export function getQuestDiff(qid: string): Promise<{ diff: DiffResult }> {
  return api.get<{ diff: DiffResult }>('/api/quests/' + qid + '/diff')
}

export function recoverQuestAgent(qid: string, payload: RecoverAgentPayload): Promise<unknown> {
  return api.post('/api/quests/' + qid + '/recover-agent', payload)
}

export function getQuestTrace(qid: string): Promise<{ items: QuestTraceItem[] }> {
  return api.get<{ items: QuestTraceItem[] }>('/api/quests/' + qid + '/trace')
}

export function spawnExecuteFromDesignQuest(qid: string): Promise<{ quest?: QuestMeta; qid?: string }> {
  return api.post<{ quest?: QuestMeta; qid?: string }>('/api/quests/' + qid + '/spawn-execute', { auto_start: false })
}

export function startQuest(qid: string): Promise<{ ok: boolean; qid: string; quest?: QuestMeta }> {
  return api.post<{ ok: boolean; qid: string; quest?: QuestMeta }>('/api/quests/' + qid + '/start')
}

export function stopQuest(qid: string): Promise<unknown> {
  return api.post('/api/quests/' + qid + '/stop', { reason: 'stopped from dashboard' })
}

export function resolveBlockedQuest(qid: string, payload: RecoverAgentPayload): Promise<unknown> {
  return api.post('/api/quests/' + qid + '/resolve-blocked', payload)
}

export function upgradeQuestWorkflowMode(qid: string, payload: WorkflowModeUpgradePayload): Promise<{ ok: boolean; qid: string; quest?: QuestMeta }> {
  return api.post<{ ok: boolean; qid: string; quest?: QuestMeta }>('/api/quests/' + qid + '/workflow-mode', payload)
}

export function cloneQuest(quest: QuestMeta): Promise<unknown> {
  return api.post('/api/quests', {
    query: quest.query,
    type: quest.type,
    inputs: quest.inputs || [],
    source_quest_id: quest.id,
    auto_start: false,
  })
}

export function getQuestThread(qid: string): Promise<ThreadViewResponse> {
  return api.get<ThreadViewResponse>('/api/quests/' + qid + '/thread')
}
