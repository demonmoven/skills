import { useEffect, useState } from 'react'
import type { QuestEvent, QuestMeta } from '../api/types'
import type { StreamState } from '../api/events'
import Icon from '../components/Icon'
import Logo from '../components/Logo'
import { fmtTime } from '../components/util'
import type { QuestActionState, QuestPhaseView, QuestStatusBadgeInfo, QuestAttention } from '../domain/questSelectors'
import { deriveQuestAttention, deriveQuestProgressLine } from '../domain/questSelectors'
import { getStuckDiagnosis } from '../features/quest/stuckDiagnosisMetrics'
import i18n from '../i18n'

type Props = {
  quest: QuestMeta
  statusBadge: QuestStatusBadgeInfo
  phaseView: QuestPhaseView
  actionState: QuestActionState
  streamState: StreamState
  /** 最近一条 SSE 事件的时间戳；用于判断数据新鲜度。0 表示尚无事件。 */
  lastEventAtMs: number
  events?: QuestEvent[]
  onJumpToTrace: () => void
}

/** CSS class for attention: snake_case → kebab-case */
function attentionCss(attention: QuestAttention): string {
  return attention.kind === 'none' ? 'none' : attention.kind.replace(/_/g, '-')
}

function handlerLabel(quest: QuestMeta, phaseView: QuestPhaseView): { label: string; sub: string; variant: 'knight' | 'mage' } {
  const isQuick = String(quest.intensity || '').toLowerCase() === 'quick'
  const role = phaseView.role
  if (role === 'review' && !isQuick && phaseView.available) {
    return { label: i18n.t('world.mage'), sub: quest.mage_id || i18n.t('quest.handler.notConfigured'), variant: 'mage' }
  }
  return { label: i18n.t('world.warrior'), sub: quest.warrior_id || i18n.t('quest.handler.notConfigured'), variant: 'knight' }
}

function connCopy(): Record<StreamState, { text: string; tone: 'live' | 'warn' | 'off' }> {
  return {
    connecting: { text: i18n.t('quest.conn.connecting'), tone: 'warn' },
    open: { text: i18n.t('quest.conn.live'), tone: 'live' },
    reconnecting: { text: i18n.t('quest.conn.reconnecting'), tone: 'warn' },
    disconnected: { text: i18n.t('quest.conn.disconnected'), tone: 'off' },
  }
}

const STALE_THRESHOLD_MS = 60_000

function runtimeEventLabel(type: string): string {
  const map: Record<string, string> = {
    'runtime.session.created': i18n.t('quest.runtime.sessionCreated'),
    'runtime.process.started': i18n.t('quest.runtime.processStarted'),
    'runtime.output.first_seen': i18n.t('quest.runtime.firstOutput'),
    'runtime.output.heartbeat': i18n.t('quest.runtime.heartbeat'),
    'runtime.protocol.signal_seen': i18n.t('quest.runtime.signalSeen'),
    'runtime.idle_warning': i18n.t('quest.runtime.idleWarning'),
    'runtime.protocol_warning': i18n.t('quest.runtime.protocolWarning'),
    'runtime.process.exited': i18n.t('quest.runtime.processExited'),
    'runtime.error.classified': i18n.t('quest.runtime.errorClassified'),
  }
  return map[type] || type
}

function latestRuntimeEvent(events: QuestEvent[] | undefined): QuestEvent | null {
  if (!events || events.length === 0) return null
  for (let i = events.length - 1; i >= 0; i--) {
    if (events[i].type.startsWith('runtime.')) return events[i]
  }
  return null
}

function latestPersistedRuntimeEvent(quest: QuestMeta): QuestEvent | null {
  const sessions = Array.isArray(quest.agent_sessions) ? quest.agent_sessions : []
  let latest = sessions[0]
  for (const item of sessions) {
    if ((item.updated_at_ms || 0) > (latest?.updated_at_ms || 0)) latest = item
  }
  if (!latest) return null
  const payload: Record<string, unknown> = {
    exit_code: latest.exit_code,
    capability_tier: latest.capability_tier,
    health_status: latest.health_status,
  }
  const type = latest.health_status === 'process_started'
    ? 'runtime.process.started'
    : latest.health_status === 'error'
    ? 'runtime.error.classified'
    : latest.health_status === 'idle'
    ? 'runtime.idle_warning'
    : latest.health_status === 'protocol_risk'
    ? 'runtime.protocol_warning'
    : latest.health_status === 'waiting_signal'
    ? 'runtime.protocol.signal_seen'
    : latest.exit_code !== undefined
      ? 'runtime.process.exited'
      : latest.first_output_at_ms
        ? 'runtime.output.first_seen'
        : latest.started_at_ms
          ? 'runtime.session.created'
          : ''
  return type ? { type, session_id: latest.session_id, payload, ts: latest.updated_at_ms } : null
}

