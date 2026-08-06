import { useState } from 'react'
import type { ActivityItem, ThreadPost } from '../api/types'
import { getQuestDetail, resolveBlockedQuest } from '../api/quests'
import { actorMeta, type ActorMeta, type ActorTone } from '../domain/actorMeta'
import i18n from '../i18n'
import { useTranslation } from 'react-i18next'
import Icon from './Icon'
import Logo from './Logo'
import { relativeTime } from './util'

// Presentation-only: avatar shape per tone
const AVATAR_FOR_TONE: Record<ActorTone, 'knight' | 'mage' | 'badge' | 'client'> = {
  maker: 'knight',
  checker: 'mage',
  system: 'badge',
  human: 'client',
  automation: 'badge',
}

// post_kind 只映射成轻状态 dot，不做类型标签
// system_workflow_upgrade（手动/API 升档 reply）与 system_escalation（blocked 投影）是两个不同 kind，
// 用不同色区分：升档走 system 色，blocked 走 blocker 色，避免把升档误当成阻塞。
const POST_KIND_DOT: Record<string, 'milestone' | 'blocker' | 'review' | 'system'> = {
  milestone: 'milestone',
  blocker: 'blocker',
  review_report: 'review',
  system_rework: 'system',
  system_escalation: 'blocker',
  system_workflow_upgrade: 'system',
  decision_note: 'milestone',
}

function postKindDot(kind?: string): string | null {
  if (!kind) return null
  return POST_KIND_DOT[kind] || null
}

// v0.5 slice 1-C: 卡片主体只处理来自自身的键盘事件。子按钮（展开/查看完整 thread/收起）的
// Enter/Space 冒泡上来时忽略——否则键盘用户在子按钮上按 Enter 会触发整卡跳 Detail 而非按钮自身。
export function cardShouldHandleKeydown(e: { currentTarget: unknown; target: unknown; key: string }): boolean {
  if (e.key !== 'Enter' && e.key !== ' ') return false
  return e.currentTarget === e.target
}

function sourceLabel(source?: string): string | null {
  if (!source || source === 'user') return null
  if (source.startsWith('automation:')) {
    const name = source.slice('automation:'.length)
    return name ? i18n.t('feed.source.automationPrefix', { name }) : i18n.t('feed.source.automation')
  }
  if (source === 'parent_spawn') return i18n.t('feed.source.parentSpawn')
  return source
}

function isAutomationActor(item: ActivityItem): boolean {
  if (item.author_role === 'automation') return true
  if (item.source && item.source.startsWith('automation:')) return true
  return false
}

function displayAuthor(item: ActivityItem, role: ActorMeta | null): string {
  if (item.kind === 'user') {
    if (isAutomationActor(item)) return i18n.t('feed.author.automation')
    return i18n.t('feed.author.you')
  }
  if (role) {
    if (role.role === 'human') return item.warrior_name || i18n.t('feed.author.default')
    return role.label
  }
  if (isAutomationActor(item)) return i18n.t('feed.author.automation')
  return item.warrior_name || i18n.t('feed.author.default')
}

function avatarFor(item: ActivityItem, role: ActorMeta | null): 'knight' | 'mage' | 'badge' | 'client' {
  if (item.kind === 'attention') return 'badge'
  if (item.kind === 'user') {
    if (isAutomationActor(item)) return 'badge'
    return 'client'
  }
  if (role) return AVATAR_FOR_TONE[role.tone] ?? 'knight'
  if (isAutomationActor(item)) return 'badge'
  return item.warrior_class === 'mage' ? 'mage' : 'knight'
}

