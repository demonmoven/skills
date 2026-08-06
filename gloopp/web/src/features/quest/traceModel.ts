import type { ContextPackData, QuestEvent, QuestSessionRow } from '../../api/types'
import i18n from '../../i18n'

export type TraceFilter = 'all' | 'tools' | 'decisions' | 'errors' | 'messages' | 'comments' | 'context'

export interface NativeToolItem {
  sid: string
  row: QuestSessionRow
  status: string
  result: string
  rawEvent: string
}

export interface MessageItem {
  sid: string
  content: string
  turn: number
  phase: number
  ts: number
  meta?: Record<string, unknown>
}

export interface CommentItem {
  sid: string
  content: string
  turn: number
  phase: number
  ts: number
  count: number
}

export interface ContextPackItem {
  sid: string
  ts: number
  turn: number
  phase: number
  data: ContextPackData
}

export interface ToolResultItem {
  sid: string
  ts: number
  turn: number
  phase: number
  toolName: string
  content: string
  meta?: Record<string, unknown>
}

export type UnifiedEntry = {
  kind: 'event'
  event: QuestEvent
  ts: number
} | {
  kind: 'tool'
  tool: NativeToolItem
  ts: number
} | {
  kind: 'message'
  message: MessageItem
  ts: number
} | {
  kind: 'comment'
  comment: CommentItem
  ts: number
} | {
  kind: 'context_pack'
  contextPack: ContextPackItem
  ts: number
} | {
  kind: 'tool_result'
  toolResult: ToolResultItem
  ts: number
}

export type GroupedEntry = {
  kind: 'event'
  event: QuestEvent
  ts: number
} | {
  kind: 'tool'
  tool: NativeToolItem
  ts: number
  group?: NativeToolItem[]
} | {
  kind: 'message'
  message: MessageItem
  ts: number
} | {
  kind: 'comment'
  comment: CommentItem
  ts: number
} | {
  kind: 'context_pack'
  contextPack: ContextPackItem
  ts: number
} | {
  kind: 'tool_result'
  toolResult: ToolResultItem
  ts: number
}

export function tryParseJSON(value: string): Record<string, unknown> | null {
  if (!value) return null
  try {
    const parsed = JSON.parse(value)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) return parsed
  } catch { /* ignore */ }
  return null
}

export function extractToolTarget(argsStr: string): string | null {
  const obj = tryParseJSON(argsStr)
  if (!obj) return null
  if (typeof obj.file_path === 'string') {
    const parts = obj.file_path.split('/')
    return parts[parts.length - 1]
  }
  if (typeof obj.pattern === 'string') return obj.pattern.slice(0, 40)
  if (typeof obj.command === 'string') {
    const short = obj.command.replace(/^\/bin\/bash\s+-c\s+/, '').slice(0, 60)
    return short
  }
  if (Array.isArray(obj.command)) return (obj.command as string[]).join(' ').slice(0, 60)
  return null
}

export function extractFullCommand(argsStr: string): string | null {
  const obj = tryParseJSON(argsStr)
  if (!obj) return null
  if (typeof obj.command === 'string') return obj.command
  if (Array.isArray(obj.command)) return (obj.command as string[]).join(' ')
  if (typeof obj.file_path === 'string') return obj.file_path
  return null
}

export function extractToolResult(resultStr: string): string | null {
  const obj = tryParseJSON(resultStr)
  if (!obj) return null
  if (typeof obj.aggregated_output === 'string') return obj.aggregated_output
  if (typeof obj.output === 'string') return obj.output
  if (typeof obj.content === 'string') return obj.content
  return null
}

export function resultSummary(resultStr: string): string | null {
  const text = extractToolResult(resultStr)
  if (!text) return null
  const lines = text.split('\n').filter(Boolean)
  if (lines.length === 0) return null
  if (lines.length === 1) return text.slice(0, 100)
  return i18n.t('traceModel.resultLines', { count: lines.length })
}

export function toolCategory(name: string): { label: string; variant: string } {
  if (name.includes('Bash') || name.includes('bash') || name.includes('command')) return { label: i18n.t('traceModel.tool.cmd'), variant: 'cmd' }
  if (name.includes('Read') || name.includes('read') || name.includes('cat')) return { label: i18n.t('traceModel.tool.read'), variant: 'read' }
  if (name.includes('Write') || name.includes('write') || name.includes('Edit') || name.includes('edit')) return { label: i18n.t('traceModel.tool.write'), variant: 'write' }
  if (name.includes('Search') || name.includes('search') || name.includes('Grep') || name.includes('grep') || name.includes('rg')) return { label: i18n.t('traceModel.tool.search'), variant: 'search' }
  if (name.includes('Glob') || name.includes('glob') || name.includes('find')) return { label: i18n.t('traceModel.tool.search'), variant: 'search' }
  return { label: i18n.t('traceModel.tool.other'), variant: 'other' }
}

