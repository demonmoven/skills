import type { QuestEvent, QuestSessionRow } from '../../api/types'
import type { CommentItem, ContextPackItem, MessageItem, NativeToolItem, ToolResultItem } from './traceModel'

export type SessionTrace = {
  sid: string
  rows: QuestSessionRow[]
  contextRows: QuestSessionRow[]
}

export type EvidenceState = {
  traces: SessionTrace[]
}

export type QuestTraceItem = {
  kind: string
  ts: number
  type?: string
  payload?: unknown
  sid?: string
  tool_name?: string
  tool_args?: string
  status?: string
  result?: string
  content?: string
  turn?: number
  phase?: number
  meta?: Record<string, unknown>
}

function textFromMeta(value: unknown): string {
  if (value == null) return ''
  if (typeof value === 'string') return value
  return JSON.stringify(value, null, 2)
}

export function historicEventsFromTraceItems(items: QuestTraceItem[]): QuestEvent[] {
  return items
    .filter((item) => item.kind === 'event')
    .map((item) => ({
      type: item.type || '',
      ts: item.ts,
      payload: item.payload as Record<string, unknown> | undefined,
      session_id: item.sid,
    }))
}

export function evidenceFromTraceItems(items: QuestTraceItem[]): EvidenceState {
  const toolRows: QuestSessionRow[] = items
    .filter((item) => item.kind === 'tool')
    .map((item, i) => ({
      seq: i,
      ts: item.ts,
      kind: 'native_tool_call_observed',
      sid: item.sid,
      tool_name: item.tool_name || '',
      tool_args: item.tool_args || '',
      meta: {
        status: item.status || 'observed',
        result: item.result || '',
      },
    }))

  const messageRows: QuestSessionRow[] = items
    .filter((item) => item.kind === 'message')
    .map((item, i) => ({
      seq: item.turn || i,
      ts: item.ts,
      kind: 'message',
      sid: item.sid,
      role: 'assistant',
      content: item.content || '',
      phase: item.phase,
      meta: item.meta,
    }))

  const contextPackRows: QuestSessionRow[] = items
    .filter((item) => item.kind === 'context_pack')
    .map((item) => ({
      seq: item.turn || 0,
      ts: item.ts,
      kind: 'context_pack',
      sid: item.sid,
      content: item.content || '',
      phase: item.phase,
      meta: item.meta,
    }))

  const toolResultRows: QuestSessionRow[] = items
    .filter((item) => item.kind === 'tool_result')
    .map((item) => ({
      seq: item.turn || 0,
      ts: item.ts,
      kind: 'tool_result',
      sid: item.sid,
      role: 'tool',
      tool_name: item.tool_name || '',
      content: item.content || item.result || '',
      phase: item.phase,
      meta: item.meta,
    }))

  return {
    traces: [{
      sid: 'trace',
      rows: [...toolRows, ...messageRows, ...contextPackRows, ...toolResultRows],
      contextRows: [],
    }],
  }
}

export function nativeEvidenceFromTrace(trace: SessionTrace | null): NativeToolItem[] {
  if (!trace) return []
  return trace.rows
    .filter((row) => row.kind === 'native_tool_call_observed')
    .map((row) => ({
      sid: trace.sid,
      row,
      status: textFromMeta(row.meta?.status) || 'observed',
      result: textFromMeta(row.meta?.result) || 'result_unavailable',
      rawEvent: textFromMeta(row.meta?.raw_event),
    }))
}

export function messagesFromTrace(trace: SessionTrace | null): MessageItem[] {
  if (!trace) return []
  return trace.rows
    .filter((row) => {
      if (row.kind !== 'message' || row.role !== 'assistant') return false
      const text = String(row.content || '').trim()
      if (!text) return false
      if (text.startsWith('{"type":"system"') || text.startsWith('[{"type":"system"')) return false
      if (text.startsWith('{') && text.includes('"hook_id"')) return false
      return true
    })
    .map((row) => ({
      sid: trace.sid,
      content: row.content || '',
      turn: row.seq,
      phase: row.phase ?? 0,
      ts: row.ts,
      meta: row.meta,
    }))
}

export function commentsFromTrace(trace: SessionTrace | null): CommentItem[] {
  if (!trace) return []
  return trace.rows
    .filter((row) => row.kind === 'comment')
    .map((row) => {
      let count = 0
      if (row.meta && typeof row.meta.count === 'number') {
        count = row.meta.count as number
      }
      return {
        sid: trace.sid,
        content: row.content || '',
        turn: row.seq,
        phase: row.phase ?? 0,
        ts: row.ts,
        count,
      }
    })
}

export function contextPacksFromTrace(trace: SessionTrace | null): ContextPackItem[] {
  if (!trace) return []
  const items = trace.rows
    .filter((row) => row.kind === 'context_pack')
    .map((row) => {
      const meta = row.meta || {}
      return {
        sid: trace.sid,
        ts: row.ts,
        turn: row.seq,
        phase: row.phase ?? 0,
        data: {
            summary: (meta.summary || meta.context_pack) as ContextPackItem['data']['summary'],
            blocks: (meta.blocks || meta.context_pack_blocks) as ContextPackItem['data']['blocks'],
          rendered: row.content,
        },
      }
    })
  return dedupeContextPacks(items)
}

export function toolResultsFromTrace(trace: SessionTrace | null): ToolResultItem[] {
  if (!trace) return []
  return trace.rows
    .filter((row) => row.kind === 'tool_result' && !!String(row.content || '').trim())
    .map((row) => ({
      sid: trace.sid,
      ts: row.ts,
      turn: row.seq,
      phase: row.phase ?? 0,
      toolName: row.tool_name || '',
      content: row.content || '',
      meta: row.meta,
    }))
}

function dedupeContextPacks(items: ContextPackItem[]): ContextPackItem[] {
  const byKey = new Map<string, ContextPackItem>()
  const order: string[] = []
  for (const item of items) {
    const key = contextPackKey(item)
    if (!byKey.has(key)) order.push(key)
    byKey.set(key, item)
  }
  return order.map((key) => byKey.get(key)!).sort((a, b) => a.ts - b.ts)
}

function contextPackKey(item: ContextPackItem): string {
  const summary = item.data.summary
  const blocks = item.data.blocks || summary?.blocks || []
  const blockNames = blocks
    .map((block) => {
      const raw = block as unknown as Record<string, unknown>
      return String(raw.name || raw.Name || '')
    })
    .filter(Boolean)
    .join(',')
  return [
    item.sid,
    item.phase,
    item.turn,
    summary?.kind || '',
    blockNames,
    (item.data.rendered || '').length,
  ].join('::')
}

export function mergeByTimestamp<T extends { ts: number }>(historic: T[], live: T[]): T[] {
  if (live.length === 0) return historic
  if (historic.length === 0) return live
  const historicTs = new Set(historic.map((item) => item.ts))
  const merged = [...historic]
  for (const item of live) {
    if (!historicTs.has(item.ts)) merged.push(item)
  }
  merged.sort((a, b) => a.ts - b.ts)
  return merged
}