function runtimeHealthCopy(event: QuestEvent | null, quest: QuestMeta): { text: string; tone: 'live' | 'warn' | 'off' } {
  if (!event) {
    if (quest.status === 'pending' || quest.status === 'success' || quest.status === 'failed' || quest.status === 'cancelled') {
      return { text: i18n.t('quest.runtime.noSession'), tone: 'off' }
    }
    return { text: i18n.t('quest.runtime.waitingSignal'), tone: 'warn' }
  }
  if (event.type === 'runtime.error.classified') {
    const payload = event.payload as Record<string, unknown> | undefined
    const category = typeof payload?.category === 'string' ? payload.category : 'unknown'
    return { text: i18n.t('quest.runtime.errorCategory', { category }), tone: 'off' }
  }
  if (event.type === 'runtime.idle_warning') {
    return { text: i18n.t('quest.runtime.idleWarning'), tone: 'warn' }
  }
  if (event.type === 'runtime.protocol_warning') {
    return { text: i18n.t('quest.runtime.protocolWarning'), tone: 'warn' }
  }
  if (event.type === 'runtime.process.exited') {
    const payload = event.payload as Record<string, unknown> | undefined
    const code = typeof payload?.exit_code === 'number' ? payload.exit_code : undefined
    return { text: code === 0 ? i18n.t('quest.runtime.processExitedClean') : i18n.t('quest.runtime.processExitedCode', { code: code ?? '' }).trim(), tone: code === 0 ? 'live' : 'off' }
  }
  return { text: runtimeEventLabel(event.type), tone: 'live' }
}

export default function CurrentWorkStatus({ quest, statusBadge, phaseView, actionState, streamState, lastEventAtMs, events, onJumpToTrace }: Props) {
  const [now, setNow] = useState(() => Date.now())
  // 低频刷新，仅用于新鲜度判断；连接态变化时由 props 触发重渲。
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 15_000)
    return () => clearInterval(timer)
  }, [])

  const questAttention = deriveQuestAttention(quest)
  const attnCss = attentionCss(questAttention)
  const diagnosis = getStuckDiagnosis(quest)
  const progressLine = deriveQuestProgressLine(quest)
  const progress = diagnosis?.recommendedAction || i18n.t(progressLine.i18nKey, progressLine.params as Record<string, unknown>)
  const handler = handlerLabel(quest, phaseView)
  const updatedMs = Number(quest.updated_at_ms) || Number(quest.started_at_ms) || Number(quest.created_at_ms) || 0

  const conn = { ...connCopy()[streamState] }
  const runtimeEvent = latestRuntimeEvent(events) || latestPersistedRuntimeEvent(quest)
  const runtime = runtimeHealthCopy(runtimeEvent, quest)
  const stale = streamState === 'open' && lastEventAtMs > 0 && now - lastEventAtMs > STALE_THRESHOLD_MS
  if (stale) {
    conn.text = i18n.t('quest.conn.stale')
    conn.tone = 'warn'
  }

  const phaseLabel = phaseView.available
    ? phaseView.name
    : quest.status === 'pending'
      ? i18n.t('quest.phase.notStarted')
      : i18n.t('quest.phase.unavailable')
  const phaseRole = phaseView.available
    ? (phaseView.role === 'review' ? i18n.t('quest.phase.roleReview') : i18n.t('quest.phase.roleExecute'))
    : ''

  return (
    <section className={'current-work-status' + (attnCss !== 'none' ? ' attn-' + attnCss : '')} aria-label={i18n.t('aria.currentWorkStatus')}>
      <div className="cws-grid">
        <div className="cws-cell cws-status">
          <span className="cws-label">{i18n.t('quest.meta.status')}</span>
          <span className="cws-value">{statusBadge.label}</span>
        </div>
        <div className="cws-cell cws-phase">
          <span className="cws-label">{i18n.t('quest.meta.phase')}</span>
          <span className="cws-value">{phaseLabel}{phaseRole && <em className="cws-phase-role">{phaseRole}</em>}</span>
        </div>
        <div className="cws-cell cws-handler">
          <span className="cws-label">{i18n.t('quest.meta.handler')}</span>
          <span className="cws-value">
            <Logo variant={handler.variant} size={16} />
            <span className="mono faint">{handler.label} · {handler.sub}</span>
          </span>
        </div>
        <div className="cws-cell cws-conn">
          <span className="cws-label">{i18n.t('quest.meta.connection')}</span>
          <span className={'cws-value cws-conn-' + conn.tone}>
            <span className={'cws-conn-dot ' + conn.tone} />
            {conn.text}
          </span>
        </div>
        <div className="cws-cell cws-runtime">
          <span className="cws-label">{i18n.t('quest.meta.agentRuntime')}</span>
          <span className={'cws-value cws-runtime-' + runtime.tone} title={runtimeEvent?.type || ''}>
            <span className={'cws-conn-dot ' + runtime.tone} />
            {runtime.text}
          </span>
        </div>
      </div>

      <div className="cws-progress-row">
        <div className="cws-progress-text">
          <Icon name={attnCss !== 'none' ? 'triangle-alert' : 'signal'} size={14} />
          <span>{progress}</span>
        </div>
        {diagnosis && (
          <div className={'cws-diagnosis source-' + diagnosis.source}>
            <span>{diagnosis.title}</span>
            <em>{diagnosis.source === 'runtime' ? i18n.t('quest.diagnosis.runtimeState') : i18n.t('quest.diagnosis.humanException')}</em>
          </div>
        )}
        <div className="cws-progress-meta">
          {updatedMs > 0 && <span className="cws-updated">{i18n.t('quest.meta.updatedAt', { time: fmtTime(updatedMs) })}</span>}
          <button type="button" className="link-button tiny" onClick={onJumpToTrace}>
            <Icon name="search" size={12} /> {i18n.t('quest.action.viewTrace')}
          </button>
        </div>
      </div>
    </section>
  )
}
