import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { ActivityItem, HumanExceptionItem, QuestMeta } from '../api/types'
import { fetchActivity } from '../api/client'
import Icon from '../components/Icon'
import FeedCard from '../components/FeedCard'
import Logo from '../components/Logo'

export default function FeedView({ sorted: _sorted, humanExceptions: _humanExceptions = [], inboxItems: _inboxItems = [], onOpen, onCreateQuest, onOpenInbox: _onOpenInbox, onOpenAdventurer, activeProject: activeProjectProp, feedSearchQuery }: {
  sorted: QuestMeta[]
  humanExceptions?: HumanExceptionItem[]
  inboxItems?: QuestMeta[]
  onOpen: (qid: string) => void
  onCreateQuest: () => void
  onOpenInbox?: () => void
  onOpenAdventurer: (advId: string) => void
  activeProject?: string
  feedSearchQuery?: string
}) {
  const { t } = useTranslation()
  const [items, setItems] = useState<ActivityItem[]>([])
  const activeProject = activeProjectProp || ''
  const search = feedSearchQuery || ''
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string>('')

  const refreshActivity = useCallback((showLoading = false) => {
    if (showLoading) setLoading(true)
    return fetchActivity(activeProject || undefined, 200)
      .then((res) => {
        setItems(res.items || [])
        setError('')
      })
      .catch((e) => {
        setError(e instanceof Error ? e.message : t('common.state.error'))
      })
      .finally(() => {
        if (showLoading) setLoading(false)
      })
  }, [activeProject, t])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    fetchActivity(activeProject || undefined, 200)
      .then((res) => {
        if (!cancelled) {
          setItems(res.items || [])
          setError('')
        }
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : t('common.state.error'))
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [activeProject, t])

  useEffect(() => {
    const id = setInterval(() => {
      void refreshActivity(false).catch(() => {})
    }, 30000)
    return () => clearInterval(id)
  }, [refreshActivity])

  const filtered = useMemo(() => {
    if (!search.trim()) return items
    const q = search.trim().toLowerCase()
    return items.filter((it) => activityMatchesFeedSearch(it, q))
  }, [items, search])

  return (
    <div className="feed-view">
      <div className="feed-main">
        <div className="feed-composer">
          <div className="feed-composer-avatar">
            <Logo variant="client" size={36} />
          </div>
          <input
            className="feed-composer-input"
            placeholder={t('feed.composer.placeholder')}
            onFocus={onCreateQuest}
            readOnly
          />
          <button className="button primary feed-composer-btn" onClick={onCreateQuest} type="button">
            <Icon name="plus" size={14} />
            {t('feed.composer.submit')}
          </button>
        </div>

        <div className="feed-list">
          {loading && items.length === 0 ? (
            <div className="feed-skeleton">
              {[0, 1, 2].map((i) => (
                <div key={i} className="feed-skeleton-card">
                  <div className="feed-skeleton-avatar" />
                  <div className="feed-skeleton-body">
                    <div className="feed-skeleton-line w60" />
                    <div className="feed-skeleton-line w100" />
                    <div className="feed-skeleton-line w80" />
                  </div>
                </div>
              ))}
            </div>
          ) : error && items.length === 0 ? (
            <div className="feed-empty"><Icon name="triangle-alert" size={14} /><span>{error}</span></div>
          ) : filtered.length === 0 ? (
            <div className="feed-empty">
              <Icon name="radio" size={14} />
              <span>{items.length === 0 ? t('feed.empty.noActivity') : t('feed.empty.noMatch')}</span>
              {items.length === 0 && (
                <button className="feed-empty-cta" onClick={onCreateQuest} type="button">
                  <Icon name="plus" size={13} />{t('feed.empty.cta')}
                </button>
              )}
            </div>
          ) : (
            filtered.map((it, i) => {
              const prev = i > 0 ? filtered[i - 1] : undefined
              return (
                <FeedCard
                  key={it.quest_id + ':' + it.ts + ':' + it.kind}
                  item={it}
                  prevQuestId={prev?.quest_id}
                  prevSpeaker={prev?.warrior_id}
                  prevName={prev?.kind === 'user' ? t('feed.speaker.you') : prev?.warrior_name}
                  onOpen={onOpen}
                  onOpenAdventurer={onOpenAdventurer}
                  onOpenInbox={_onOpenInbox}
                  onActivityChanged={() => { void refreshActivity(false) }}
                />
              )
            })
          )}
        </div>
      </div>
    </div>
  )
}

export function activityMatchesFeedSearch(item: ActivityItem, normalizedQuery: string): boolean {
  const q = normalizedQuery.trim().toLowerCase()
  if (!q) return true
  const fields = [
    item.query,
    item.summary,
    item.warrior_name,
    item.group_id,
    item.merge_owner_leaf_id,
    ...(item.fanout_leaf_ids || []),
    ...(item.fanout_quest_ids || []),
    ...(item.reply_previews || []).map((reply) => reply.summary || ''),
    ...(item.reply_previews || []).map((reply) => reply.post_id || ''),
  ]
  return fields.some((value) => (value || '').toLowerCase().includes(q))
}
