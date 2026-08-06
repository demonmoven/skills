import type { FailureAttributionPayload, QuestEvent } from '../api/types'
import { useTranslation } from 'react-i18next'
import AgentMessageRenderer from './AgentMessageRenderer'
import Icon from './Icon'
import { fmtTime } from './util'

function payloadText(payload: unknown): string {
  if (payload == null) return ''
  if (typeof payload === 'string') return payload
  try {
    return JSON.stringify(payload, null, 2)
  } catch {
    return String(payload)
  }
}

function isCompactPayload(payload: unknown): payload is Record<string, unknown> {
  const obj = asObject(payload)
  if (!obj) return false
  const entries = Object.entries(obj)
  if (entries.length === 0 || entries.length > 6) return false
  return entries.every(([, v]) =>
    v == null || typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean'
  )
}

function CompactPayload({ payload }: { payload: Record<string, unknown> }) {
  const entries = Object.entries(payload).filter(([, v]) => v != null && v !== '')
  if (entries.length === 0) return null
  return (
    <div className="event-chips">
      {entries.map(([k, v]) => (
        <span className="event-chip" key={k}>
          <span className="event-chip-key">{k}</span>
          <span className="event-chip-val">{String(v)}</span>
        </span>
      ))}
    </div>
  )
}

function asObject(payload: unknown): Record<string, unknown> | null {
  return payload && typeof payload === 'object' && !Array.isArray(payload)
    ? payload as Record<string, unknown>
    : null
}

function asFailurePayload(payload: unknown): FailureAttributionPayload {
  return (asObject(payload) || {}) as FailureAttributionPayload
}

function compactValue(value: unknown): string {
  if (value == null || value === '') return ''
  if (Array.isArray(value)) return value.map(compactValue).filter(Boolean).join(', ')
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function detailRows(details: unknown): { key: string; value: string }[] {
  const obj = asObject(details)
  if (!obj) return []
  return Object.entries(obj)
    .map(([key, value]) => ({ key, value: compactValue(value) }))
    .filter((row) => row.value !== '')
}

function noteText(payload: unknown): string | null {
  const obj = asObject(payload)
  if (!obj) return null
  if (typeof obj.content === 'string') return obj.content
  if (typeof obj.note === 'string') return obj.note
  return null
}

function isNotePayload(payload: unknown): boolean {
  return noteText(payload) != null
}

function NoteCard({ payload }: { payload: unknown }) {
  const obj = asObject(payload) || {}
  const text = noteText(payload) || ''
  const isUser = obj.source === 'user'
  return (
    <div className={`note-card${isUser ? ' note-user' : ''}`}>
      <p className="note-content">{text}</p>
    </div>
  )
}

function eventIcon(type: string): string {
  if (type === 'failure.attributed') return 'shield-alert'
  if (type === 'quest.note') return 'message-circle'
  if (type.includes('review')) return 'mage'
  if (type.includes('blocked') || type.includes('failed')) return 'triangle-alert'
  return 'sword'
}

function FailureAttributionCard({ payload }: { payload: FailureAttributionPayload }) {
  const { t } = useTranslation()
  const rows = detailRows(payload.details)
  return (
    <div className="failure-attribution-card">
      <div className="failure-card-head">
        <div>
          <strong>{payload.reason || 'failure'}</strong>
          <span>{[payload.stage, payload.category].filter(Boolean).join(' · ') || 'unclassified'}</span>
        </div>
        <em className={payload.recoverable ? 'recoverable' : 'terminal'}>
          {payload.recoverable ? t('eventTimeline.recoverable') : t('eventTimeline.needHuman')}
        </em>
      </div>
      {payload.message && <p>{payload.message}</p>}
      {Array.isArray(payload.actions) && payload.actions.length > 0 && (
        <div className="failure-actions">
          {payload.actions.map((action) => (
            <span className="chip mono tiny" key={action}>{action}</span>
          ))}
        </div>
      )}
      {rows.length > 0 && (
        <dl className="failure-details">
          {rows.map((row) => (
            <div key={row.key}>
              <dt>{row.key}</dt>
              <dd>{row.value}</dd>
            </div>
          ))}
        </dl>
      )}
    </div>
  )
}

export default function EventTimeline({ events }: { events: QuestEvent[] }) {
  const { t } = useTranslation()
  if (events.length === 0) {
    return (
      <div className="empty-panel">
        <Icon name="clock" />
        <span>{t('eventTimeline.waitingSSE')}</span>
      </div>
    )
  }

  return (
    <div className="timeline">
      {events.slice(-80).map((event, index) => (
        <div
          className={'timeline-row' + (event.type === 'failure.attributed' ? ' failure-row' : event.type === 'quest.note' ? ' note-row' : '')}
          key={index + ':' + event.type + ':' + event.ts}
        >
          <div className="timeline-dot">
            <Icon name={eventIcon(event.type)} />
          </div>
          <div className="timeline-body">
            <div className="timeline-meta">
              <strong>{event.type}</strong>
              <span>{fmtTime(event.ts || event.timestamp)}</span>
            </div>
            {event.type === 'failure.attributed'
              ? <FailureAttributionCard payload={asFailurePayload(event.payload)} />
              : event.type === 'quest.note' && isNotePayload(event.payload)
                ? <NoteCard payload={event.payload} />
                : isCompactPayload(event.payload)
                  ? <CompactPayload payload={event.payload} />
                  : payloadText(event.payload) && (
                    <AgentMessageRenderer
                      content={payloadText(event.payload)}
                      className="timeline-payload"
                      compact
                      maxLength={3200}
                    />
                  )}
          </div>
        </div>
      ))}
    </div>
  )
}
