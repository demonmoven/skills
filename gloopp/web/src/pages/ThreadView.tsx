import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useEventStream, type StreamState } from '../api/events'
import { getQuestThread } from '../api/quests'
import type { ThreadActionEntrypoint, ThreadActor, ThreadFanoutSummary, ThreadPost, ThreadViewResponse } from '../api/types'
import Icon from '../components/Icon'
import StatusBadge from '../components/StatusBadge'
import ThreadLedgerPanel from '../components/ThreadLedgerPanel'
import { fmtTime, shortId } from '../components/util'
import { fallbackActorMeta } from '../domain/actorMeta'

type Props = {
  qid: string
  onBack: () => void
  onOpenThread: (qid: string) => void
  onOpenDetail: (qid: string) => void
  onError: (msg: string) => void
  onChanged: () => void
}

export default function ThreadView({ qid, onBack, onOpenThread, onOpenDetail, onError, onChanged }: Props) {
  const { t } = useTranslation()
  const [data, setData] = useState<ThreadViewResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [composerText, setComposerText] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [replyingTo, setReplyingTo] = useState<ThreadPost | null>(null)

  const loadThread = useCallback(() => {
    setLoading(true)
    setError('')
    return getQuestThread(qid)
      .then((res) => {
        if (!res.ok) {
          setError('Failed to load thread')
          return
        }
        setData(res)
      })
      .catch((e) => {
        setError(e instanceof Error ? e.message : String(e))
      })
      .finally(() => setLoading(false))
  }, [qid])

  useEffect(() => {
    void loadThread()
  }, [loadThread])

  // P2: Real-time thread streaming via SSE.
  // When semantic events arrive for this quest, refetch the thread to show new posts.
  const loadThreadRef = useRef(loadThread)
  loadThreadRef.current = loadThread
  const lastFetchRef = useRef(0)
  const streamState: StreamState = useEventStream(() => {
    const now = Date.now()
    // Debounce: don't refetch more than once every 1.5s.
    if (now - lastFetchRef.current < 1500) return
    lastFetchRef.current = now
    void loadThreadRef.current()
  }, qid, true)

  const handleSubmitComment = useCallback(() => {
    const text = composerText.trim()
    if (!text) return
    setSubmitting(true)
    import('../api/quests').then(({ commentOnQuest }) => {
      return commentOnQuest(qid, text, replyingTo?.post_id)
        .then(() => {
          setComposerText('')
          setReplyingTo(null)
          onChanged()
          return loadThread()
        })
        .catch((e) => onError(e instanceof Error ? e.message : String(e)))
        .finally(() => setSubmitting(false))
    })
  }, [composerText, qid, replyingTo, loadThread, onError, onChanged])

  const pinnedOutcome = useMemo(() => {
    if (!data?.posts) return ''
    const decision = data.posts.find((p) => p.kind === 'decision_note')
    return decision?.content || ''
  }, [data])

  const orderedPosts = useMemo(() => {
    if (!data?.posts) return []
    return [...data.posts].sort((a, b) => (a.created_at_ms || 0) - (b.created_at_ms || 0))
  }, [data])

  if (loading && !data) {
    return (
      <div className="thread-view loading">
        <div className="thread-view-header">
          <button className="icon-button" onClick={onBack} aria-label={t('aria.back')}>
            <Icon name="chevron-left" size={18} />
          </button>
          <span>{t('common.state.loading')}</span>
        </div>
      </div>
    )
  }

  if (error && !data) {
    return (
      <div className="thread-view error">
        <div className="thread-view-header">
          <button className="icon-button" onClick={onBack} aria-label={t('aria.back')}>
            <Icon name="chevron-left" size={18} />
          </button>
          <span className="error-text">{error}</span>
        </div>
      </div>
    )
  }

  if (!data) return null

  const { quest, posts, actors, fanout_summary, action_entrypoints } = data

  return (
    <div className="thread-view">
      <div className="thread-view-header">
        <button className="icon-button" onClick={onBack} aria-label={t('aria.back')}>
          <Icon name="chevron-left" size={18} />
        </button>
        <div className="thread-view-title">
          <h1 className="thread-query">{quest.query}</h1>
          <div className="thread-view-meta">
            <span className="mono thread-short-id">#{shortId(quest.id)}</span>
            <StatusBadge status={quest.status} />
            <span className="mono thread-updated">{fmtTime(quest.updated_at_ms || quest.created_at_ms)}</span>
            <span className={'thread-stream-indicator stream-' + streamState} title={t('quest.thread.stream_' + streamState)}>
              <span className="thread-stream-dot" />
              {streamState === 'open' ? t('quest.thread.stream_live') : t('quest.thread.stream_' + streamState)}
            </span>
          </div>
        </div>
        <div className="thread-view-actions">
          <button className="button secondary" onClick={() => onOpenDetail(quest.id)}>
            <Icon name="sliders" size={14} />
            {t('quest.detail.open_management')}
          </button>
        </div>
      </div>

      {fanout_summary && <FanoutSummaryBar summary={fanout_summary} onOpenLeaf={onOpenThread} />}

      {action_entrypoints && action_entrypoints.length > 0 && (
        <ActionEntrypointsBar entries={action_entrypoints} onOpenDetail={() => onOpenDetail(quest.id)} />
      )}

      {actors && actors.length > 0 && (
        <div className="thread-actors-bar">
          {actors.map((actor) => (
            <ActorChip key={actor.identity} actor={actor} />
          ))}
        </div>
      )}

      <ThreadLedgerPanel posts={orderedPosts} pinnedOutcome={pinnedOutcome} onReply={setReplyingTo} />

      <div className="thread-composer">
        {replyingTo && (
          <div className="thread-composer-reply-context">
            <span className="mono">
              {t('quest.thread.replying_to', { author: replyingTo.author_identity || replyingTo.author_role || 'post' })}
            </span>
            <button className="icon-button tiny" onClick={() => setReplyingTo(null)} aria-label={t('common.action.cancel')}>
              <Icon name="x" size={12} />
            </button>
          </div>
        )}
        <textarea
          className="thread-composer-input"
          placeholder={t('quest.thread.reply_placeholder')}
          value={composerText}
          onChange={(e) => setComposerText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
              e.preventDefault()
              void handleSubmitComment()
            }
          }}
          rows={2}
        />
        <div className="thread-composer-actions">
          <span className="thread-composer-hint">{t('quest.thread.cmd_enter_to_send')}</span>
          <button
            className="button primary"
            onClick={handleSubmitComment}
            disabled={submitting || !composerText.trim()}
          >
            {submitting ? t('common.state.sending') : t('quest.thread.send')}
          </button>
        </div>
      </div>
    </div>
  )
}

