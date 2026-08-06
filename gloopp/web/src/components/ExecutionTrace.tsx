import { useState, useMemo } from 'react'
import type { QuestEvent } from '../api/types'
import i18n from '../i18n'
import { useTranslation } from 'react-i18next'
import {
  buildUnifiedEntries,
  compactPayload,
  countTraceErrors,
  entryColor,
  eventCategory,
  eventLabel,
  extractFullCommand,
  extractToolResult,
  extractToolTarget,
  filterTraceEntries,
  groupConsecutiveTools,
  payloadAsGrid,
  resultSummary,
  selectKeyEvents,
  toolCategory,
  tryParseJSON,
  type CommentItem,
  type ContextPackItem,
  type GroupedEntry,
  type MessageItem,
  type NativeToolItem,
  type ToolResultItem,
  type TraceFilter,
  type UnifiedEntry,
} from '../features/quest/traceModel'
import { blockRoleLabel } from '../domain/contextLabels'
import AgentMessageRenderer from './AgentMessageRenderer'
import Icon from './Icon'
import ContextPackView from './ContextPackView'
import { fmtTime } from './util'

interface Props {
  events: QuestEvent[]
  nativeEvidence: NativeToolItem[]
  messages?: MessageItem[]
  comments?: CommentItem[]
  contextPacks?: ContextPackItem[]
  toolResults?: ToolResultItem[]
  loading?: boolean
  messageArtifactLinks?: Record<string, { artifactId: string; artifactName: string }>
  onJumpToArtifact?: (artifactId: string) => void
  onJumpToConclusion?: () => void
}

function TraceMinimap({ entries }: { entries: UnifiedEntry[] }) {
  if (entries.length === 0) return null
  return (
    <div className="exec-minimap" aria-hidden="true">
      {entries.map((e, i) => (
        <div
          key={i}
          className="exec-minimap-bar"
          style={{ background: entryColor(e) }}
        />
      ))}
    </div>
  )
}

function ToolMetaGrid({ tool }: { tool: NativeToolItem }) {
  const row = tool.row
  const args = row.tool_args ? tryParseJSON(row.tool_args) : null
  const pairs: Array<{ key: string; value: string }> = []

  pairs.push({ key: 'TOOL', value: row.tool_name || 'unknown' })
  pairs.push({ key: 'STATUS', value: tool.status })
  if (row.phase != null) pairs.push({ key: 'PHASE', value: String(row.phase) })
  if (row.sid) pairs.push({ key: 'SESSION', value: row.sid.slice(0, 8) })
  if (args && typeof args.file_path === 'string') pairs.push({ key: 'FILE', value: args.file_path as string })

  return (
    <div className="exec-meta-grid">
      {pairs.map((p, i) => (
        <div key={i} className="exec-meta-cell">
          <span className="exec-meta-key">{p.key}</span>
          <span className="exec-meta-val" title={p.value}>{p.value}</span>
        </div>
      ))}
    </div>
  )
}

function phaseLabel(phase: number | undefined): string {
  if (phase === 0) return i18n.t('execTrace.phase.warrior')
  if (phase === 1) return i18n.t('execTrace.phase.mage')
  return i18n.t('execTrace.phase.default')
}

function contextRoleSummary(cp: ContextPackItem): string {
  const blocks = cp.data.blocks || cp.data.summary?.blocks || []
  const counts = new Map<string, number>()
  for (const block of blocks) {
    const raw = block as unknown as Record<string, unknown>
    const role = String(raw.role || raw.Role || '')
    if (!role) continue
    counts.set(role, (counts.get(role) || 0) + 1)
  }
  const ordered = ['control', 'primary_input', 'evidence', 'history', 'protocol', 'supporting']
  const parts = ordered
    .filter(role => counts.has(role))
    .map(role => `${blockRoleLabel(role)} ${counts.get(role)}`)
  return parts.join(' · ')
}