export default function FeedCard({ item, prevQuestId, prevSpeaker, prevName, onOpen, onOpenAdventurer, onOpenInbox, onActivityChanged }: {
  item: ActivityItem
  prevQuestId?: string
  prevSpeaker?: string
  prevName?: string
  onOpen: (qid: string) => void
  onOpenAdventurer: (advId: string) => void
  onOpenInbox?: () => void
  onActivityChanged?: () => void
}) {
  const { t } = useTranslation()
  const isUser = item.kind === 'user'
  const isAttention = item.kind === 'attention'
  const role = actorMeta(item.author_role)
  const isSystem = item.author_role === 'system' || item.post_kind?.startsWith('system_')
  const isAutomation = isAutomationActor(item)
  const authorName = isAttention ? t('feed.attention.author') : displayAuthor(item, role)
  const avatarVariant = avatarFor(item, role)
  const avatarClickable = !isUser && !isSystem && !isAutomation && !!item.warrior_id
  const kindDot = postKindDot(item.post_kind)
  const fanoutLeafCount = item.fanout_fold ? (item.fanout_leaf_count || item.fanout_leaf_ids?.length || 0) : 0
  const replyPreviews = item.reply_previews || []
  const hiddenReplyCount = Math.max(0, (item.reply_count || 0) - replyPreviews.length)
  const srcLabel = sourceLabel(item.source)
  const isCandidate = isUser && item.triage_mode === 'candidate' && item.status === 'blocked'

  const [expanded, setExpanded] = useState(false)
  const [attentionBusy, setAttentionBusy] = useState<string>('')
  // v0.5 slice 1-C: inline thread expand — 展开时拉 getQuestDetail(thread_posts)，不伪造未拉取的回复。
  const [threadExpanded, setThreadExpanded] = useState(false)
  const [threadPosts, setThreadPosts] = useState<ThreadPost[] | null>(null)
  const [threadLoading, setThreadLoading] = useState(false)
  const [threadError, setThreadError] = useState(false)
  const isLong = (item.summary || '').length > 280

  async function toggleThreadExpand() {
    if (threadExpanded) { setThreadExpanded(false); return }
    if (threadPosts) { setThreadExpanded(true); return }
    setThreadLoading(true)
    setThreadError(false)
    try {
      const detail = await getQuestDetail(item.quest_id)
      setThreadPosts(detail.thread_posts || [])
      setThreadExpanded(true)
    } catch {
      setThreadError(true)
    } finally {
      setThreadLoading(false)
    }
  }

  // 展开态渲染完整 thread replies（排除当前卡片代表的 root post），按时间排序；拉取前为 null，不伪造。
  // 复制后再 filter/sort，避免原地改 threadPosts（该数组可能被别的视图复用）。
  const expandedThreadReplies = threadExpanded && threadPosts
    ? [...threadPosts]
        .filter((p) => p.post_id !== item.post_id && p.post_id !== item.root_post_id)
        .sort((a, b) => a.created_at_ms - b.created_at_ms)
    : null

  const sameSpeaker = prevSpeaker !== undefined && prevSpeaker === item.warrior_id
  const continues = !!prevQuestId && prevQuestId === item.quest_id && sameSpeaker
  const showAuthor = !continues

  const replyTarget = !isUser && !isAttention && item.reply_to
    ? item.reply_to
    : !isUser && !isAttention && prevQuestId === item.quest_id && !sameSpeaker && prevName
    ? prevName
    : t('feed.author.you')

  const hasImpact = !isUser && !!item.impact && (
    !!item.impact.what_changed ||
    !!item.impact.affected?.length ||
    !!item.impact.not_touched?.length ||
    !!item.impact.caveats?.length
  )

  const showProject = item.project && item.project !== '(未分类)'
  const summary = item.summary || ''

  async function handleAttentionAction(action: string) {
    if (item.action_endpoint_hint !== 'resolve-blocked') {
      onOpen(item.quest_id)
      return
    }
    if ((action === 'cancel' || action === 'user-review') && !window.confirm(t('feed.attention.confirm', { action: attentionActionLabel(action) }))) {
      return
    }
    setAttentionBusy(action)
    try {
      await resolveBlockedQuest(item.quest_id, {
        action,
        add_turns: 0,
        add_duration_minutes: 0,
      })
      onActivityChanged?.()
    } finally {
      setAttentionBusy('')
    }
  }

  function attentionActionLabel(action: string): string {
    switch (action) {
      case 'continue': return t('feed.attention.action.continue')
      case 'user-review': return t('feed.attention.action.userReview')
      case 'cancel': return t('feed.attention.action.cancel')
      case 'answer': return t('feed.attention.action.answer')
      case 'pass': return t('feed.attention.action.pass')
      case 'request_changes': return t('feed.attention.action.requestChanges')
      case 'reject': return t('feed.attention.action.reject')
      case 'apply': return t('feed.attention.action.apply')
      case 'discard': return t('feed.attention.action.discard')
      default: return action
    }
  }

  return (
    <article className={'feed-card' + (continues ? ' continues-group' : '') + (isAutomation ? ' is-automation' : '') + (isAttention ? ' is-attention risk-' + (item.risk_level || 'medium') : '')}>
      <div className="feed-card-main" onClick={() => onOpen(item.quest_id)} role="button" tabIndex={0}
        onKeyDown={(e) => { if (cardShouldHandleKeydown(e)) { e.preventDefault(); onOpen(item.quest_id) } }}>
        {avatarClickable ? (
          <button
            className="feed-avatar feed-avatar-link"
            type="button"
            onClick={(e) => { e.stopPropagation(); onOpenAdventurer(item.warrior_id!) }}
            title={t('feed.avatar.viewAdventurerName', { name: item.warrior_name || '' }).trim()}
            aria-label={t('feed.avatar.viewAdventurerName', { name: item.warrior_name || '' }).trim()}
          >
            <Logo variant={avatarVariant} size={40} />
          </button>
        ) : (
          <div className="feed-avatar">
            <Logo variant={avatarVariant} size={40} />
          </div>
        )}
        <div className="feed-body">
          {showAuthor && (
            <header className="feed-card-head">
              <span className="feed-author-name">{authorName}</span>
              {!isUser && !isSystem && !isAutomation && !isAttention && (
                <span className={'feed-verified-badge' + (item.warrior_class === 'mage' ? ' mage' : '')}
                  title={t('feed.role.verifiedTitle')}>
                  <Icon name="circle-check" size={14} />
                </span>
              )}
              {role && !isUser && !isAttention && (
                <span className={'feed-role-chip ' + role.tone}>
                  <Icon name={role.icon} size={11} />
                  {role.label}
                </span>
              )}
              {isAutomation && (
                <span className="feed-role-chip automation">
                  <Icon name="bolt" size={11} />
                  Automation
                </span>
              )}
              {isAttention && (
                <span className={'feed-role-chip attention risk-' + (item.risk_level || 'medium')}>
                  <Icon name="triangle-alert" size={11} />
                  {t('feed.attention.label')}
                </span>
              )}
              {item.warrior_title && <em className="feed-author-sub"> · {item.warrior_title}</em>}
              {item.warrior_level ? <em className="mono feed-author-sub"> Lv.{item.warrior_level}</em> : null}
              <span className="feed-sep">·</span>
              <span className="mono feed-handle">#{item.short_id}</span>
              <span className="feed-sep">·</span>
              <span className="mono feed-time">{relativeTime(item.ts)}</span>
              {kindDot && <span className={'feed-state-dot kind-' + kindDot} title={item.post_kind} />}
              {srcLabel && <span className="feed-source-inline">· {srcLabel}</span>}
            </header>
          )}
          {!isUser && !isAttention && (
            <div className="feed-reply-context" title={item.causal_refs?.length ? `causal ${item.causal_refs.length}` : undefined}>
              <Icon name="corner-up-left" size={12} />
              <span>{isSystem ? 'thread update' : t('feed.reply.contextLabel')} <em>{replyTarget}</em></span>
            </div>
          )}
          {summary && !item.fanout_fold && (
            <>
              <pre className={'feed-summary-inline' + (isUser ? ' feed-root-body' : '') + (isLong && !expanded ? ' collapsed' : '')}>{summary}</pre>
              {isLong && (
                <button className="feed-expand-btn" type="button" onClick={(e) => { e.stopPropagation(); setExpanded(!expanded) }}>
                  {expanded ? t('feed.expand.collapse') : t('feed.expand.expand')}
                </button>
              )}
            </>
          )}
          {isAttention && (
            <div className="feed-attention-panel">
              {item.recommended_action && (
                <div className="feed-attention-row">
                  <span>{t('feed.attention.recommended')}</span>
                  <strong>{item.recommended_action}</strong>
                </div>
              )}
              <div className="feed-attention-meta">
                {item.risk_level && <span>{t('feed.attention.risk')}: {item.risk_level}</span>}
                {item.blocked_reason_code && <span className="mono">{item.blocked_reason_code}</span>}
                {item.blocked_category && <span>{item.blocked_category}</span>}
              </div>
            </div>
          )}
          {fanoutLeafCount > 0 && (
            <div className="feed-fanout-fold" title={item.merge_owner_leaf_id ? `merge owner ${item.merge_owner_leaf_id}` : undefined}>
              <Icon name="git-branch" size={13} />
              <span>{t('feed.expand.fanout', { count: fanoutLeafCount })}</span>
            </div>
          )}
          {isUser && replyPreviews.length > 0 && (
            <div className="feed-reply-preview-list">
              {expandedThreadReplies
                ? expandedThreadReplies.length === 0
                  ? <div className="feed-reply-preview-more static">{t('feed.expand.noMoreReplies')}</div>
                  : expandedThreadReplies.map((post) => {
                      const postRole = actorMeta(post.author_role)
                      const postIsAutomation = post.author_role === 'automation'
                      const content = post.content || ''
                      const truncated = content.length > 200 ? content.slice(0, 200) + '…' : content
                      return (
                        <div className={'feed-reply-preview role-' + (postRole?.tone || (postIsAutomation ? 'automation' : 'unknown'))} key={post.post_id}>
                          <span className="feed-reply-preview-dot">
                            <Icon name={postRole?.icon || (postIsAutomation ? 'bolt' : 'message-circle')} size={10} />
                          </span>
                          <span className="feed-reply-preview-meta">
                            {postRole?.label || (postIsAutomation ? 'Automation' : 'Reply')}
                          </span>
                          <span className="feed-reply-preview-text">{truncated}</span>
                          {post.source_event_id ? <span className="mono feed-reply-preview-event" title={`event #${post.source_event_id}`}>#{post.source_event_id}</span> : null}
                        </div>
                      )
                    })
                : replyPreviews.map((reply) => {
                    const replyRole = actorMeta(reply.author_role)
                    const replyIsAutomation = reply.author_role === 'automation'
                    return (
                      <div className={'feed-reply-preview role-' + (replyRole?.tone || (replyIsAutomation ? 'automation' : 'unknown'))} key={reply.post_id}>
                        <span className="feed-reply-preview-dot">
                          <Icon name={replyRole?.icon || (replyIsAutomation ? 'bolt' : 'message-circle')} size={10} />
                        </span>
                        <span className="feed-reply-preview-meta">
                          {replyRole?.label || (replyIsAutomation ? 'Automation' : 'Reply')}
                        </span>
                        <span className="feed-reply-preview-text">{reply.summary}</span>
                      </div>
                    )
                  })}
              {hiddenReplyCount > 0 && !threadExpanded && (
                <button
                  className="feed-reply-preview-more"
                  type="button"
                  disabled={threadLoading}
                  onClick={(e) => { e.stopPropagation(); void toggleThreadExpand() }}
                >
                  <Icon name={threadLoading ? 'loader' : 'chevron-down'} size={11} />
                  {threadLoading ? t('feed.expand.loading') : threadError ? t('feed.expand.loadFailed') : t('feed.expand.expandReplies', { count: hiddenReplyCount })}
                </button>
              )}
              {threadExpanded && threadPosts && (
                <>
                  <button
                    className="feed-reply-preview-more link"
                    type="button"
                    onClick={(e) => { e.stopPropagation(); onOpen(item.quest_id) }}
                  >
                    <Icon name="message-square" size={11} />
                    {t('feed.expand.viewThread')}
                  </button>
                  <button
                    className="feed-reply-preview-more"
                    type="button"
                    onClick={(e) => { e.stopPropagation(); setThreadExpanded(false) }}
                  >
                    <Icon name="chevron-up" size={11} />
                    {t('feed.expand.collapseThread')}
                  </button>
                </>
              )}
            </div>
          )}
          {hasImpact && item.impact && (
            <div className="feed-impact-inline">
              {item.impact.what_changed && <span><em>{t('feed.impact.changed')}</em> {item.impact.what_changed}</span>}
              {!!item.impact.affected?.length && <span><em>{t('feed.impact.affected')}</em> {item.impact.affected.join('、')}</span>}
              {!!item.impact.not_touched?.length && <span><em>{t('feed.impact.untouched')}</em> {item.impact.not_touched.join('、')}</span>}
              {!!item.impact.caveats?.length && <span><em>{t('feed.impact.risk')}</em> {item.impact.caveats.join('、')}</span>}
            </div>
          )}
          <div className="feed-card-actions">
            {isCandidate && (
              <span className="feed-candidate-actions">
                {onOpenInbox ? (
                  <button
                    className="feed-candidate-tag"
                    type="button"
                    onClick={(e) => { e.stopPropagation(); onOpenInbox() }}
                  >
                    <Icon name="inbox" size={11} />
                    {t('feed.candidate.tag')}
                  </button>
                ) : (
                  <em className="feed-candidate-tag">{t('feed.candidate.tag')}</em>
                )}
              </span>
            )}
            {isAttention && item.available_actions?.length ? (
              <span className="feed-attention-actions">
                {item.available_actions.slice(0, 4).map((action) => (
                  <button
                    key={action}
                    className={'feed-attention-action ' + (action === 'cancel' ? 'danger' : action === 'continue' ? 'primary' : '')}
                    type="button"
                    disabled={!!attentionBusy}
                    onClick={(e) => { e.stopPropagation(); void handleAttentionAction(action) }}
                  >
                    {attentionBusy === action ? t('feed.attention.action.busy') : attentionActionLabel(action)}
                  </button>
                ))}
              </span>
            ) : null}
            {showProject && <span className="feed-project-tag">{item.project}</span>}
            {isUser && (item.warrior_adv_name || item.mage_adv_name) && (
              <>
                {item.warrior_adv_name && (
                  <button
                    className="feed-mention-tag"
                    type="button"
                    onClick={(e) => { e.stopPropagation(); item.warrior_adv_id && onOpenAdventurer(item.warrior_adv_id) }}
                    title={t('feed.mention.viewAdventurer', { name: item.warrior_adv_name })}
                  >
                    <Icon name="sword" size={11} />@{item.warrior_adv_name}
                  </button>
                )}
                {item.mage_adv_name && (
                  <button
                    className="feed-mention-tag mage"
                    type="button"
                    onClick={(e) => { e.stopPropagation(); item.mage_adv_id && onOpenAdventurer(item.mage_adv_id) }}
                    title={t('feed.mention.viewAdventurer', { name: item.mage_adv_name })}
                  >
                    <Icon name="wand-sparkles" size={11} />@{item.mage_adv_name}
                  </button>
                )}
              </>
            )}
            {isUser && item.parent_short_id && (
              <span className="feed-link-tag" title={t('feed.link.derivedFromTitle', { id: item.parent_quest_id })}>
                <Icon name="git-merge" size={11} />{t('feed.link.derivedFrom', { id: item.parent_short_id })}
              </span>
            )}
            {isUser && item.child_short_id && (
              <span className="feed-link-tag" title={t('feed.link.generatedTitle', { id: item.child_quest_id })}>
                <Icon name="chevron-right" size={11} />{t('feed.link.generated', { id: item.child_short_id })}
              </span>
            )}
            {isUser && item.group_id && (
              <span className="feed-link-tag" title={'fanout group ' + item.group_id}>
                <Icon name="git-branch" size={11} />{item.group_id}
              </span>
            )}
            <span className="feed-thread-tag">
              <Icon name="message-square" size={11} />
              thread <span className="mono">#{item.short_id}</span>
              {item.reply_count ? <em>{t('feed.thread.replyCount', { count: item.reply_count })}</em> : null}
            </span>
          </div>
        </div>
      </div>
    </article>
  )
}