function ActorChip({ actor }: { actor: ThreadActor }) {
  const meta = fallbackActorMeta(actor.role)
  return (
    <span className={'thread-actor-chip role-' + meta.tone}>
      <Icon name={meta.icon} size={12} />
      <span className="thread-actor-name">{actor.name || actor.identity}</span>
      <span className="thread-actor-role">{actor.role}</span>
    </span>
  )
}

function FanoutSummaryBar({ summary, onOpenLeaf }: { summary: ThreadFanoutSummary; onOpenLeaf: (qid: string) => void }) {
  const { t } = useTranslation()
  if (summary.is_root && summary.leaves && summary.leaves.length > 0) {
    return (
      <div className="thread-fanout-bar">
        <div className="thread-fanout-label">
          <Icon name="git-branch" size={14} />
          {t('quest.thread.fanout_root', { count: summary.leaves.length })}
        </div>
        <div className="thread-fanout-leaves">
          {summary.leaves.map((leaf) => (
            <button
              key={leaf.quest_id}
              className="thread-fanout-leaf-chip"
              onClick={() => onOpenLeaf(leaf.quest_id)}
              title={leaf.latest_post_preview}
            >
              <span className="mono">#{shortId(leaf.quest_id)}</span>
              <span className={'status-dot status-' + leaf.status} />
              {leaf.latest_post_preview && (
                <span className="thread-fanout-leaf-preview">{leaf.latest_post_preview.slice(0, 60)}</span>
              )}
            </button>
          ))}
        </div>
      </div>
    )
  }
  if (summary.is_leaf && summary.root_quest_id) {
    return (
      <div className="thread-fanout-bar">
        <div className="thread-fanout-label">
          <Icon name="git-branch" size={14} />
          {t('quest.thread.fanout_leaf')}
        </div>
        <button className="thread-fanout-root-link" onClick={() => onOpenLeaf(summary.root_quest_id!)}>
          {t('quest.thread.open_root')} #{shortId(summary.root_quest_id)}
        </button>
      </div>
    )
  }
  return null
}

function ActionEntrypointsBar({ entries, onOpenDetail }: { entries: ThreadActionEntrypoint[]; onOpenDetail: () => void }) {
  const { t } = useTranslation()
  const primary = entries.filter((e) => e.priority === 0)
  const secondary = entries.filter((e) => e.priority > 0)

  return (
    <div className="thread-action-bar">
      {primary.map((entry, i) => (
        <button
          key={'primary-' + i}
          className={'button action-entrypoint action-' + entry.action_type}
          onClick={onOpenDetail}
          title={entry.hint}
        >
          <ActionIcon type={entry.action_type} />
          {entry.hint || entry.action_type}
        </button>
      ))}
      {secondary.map((entry, i) => (
        <button
          key={'secondary-' + i}
          className={'button secondary action-entrypoint action-' + entry.action_type}
          onClick={onOpenDetail}
          title={entry.hint}
        >
          <ActionIcon type={entry.action_type} />
          {entry.hint || entry.action_type}
        </button>
      ))}
    </div>
  )
}

function ActionIcon({ type }: { type: string }) {
  switch (type) {
    case 'answer':
      return <Icon name="message-square" size={14} />
    case 'resolve_review':
      return <Icon name="check-circle" size={14} />
    case 'resolve_blocked':
      return <Icon name="alert-triangle" size={14} />
    case 'add_comment':
      return <Icon name="edit-3" size={14} />
    default:
      return <Icon name="zap" size={14} />
  }
}