function toolResultSummary(value: string): string | null {
  if (!value) return null
  try {
    const parsed = JSON.parse(value) as { message?: unknown; data?: unknown }
    const data = parsed.data && typeof parsed.data === 'object' ? parsed.data as { summary?: unknown; status?: unknown } : null
    if (typeof data?.summary === 'string' && data.summary.trim()) {
      return data.summary.trim().split('\n')[0]?.slice(0, 140) || null
    }
    if (typeof parsed.message === 'string') return parsed.message
    if (typeof data?.status === 'string') return data.status
  } catch { /* fall through */ }
  return value.split('\n')[0]?.slice(0, 140) || null
}

/** 摘要层人话 label：把工程化 event type 翻译成用户可读的动作描述。 */
function summaryEventLabel(type: string, payload: unknown): string {
  const p = (payload && typeof payload === 'object' && !Array.isArray(payload)) ? payload as Record<string, unknown> : {}
  if (type === 'quest.started') return i18n.t('execTrace.event.questStarted')
  if (type === 'quest.failed') return i18n.t('execTrace.event.questFailed')
  if (type === 'quest.cancelled') return i18n.t('execTrace.event.questCancelled')
  if (type === 'phase.started') {
    const phase = String(p.phase_name || p.phase || '')
    return phase ? i18n.t('execTrace.event.phaseEntered', { phase }) : i18n.t('execTrace.event.phaseEnteredGeneric')
  }
  if (type === 'phase.completed') {
    const phase = String(p.phase_name || p.phase || '')
    return phase ? i18n.t('execTrace.event.phaseCompleted', { phase }) : i18n.t('execTrace.event.phaseCompletedGeneric')
  }
  if (type.includes('review') && type.includes('verdict')) return i18n.t('execTrace.event.mageVerdict')
  if (type.includes('review')) return i18n.t('execTrace.event.mageReview')
  if (type === 'quest.agent_update') return i18n.t('execTrace.event.warriorReport')
  if (type === 'failure.attributed') return i18n.t('execTrace.event.failureAttributed')
  if (type === 'quest.user_review') return i18n.t('execTrace.event.userReview')
  if (type.includes('blocked')) return i18n.t('execTrace.event.blocked')
  return eventLabel(type)
}