export function eventCategory(type: string): { label: string; variant: string } {
  if (type === 'quest.agent_update') return { label: i18n.t('traceModel.eventCat.update'), variant: 'update' }
  if (type.includes('phase')) return { label: i18n.t('traceModel.eventCat.phase'), variant: 'phase' }
  if (type.includes('review') || type.includes('verdict')) return { label: i18n.t('traceModel.eventCat.decision'), variant: 'decision' }
  if (type === 'quest.note') return { label: i18n.t('traceModel.eventCat.note'), variant: 'note' }
  if (type === 'failure.attributed' || type.includes('failed') || type.includes('blocked')) return { label: i18n.t('traceModel.eventCat.error'), variant: 'error' }
  return { label: i18n.t('traceModel.eventCat.default'), variant: 'default' }
}

export function isSemanticEvent(event: QuestEvent): boolean {
  if (event.type.startsWith('micro.')) return false
  if (event.type === 'heartbeat' || event.type === 'ping') return false
  return true
}

export function isErrorEvent(event: QuestEvent): boolean {
  return event.type === 'failure.attributed' ||
    event.type.includes('failed') ||
    event.type.includes('blocked') ||
    event.type.includes('error')
}

export function isDecisionEvent(event: QuestEvent): boolean {
  return event.type.includes('phase') ||
    event.type.includes('review') ||
    event.type === 'quest.agent_update' ||
    event.type === 'failure.attributed' ||
    event.type.includes('verdict')
}

/** 状态切换事件：quest 生命周期的关键状态迁移。与 isDecisionEvent/isErrorEvent 取并集构成"关键节点"。 */
export function isStateChangeEvent(event: QuestEvent): boolean {
  return event.type === 'quest.started' ||
    event.type === 'quest.failed' ||
    event.type === 'quest.cancelled'
}

/** 关键节点判定：决策事件 ∪ 错误事件 ∪ 状态切换事件。isSemanticEvent 定义过宽，不纳入。 */
export function isKeyNodeEvent(event: QuestEvent): boolean {
  return isDecisionEvent(event) || isErrorEvent(event) || isStateChangeEvent(event)
}

/**
 * 从事件流中选取关键节点，按时间倒序取最近 limit 个，返回值按时间正序排列。
 * 用于详情页默认摘要轨迹层；原始全量轨迹留在调试层。
 */
export function selectKeyEvents(events: QuestEvent[], limit = 5): QuestEvent[] {
  const key = events.filter(isKeyNodeEvent)
  if (key.length === 0) return []
  const sorted = key.slice().sort((a, b) => eventTs(a) - eventTs(b))
  return sorted.slice(-limit)
}

function eventTs(event: QuestEvent): number {
  return event.ts || event.timestamp || 0
}

export function eventLabel(type: string): string {
  if (type === 'quest.agent_update') return i18n.t('traceModel.eventLabel.agentUpdate')
  if (type === 'phase.started') return i18n.t('traceModel.eventLabel.phaseStarted')
  if (type === 'phase.completed') return i18n.t('traceModel.eventLabel.phaseCompleted')
  if (type === 'quest.note') return i18n.t('traceModel.eventLabel.note')
  if (type === 'failure.attributed') return i18n.t('traceModel.eventLabel.failureAttributed')
  if (type.includes('review')) return i18n.t('traceModel.eventLabel.review')
  if (type.includes('blocked')) return i18n.t('traceModel.eventLabel.blocked')
  if (type.includes('verdict')) return i18n.t('traceModel.eventLabel.verdict')
  return type
}

