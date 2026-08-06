import type { ThreadPost } from '../api/types'
import { fallbackActorMeta, type ActorTone } from '../domain/actorMeta'
import Icon from './Icon'
import MarkdownRenderer from './MarkdownRenderer'
import { fmtTime } from './util'

type Props = {
  posts: ThreadPost[]
  pinnedOutcome?: string
  onReply?: (post: ThreadPost) => void
}

// Presentation-only: CSS class suffix per tone
const CLS_FOR_TONE: Record<ActorTone, string> = {
  maker: 'maker',
  checker: 'checker',
  system: 'system',
  human: 'human',
  automation: 'automation',
}

function kindLabel(kind?: string): string {
  switch (kind) {
    case 'root': return 'Root'
    case 'maker_report': return 'MakerReport'
    case 'review_report': return 'ReviewReport'
    case 'system_rework': return 'Rework'
    case 'system_escalation': return 'Escalation'
    case 'system_workflow_upgrade': return 'WorkflowUpgrade'
    case 'decision_note': return 'DecisionNote'
    case 'milestone': return 'Milestone'
    case 'blocker': return 'Blocker'
    default: return kind || 'Post'
  }
}

function roleMeta(role?: string) {
  const meta = fallbackActorMeta(role)
  return { ...meta, cls: CLS_FOR_TONE[meta.tone as ActorTone] || 'unknown' }
}

function shortRef(value?: string): string {
  if (!value) return ''
  if (value.length <= 44) return value
  return value.slice(0, 22) + '...' + value.slice(-14)
}

export default function ThreadLedgerPanel({ posts, pinnedOutcome, onReply }: Props) {
  if ((!posts || posts.length === 0) && !pinnedOutcome) return null

  const ordered = [...(posts || [])].sort((a, b) => (a.created_at_ms || 0) - (b.created_at_ms || 0))

  return (
    <section className="panel thread-ledger-panel mt-3">
      <div className="panel-title">
        <h2>Thread Ledger</h2>
        <span className="mono">{ordered.length} posts</span>
      </div>

      {pinnedOutcome && (
        <div className="thread-pinned-outcome">
          <Icon name="sticky-note" size={14} />
          <MarkdownRenderer source={pinnedOutcome} className="compact" stripFrontmatter />
        </div>
      )}

      {ordered.length > 0 && (
        <div className="thread-ledger-list">
          {ordered.map((post) => {
            const meta = roleMeta(post.author_role)
            return (
              <article className={'thread-ledger-item role-' + meta.cls} key={post.post_id}>
                <div className="thread-ledger-rail">
                  <span className="thread-ledger-node">
                    <Icon name={meta.icon} size={13} />
                  </span>
                </div>
                <div className="thread-ledger-body">
                  <div className="thread-ledger-head">
                    <span className={'thread-role-chip ' + meta.cls}>{meta.label}</span>
                    <span className="thread-kind-chip">{kindLabel(post.kind)}</span>
                    <span className="mono thread-ledger-time">{fmtTime(post.created_at_ms)}</span>
                  </div>
                  {post.content && (
                    <MarkdownRenderer source={post.content} className="compact thread-ledger-copy" stripFrontmatter />
                  )}
                  <div className="thread-ledger-meta">
                    <span className="mono">{post.post_id}</span>
                    {post.parent_reply_id && <span className="mono">reply {post.parent_reply_id}</span>}
                    {post.source_event_id ? <span className="mono">event #{post.source_event_id}</span> : null}
                    {onReply && post.kind !== 'root' && (
                      <button className="thread-ledger-reply-btn" onClick={() => onReply(post)} title="Reply to this post">
                        <Icon name="message-circle" size={12} />
                      </button>
                    )}
                    {post.causal_refs?.slice(0, 3).map((ref) => (
                      <span className="mono thread-ledger-ref" title={ref} key={'causal-' + ref}>causal {shortRef(ref)}</span>
                    ))}
                    {(post.causal_refs?.length || 0) > 3 ? <span className="mono">+{(post.causal_refs?.length || 0) - 3} causal</span> : null}
                    {post.artifact_refs?.slice(0, 3).map((artifact) => {
                      const label = artifact.artifact_type || artifact.id || 'artifact'
                      const ref = artifact.ref || artifact.id
                      return (
                        <span
                          className="mono thread-ledger-ref"
                          title={[artifact.id, artifact.ref, artifact.role].filter(Boolean).join(' · ')}
                          key={'artifact-' + artifact.id + '-' + artifact.artifact_type}
                        >
                          artifact {label}{ref ? ' ' + shortRef(ref) : ''}
                        </span>
                      )
                    })}
                    {(post.artifact_refs?.length || 0) > 3 ? <span className="mono">+{(post.artifact_refs?.length || 0) - 3} artifacts</span> : null}
                  </div>
                </div>
              </article>
            )
          })}
        </div>
      )}
    </section>
  )
}