export default function ExecutionTrace({ events, nativeEvidence, messages = [], comments = [], contextPacks = [], toolResults = [], loading, messageArtifactLinks, onJumpToArtifact, onJumpToConclusion }: Props) {
  const { t } = useTranslation()
  const [filter, setFilter] = useState<TraceFilter>('all')
  const [viewMode, setViewMode] = useState<'summary' | 'full'>('summary')
  const [expandedItems, setExpandedItems] = useState<Set<number>>(new Set())

  // 默认摘要层：关键节点（决策 ∪ 错误 ∪ 状态切换，最近 5 个）+ 最近 2 条 agent 输出，
  // 构成"摘要版工作轨迹"——既有状态变化脉络，也有 agent 最近在说什么。原始流下沉到完整层。
  const keyEvents = useMemo(() => selectKeyEvents(events, 5), [events])
  const summaryMessages = useMemo(() => messages.slice(-2), [messages])
  const isSummary = viewMode === 'summary'

  const fullUnified = useMemo<UnifiedEntry[]>(
    () => buildUnifiedEntries({ events, nativeEvidence, messages, comments, contextPacks, toolResults }),
    [events, nativeEvidence, messages, comments, contextPacks, toolResults],
  )

  const unified = useMemo<UnifiedEntry[]>(
    () => isSummary
      ? buildUnifiedEntries({ events: keyEvents, nativeEvidence: [], messages: summaryMessages, comments: [], contextPacks: [], toolResults: [] })
      : fullUnified,
    [isSummary, keyEvents, summaryMessages, fullUnified],
  )

  const grouped = useMemo(() => groupConsecutiveTools(unified), [unified])
  const filtered = useMemo(() => isSummary ? grouped : filterTraceEntries(grouped, filter), [grouped, filter, isSummary])
  // 异常计数基于全量，避免摘要层隐藏后误判为无异常。
  const errorCount = useMemo(() => countTraceErrors(fullUnified), [fullUnified])

  function toggleExpand(index: number) {
    setExpandedItems(prev => {
      const next = new Set(prev)
      if (next.has(index)) next.delete(index)
      else next.add(index)
      return next
    })
  }

  if (loading) {
    return (
      <div className="exec-trace">
        <div className="exec-trace-empty">
          <Icon name="spinner" className="spin" />
          <span>{t('execTrace.loading')}</span>
        </div>
      </div>
    )
  }

  if (fullUnified.length === 0) {
    return (
      <div className="exec-trace">
        <div className="exec-trace-empty">
          <Icon name="clock" />
          <span>{t('execTrace.waiting')}</span>
        </div>
      </div>
    )
  }

  return (
    <div className="exec-trace">
      <div className="exec-trace-header">
        <h2>{t('execTrace.title')}</h2>
        <div className="exec-trace-view-toggle">
          <button className={isSummary ? 'active' : ''} onClick={() => setViewMode('summary')}>{t('execTrace.view.summary')}</button>
          <button className={!isSummary ? 'active' : ''} onClick={() => setViewMode('full')}>{t('execTrace.view.full')}</button>
        </div>
        {!isSummary && (
          <div className="exec-trace-filters">
            <button className={filter === 'all' ? 'active' : ''} onClick={() => setFilter('all')}>
              {t('execTrace.filter.all')} <em>{fullUnified.length}</em>
            </button>
            <button className={filter === 'tools' ? 'active' : ''} onClick={() => setFilter('tools')}>
              {t('execTrace.filter.tools')} <em>{nativeEvidence.length}</em>
            </button>
            {messages.length > 0 && (
              <button className={filter === 'messages' ? 'active' : ''} onClick={() => setFilter('messages')}>
                {t('execTrace.filter.messages')} <em>{messages.length}</em>
              </button>
            )}
            {comments.length > 0 && (
              <button className={filter === 'comments' ? 'active' : ''} onClick={() => setFilter('comments')}>
                {t('execTrace.filter.comments')} <em>{comments.length}</em>
              </button>
            )}
            {contextPacks.length > 0 && (
              <button className={filter === 'context' ? 'active' : ''} onClick={() => setFilter('context')}>
                {t('execTrace.filter.context')} <em>{contextPacks.length}</em>
              </button>
            )}
            <button className={filter === 'decisions' ? 'active' : ''} onClick={() => setFilter('decisions')}>
              {t('execTrace.filter.decisions')}
            </button>
            {errorCount > 0 && (
              <button className={`err ${filter === 'errors' ? 'active' : ''}`} onClick={() => setFilter('errors')}>
                {t('execTrace.filter.errors')} <em>{errorCount}</em>
              </button>
            )}
          </div>
        )}
      </div>

      {isSummary ? (
        <div className="exec-trace-summary-hint">
          <Icon name="signal" size={12} />
          <span>{t('execTrace.summaryHint', { keyCount: keyEvents.length, msgCount: summaryMessages.length, total: events.length })}</span>
          {(events.length > keyEvents.length || messages.length > summaryMessages.length) && (
            <button className="link-button tiny" onClick={() => setViewMode('full')}>{t('execTrace.viewFull')}</button>
          )}
        </div>
      ) : (
        <TraceMinimap entries={unified} />
      )}

      {isSummary && keyEvents.length === 0 ? (
        <div className="exec-trace-empty">
          <Icon name="clock" />
          <span>{t('execTrace.noKeyEvents')}</span>
          <button className="link-button tiny" onClick={() => setViewMode('full')}>{t('execTrace.viewFull')}</button>
        </div>
      ) : (
        <>
          <div className="exec-trace-table-head">
            <span className="exec-col-type">TYPE</span>
            <span className="exec-col-detail">DETAILS</span>
            <span className="exec-col-time">TIME</span>
          </div>

          <div className="exec-trace-list">
            {filtered.map((entry, i) => {
          if (entry.kind === 'event') {
            const ev = entry.event
            const cat = eventCategory(ev.type)
            const payload = compactPayload(ev.payload)
            const isExpanded = expandedItems.has(i)
            const grid = payloadAsGrid(ev.payload)

            return (
              <div className={`exec-trace-row exec-event ${cat.variant}`} key={`ev-${i}`}>
                <div className="exec-col-type">
                  <span className={`exec-pill ${cat.variant}`}>{cat.label}</span>
                </div>
                <div className="exec-col-detail">
                  <button
                    type="button"
                    className="exec-row-btn"
                    onClick={() => toggleExpand(i)}
                    aria-expanded={isExpanded}
                  >
                    <Icon name={isExpanded ? 'chevron-down' : 'chevron-right'} size={12} className="exec-chevron" />
                    <strong>{isSummary ? summaryEventLabel(ev.type, ev.payload) : eventLabel(ev.type)}</strong>
                    {!isExpanded && payload && <span className="exec-detail-hint" title={payload}>{payload}</span>}
                  </button>
                  {isExpanded && grid && (
                    <div className="exec-meta-grid">
                      {grid.map((p, gi) => (
                        <div key={gi} className="exec-meta-cell">
                          <span className="exec-meta-key">{p.key}</span>
                          <span className="exec-meta-val" title={p.value}>{p.value}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
                <div className="exec-col-time">{fmtTime(ev.ts || ev.timestamp)}</div>
              </div>
            )
          }

          if (entry.kind === 'message') {
            const msg = entry.message
            const isExpanded = expandedItems.has(i)
            const label = phaseLabel(msg.phase)
            const linkKey = msg.sid + ':' + String(msg.phase) + ':' + String(msg.turn) + ':' + String(msg.ts)
            const archived = messageArtifactLinks?.[linkKey]
            const isWarrior = (msg.phase ?? 0) === 0

            return (
              <div className="exec-trace-row exec-message" key={`msg-${i}`} data-message-key={linkKey} data-sid={msg.sid} data-turn={msg.turn}>
                <div className="exec-col-type">
                  <span className="exec-pill message">{label}</span>
                </div>
                <div className="exec-col-detail">
                  <button
                    type="button"
                    className="exec-row-btn"
                    onClick={() => toggleExpand(i)}
                    aria-expanded={isExpanded}
                  >
                    <Icon name={isExpanded ? 'chevron-down' : 'chevron-right'} size={12} className="exec-chevron" />
                    <strong>Turn {msg.turn}</strong>
                    {!isExpanded && (
                      <span className="exec-detail-hint exec-message-preview">
                        <AgentMessageRenderer content={msg.content} compact maxLength={140} />
                      </span>
                    )}
                  </button>
                  <div className="exec-message-meta">
                    {isWarrior && onJumpToConclusion && (
                      <button type="button" className="link-button micro" onClick={onJumpToConclusion}>
                        <Icon name="flag" size={11} /> {t('execTrace.pinConclusion')}
                      </button>
                    )}
                    {archived && onJumpToArtifact && (
                      <button
                        type="button"
                        className="link-button micro"
                        title={t('execTrace.archivedTitle', { name: archived.artifactName })}
                        onClick={() => onJumpToArtifact(archived.artifactId)}
                      >
                        <Icon name="archive" size={11} /> {t('execTrace.archived', { name: archived.artifactName })}
                      </button>
                    )}
                  </div>
                  {isExpanded && (
                    <div className="exec-message-content">
                      <AgentMessageRenderer content={msg.content} className="exec-message-body" maxLength={6000} />
                    </div>
                  )}
                </div>
                <div className="exec-col-time">{fmtTime(msg.ts)}</div>
              </div>
            )
          }

          if (entry.kind === 'comment') {
            const cmt = entry.comment
            const label = phaseLabel(cmt.phase)
            return (
              <div className="exec-trace-row exec-comment" key={`comment-${i}`}>
                <div className="exec-col-type">
                  <span className="exec-pill comment">{t('execTrace.commentInjected')}</span>
                </div>
                <div className="exec-col-detail">
                  <div className="exec-row-btn" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <strong>{cmt.content || t('execTrace.newComments', { count: cmt.count })}</strong>
                    <span className="exec-detail-hint">
                      Turn {cmt.turn} · {label}{t('execTrace.phaseSuffix')}
                    </span>
                  </div>
                </div>
                <div className="exec-col-time">{fmtTime(cmt.ts)}</div>
              </div>
            )
          }

          if (entry.kind === 'context_pack') {
            const cp = entry.contextPack
            const isExpanded = expandedItems.has(i)
            const blockCount = cp.data.blocks?.length || cp.data.summary?.blocks?.length || 0
            const label = phaseLabel(cp.phase)
            const roles = contextRoleSummary(cp)

            return (
              <div className="exec-trace-row exec-context-pack" key={`ctx-${i}`}>
                <div className="exec-col-type">
                  <span className="exec-pill decision">{t('execTrace.contextPack')}</span>
                </div>
                <div className="exec-col-detail">
                  <button
                    type="button"
                    className="exec-row-btn"
                    onClick={() => toggleExpand(i)}
                    aria-expanded={isExpanded}
                  >
                    <Icon name={isExpanded ? 'chevron-down' : 'chevron-right'} size={12} className="exec-chevron" />
                    <strong>{label}{t('execTrace.phaseSuffix')} · {t('execTrace.contextStart')}</strong>
                    <span className="exec-detail-hint">
                      {t('execTrace.contextBlocks', { count: blockCount })} · Turn {cp.turn}
                      {roles && ` · ${roles}`}
                    </span>
                  </button>
                  {isExpanded && (
                    <div className="mt-2">
                      <ContextPackView data={cp.data} compact />
                    </div>
                  )}
                </div>
                <div className="exec-col-time">{fmtTime(cp.ts)}</div>
              </div>
            )
          }

          if (entry.kind === 'tool_result') {
            const tr = entry.toolResult
            const isExpanded = expandedItems.has(i)
            const label = phaseLabel(tr.phase)
            const summary = toolResultSummary(tr.content)

            return (
              <div className="exec-trace-row exec-tool-result" key={`tool-result-${i}`}>
                <div className="exec-col-type">
                  <span className="exec-pill phase">{t('execTrace.result')}</span>
                </div>
                <div className="exec-col-detail">
                  <button
                    type="button"
                    className="exec-row-btn"
                    onClick={() => toggleExpand(i)}
                    aria-expanded={isExpanded}
                  >
                    <Icon name={isExpanded ? 'chevron-down' : 'chevron-right'} size={12} className="exec-chevron" />
                    <strong>{tr.toolName || 'tool_result'}</strong>
                    <span className="exec-detail-hint">Turn {tr.turn} · {label}{t('execTrace.phaseSuffix')}</span>
                  </button>
                  {!isExpanded && summary && <p className="exec-tool-summary">{summary}</p>}
                  {isExpanded && (
                    <div className="exec-tool-detail">
                      <pre className="exec-tool-output">{tr.content.length > 4000 ? tr.content.slice(0, 4000) + '\n...' : tr.content}</pre>
                    </div>
                  )}
                </div>
                <div className="exec-col-time">{fmtTime(tr.ts)}</div>
              </div>
            )
          }

          const { tool, group } = entry as GroupedEntry & { kind: 'tool' }
          const isFailed = tool.status === 'failed'
          const isExpanded = expandedItems.has(i)
          const target = tool.row.tool_args ? extractToolTarget(tool.row.tool_args) : null
          const toolName = tool.row.tool_name || 'tool'
          const cat = toolCategory(toolName)
          const count = group ? group.length : 1
          const summary = !isExpanded ? resultSummary(tool.result) : null

          return (
            <div className={`exec-trace-row exec-tool ${isFailed ? 'failed' : 'ok'}`} key={`tool-${i}`}>
              <div className="exec-col-type">
                <span className={`exec-pill ${cat.variant}${isFailed ? ' pill-err' : ''}`}>{cat.label}</span>
              </div>
              <div className="exec-col-detail">
                <button
                  type="button"
                  className="exec-row-btn"
                  onClick={() => toggleExpand(i)}
                  aria-expanded={isExpanded}
                >
                  <Icon name={isExpanded ? 'chevron-down' : 'chevron-right'} size={12} className="exec-chevron" />
                  <strong>{toolName}</strong>
                  {target && <span className="exec-detail-hint" title={target}>{target}</span>}
                  {count > 1 && <span className="exec-tool-count">×{count}</span>}
                  {isFailed && <span className="exec-fail-badge">failed</span>}
                </button>

                {!isExpanded && isFailed && (() => {
                  const errResult = extractToolResult(tool.result)
                  const errText = errResult || tool.result || ''
                  const firstLine = errText.split('\n')[0]?.slice(0, 120)
                  return firstLine ? <p className="exec-tool-error-hint">{firstLine}</p> : null
                })()}

                {!isExpanded && !isFailed && summary && <p className="exec-tool-summary">{summary}</p>}

                {isExpanded && (
                  <div className="exec-tool-detail">
                    {group ? (
                      <>
                        <ToolMetaGrid tool={tool} />
                        <div className="exec-tool-group-items">
                          {group.map((item, gi) => {
                            const itemTarget = item.row.tool_args ? extractToolTarget(item.row.tool_args) : null
                            const itemCmd = item.row.tool_args ? extractFullCommand(item.row.tool_args) : null
                            const itemResult = extractToolResult(item.result)
                            return (
                              <details key={gi} className="exec-tool-group-item">
                                <summary>
                                  <span className="exec-group-item-target">{itemTarget || toolName}</span>
                                  <span className={`exec-tool-status mini ${item.status}`}>
                                    {item.status === 'failed' ? 'failed' : ''}
                                  </span>
                                  <span className="exec-time">{fmtTime(item.row.ts)}</span>
                                </summary>
                                {itemCmd && <code className="exec-tool-full-cmd">{itemCmd}</code>}
                                {itemResult && (
                                  <pre className="exec-tool-output">{itemResult.length > 2000 ? itemResult.slice(0, 2000) + '\n...' : itemResult}</pre>
                                )}
                              </details>
                            )
                          })}
                        </div>
                      </>
                    ) : (
                      <>
                        <ToolMetaGrid tool={tool} />
                        {(() => {
                          const fullCmd = tool.row.tool_args ? extractFullCommand(tool.row.tool_args) : null
                          const fullResult = extractToolResult(tool.result)
                          return (
                            <>
                              {fullCmd && (
                                <div className="exec-detail-section">
                                  <span className="exec-detail-section-label">COMMAND</span>
                                  <code className="exec-tool-full-cmd">{fullCmd}</code>
                                </div>
                              )}
                              {fullResult && (
                                <div className="exec-detail-section">
                                  <span className="exec-detail-section-label">OUTPUT</span>
                                  <pre className="exec-tool-output">{fullResult.length > 4000 ? fullResult.slice(0, 4000) + '\n...' : fullResult}</pre>
                                </div>
                              )}
                              {!fullResult && tool.result && tool.result !== 'result_unavailable' && (
                                <div className="exec-detail-section">
                                  <span className="exec-detail-section-label">RAW</span>
                                  <pre className="exec-tool-output">{tool.result.length > 4000 ? tool.result.slice(0, 4000) + '\n...' : tool.result}</pre>
                                </div>
                              )}
                            </>
                          )
                        })()}
                      </>
                    )}
                  </div>
                )}
              </div>
              <div className="exec-col-time">{fmtTime(tool.row.ts)}</div>
            </div>
          )
        })}
      </div>
        </>
      )}
    </div>
  )
}
