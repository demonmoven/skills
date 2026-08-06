import type { FanoutGroup, FanoutLeaf } from '../api/types'
import { useTranslation } from 'react-i18next'
import Icon from './Icon'
import { fmtTime, shortId } from './util'

type Props = {
  group?: FanoutGroup | null
  onOpenQuest: (qid: string) => void
}

function labelForMergeStrategy(strategy?: string): string {
  switch (strategy) {
    case 'single_leaf': return 'single leaf'
    case 'sequential': return 'sequential'
    case 'no_merge': return 'no merge'
    default: return strategy || 'unspecified'
  }
}

function leafTitle(leaf: FanoutLeaf): string {
  return leaf.fanout_leaf_id || shortId(leaf.quest_id)
}

export function visibleFanoutLeaves(group?: FanoutGroup | null): FanoutLeaf[] {
  if (!group || !group.group_id || !Array.isArray(group.leaves)) return []
  return group.leaves
    .filter((leaf) => !!leaf.quest_id && !!leaf.fanout_leaf_id)
    .sort((a, b) => {
      if (a.is_current !== b.is_current) return a.is_current ? -1 : 1
      const left = a.fanout_leaf_id || ''
      const right = b.fanout_leaf_id || ''
      return left.localeCompare(right)
    })
}

export default function FanoutTreePanel({ group, onOpenQuest }: Props) {
  const { t } = useTranslation()
  const leaves = visibleFanoutLeaves(group)
  if (!group || leaves.length === 0) return null

  const mergeOwners = new Set(leaves.map((leaf) => leaf.merge_owner_leaf_id).filter(Boolean))

  return (
    <section className="panel fanout-tree-panel mt-3">
      <div className="panel-title">
        <h2>Fanout Tree</h2>
        <span className="mono">{group.group_id} · {leaves.length} leaves</span>
      </div>

      <div className="fanout-tree-list" role="list">
        {leaves.map((leaf) => {
          const isMergeOwner = mergeOwners.has(leaf.fanout_leaf_id)
          return (
            <article className={'fanout-leaf-card' + (leaf.is_current ? ' is-current' : '')} key={leaf.quest_id} role="listitem">
              <div className="fanout-leaf-main">
                <div className="fanout-leaf-head">
                  <span className="fanout-leaf-icon">
                    <Icon name={isMergeOwner ? 'git-merge' : 'git-branch'} size={14} />
                  </span>
                  <strong className="mono">{leafTitle(leaf)}</strong>
                  {leaf.is_current && <span className="thread-kind-chip">Current</span>}
                  {isMergeOwner && <span className="thread-kind-chip">Merge owner</span>}
                </div>
                <p>{leaf.query}</p>
                <div className="fanout-leaf-scopes">
                  {(leaf.ownership_scopes || []).map((scope) => (
                    <span className="mono" key={scope}>{scope}</span>
                  ))}
                  {(!leaf.ownership_scopes || leaf.ownership_scopes.length === 0) && (
                    <span className="muted">No ownership scope declared</span>
                  )}
                </div>
              </div>

              <div className="fanout-leaf-meta">
                <span><em>status</em>{leaf.status}</span>
                <span><em>merge</em>{labelForMergeStrategy(leaf.merge_strategy)}</span>
                {leaf.merge_owner_leaf_id && <span><em>owner</em>{leaf.merge_owner_leaf_id}</span>}
                <span><em>updated</em>{fmtTime(leaf.updated_at_ms || leaf.created_at_ms)}</span>
              </div>

              <button
                type="button"
                className="link-button tiny fanout-open-button"
                onClick={() => onOpenQuest(leaf.quest_id)}
                disabled={leaf.is_current}
                title={leaf.is_current ? t('fanout.currentQuest') : t('fanout.openLeaf')}
              >
                <Icon name="arrow-up-right" size={12} /> {leaf.is_current ? t('fanout.current') : t('fanout.open')}
              </button>
            </article>
          )
        })}
      </div>
    </section>
  )
}