export function compactPayload(payload: unknown): string | null {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) return null
  const obj = payload as Record<string, unknown>
  const entries = Object.entries(obj).filter(([, v]) => v != null && v !== '')
  if (entries.length === 0) return null
  if (entries.length > 4) return null
  if (!entries.every(([, v]) => typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean')) return null
  return entries.map(([k, v]) => `${k}: ${v}`).join(' · ')
}

export function payloadAsGrid(payload: unknown): Array<{ key: string; value: string }> | null {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) return null
  const obj = payload as Record<string, unknown>
  const entries = Object.entries(obj).filter(([, v]) => v != null && v !== '')
  if (entries.length === 0) return null
  if (entries.length > 9) return null
  return entries
    .filter(([, v]) => typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean')
    .map(([k, v]) => ({ key: k, value: String(v) }))
}

function toolGroupKey(tool: NativeToolItem): string {
  const name = tool.row.tool_name || 'tool'
  const target = tool.row.tool_args ? extractToolTarget(tool.row.tool_args) : null
  return `${name}::${target || ''}`
}

export function groupConsecutiveTools(entries: UnifiedEntry[]): GroupedEntry[] {
  const result: GroupedEntry[] = []
  let i = 0
  while (i < entries.length) {
    const entry = entries[i]
    if (entry.kind === 'message' || entry.kind === 'comment' || entry.kind === 'event' || entry.kind === 'context_pack' || entry.kind === 'tool_result' || (entry.kind === 'tool' && entry.tool.status === 'failed')) {
      result.push(entry as GroupedEntry)
      i++
      continue
    }
    const key = toolGroupKey(entry.tool)
    const group: NativeToolItem[] = [entry.tool]
    let j = i + 1
    while (j < entries.length) {
      const next = entries[j]
      if (next.kind !== 'tool') break
      if (next.tool.status === 'failed') break
      if (toolGroupKey(next.tool) !== key) break
      group.push(next.tool)
      j++
    }
    if (group.length > 1) {
      result.push({ kind: 'tool', tool: entry.tool, ts: entry.ts, group })
    } else {
      result.push({ kind: 'tool', tool: entry.tool, ts: entry.ts })
    }
    i = j
  }
  return result
}

export function entryColor(entry: UnifiedEntry): string {
  if (entry.kind === 'comment') return 'var(--orange, #f97316)'
  if (entry.kind === 'context_pack') return 'var(--purple, #8b5cf6)'
  if (entry.kind === 'tool_result') return 'var(--green)'
  if (entry.kind === 'message') return 'var(--ink-500, #6b7280)'
  if (entry.kind === 'tool') {
    if (entry.tool.status === 'failed') return 'var(--red)'
    const name = entry.tool.row.tool_name || ''
    if (name.includes('Read') || name.includes('read') || name.includes('cat')) return 'var(--blue, #3b82f6)'
    if (name.includes('Write') || name.includes('write') || name.includes('Edit') || name.includes('edit')) return 'var(--green)'
    if (name.includes('Search') || name.includes('search') || name.includes('Grep') || name.includes('grep') || name.includes('Glob') || name.includes('glob')) return 'var(--purple, #8b5cf6)'
    return 'var(--green)'
  }
  if (entry.kind === 'event') {
    if (isErrorEvent(entry.event)) return 'var(--red)'
    if (isDecisionEvent(entry.event)) return 'var(--purple, #8b5cf6)'
  }
  return 'var(--ink-300)'
}

export function buildUnifiedEntries({
  events,
  nativeEvidence,
  messages,
  comments,
  contextPacks,
  toolResults = [],
}: {
  events: QuestEvent[]
  nativeEvidence: NativeToolItem[]
  messages: MessageItem[]
  comments: CommentItem[]
  contextPacks: ContextPackItem[]
  toolResults?: ToolResultItem[]
}): UnifiedEntry[] {
  const entries: UnifiedEntry[] = []
  for (const ev of dedupeSemanticEvents(events.filter(isSemanticEvent))) {
    entries.push({ kind: 'event', event: ev, ts: ev.ts || ev.timestamp || 0 })
  }
  for (const tool of nativeEvidence) {
    entries.push({ kind: 'tool', tool, ts: tool.row.ts || 0 })
  }
  for (const msg of messages) {
    entries.push({ kind: 'message', message: msg, ts: msg.ts })
  }
  for (const c of comments) {
    entries.push({ kind: 'comment', comment: c, ts: c.ts })
  }
  for (const cp of contextPacks) {
    entries.push({ kind: 'context_pack', contextPack: cp, ts: cp.ts })
  }
  for (const tr of toolResults) {
    entries.push({ kind: 'tool_result', toolResult: tr, ts: tr.ts })
  }
  entries.sort((a, b) => a.ts - b.ts)
  return entries
}

function dedupeSemanticEvents(events: QuestEvent[]): QuestEvent[] {
  const seen = new Set<string>()
  const out: QuestEvent[] = []
  for (const event of events) {
    const key = semanticEventKey(event)
    if (seen.has(key)) continue
    seen.add(key)
    out.push(event)
  }
  return out
}

function semanticEventKey(event: QuestEvent): string {
  const payload = event.payload && typeof event.payload === 'object' && !Array.isArray(event.payload)
    ? event.payload as Record<string, unknown>
    : {}
  return [
    event.type,
    event.qid || event.quest_id || '',
    event.session_id || '',
    payload.comment || '',
    payload.verdict || '',
    payload.reason || '',
    payload.phase || payload.phase_idx || '',
  ].map(String).join('::')
}

export function filterTraceEntries(entries: GroupedEntry[], filter: TraceFilter): GroupedEntry[] {
  if (filter === 'all') return entries
  if (filter === 'tools') return entries.filter(e => e.kind === 'tool')
  if (filter === 'messages') return entries.filter(e => e.kind === 'message')
  if (filter === 'comments') return entries.filter(e => e.kind === 'comment')
  if (filter === 'context') return entries.filter(e => e.kind === 'context_pack')
  if (filter === 'errors') return entries.filter(e =>
    (e.kind === 'tool' && e.tool.status === 'failed') ||
    (e.kind === 'event' && isErrorEvent(e.event))
  )
  if (filter === 'decisions') return entries.filter(e =>
    e.kind === 'event' && isDecisionEvent(e.event)
  )
  return entries
}

export function countTraceErrors(entries: UnifiedEntry[]): number {
  return entries.filter(e =>
    (e.kind === 'tool' && e.tool.status === 'failed') ||
    (e.kind === 'event' && isErrorEvent(e.event))
  ).length
}
