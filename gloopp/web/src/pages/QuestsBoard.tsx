import { useState } from 'react'
import type { AdventurerFile, ExecutorInfo, HumanExceptionItem, QuestMeta, QuestStatus } from '../api/types'
import type { FieldKey } from '../components/FieldsPopover'
import type { GroupBy } from '../components/GroupPopover'
import type { SortBy, SortDir } from '../components/SortPopover'
import Icon from '../components/Icon'
import QuestSummary, { splitQuestSummary } from '../components/QuestSummary'
import { fmtBlockedReason, hasApplyFailed, relativeTime, shortId, typeLabel } from '../components/util'
import { questTurnUsageLabel } from '../domain/questSelectors'
import QuestCard from './QuestCard'
import FeedView from './FeedView'
import { QuestLoadingState, QuestEmptyState } from './QuestStates'
import { useTranslation } from 'react-i18next'
import {
  buildGroups,
  buildPhaseLanes,
  failureSummary,
  GROUP_MOBILE_ORDER,
  MOBILE_ORDER,
  questActivityTime,
  sortQuests,
  truncateText,
} from './questsBoardHelpers'

type Props = {
  quests: QuestMeta[]
  inboxItems?: QuestMeta[]
  humanExceptions?: HumanExceptionItem[]
  loading?: boolean
  view: 'feed' | 'board' | 'list'
  onOpen: (qid: string) => void
  groupBy: GroupBy
  sortBy: SortBy
  sortDir: SortDir
  visibleFields: Set<FieldKey>
  executors: ExecutorInfo[]
  adventurers: AdventurerFile[]
  onCreateQuest: () => void
  onConfigureAgents: () => void
  onCreateAdventurer: () => void
  onOpenAdventurer: (advId: string) => void
  onOpenInbox?: () => void
  onSwitchView?: (view: 'feed' | 'board' | 'list') => void
  activeProject?: string
  feedSearchQuery?: string
}

export default function QuestsBoard({
  quests,
  inboxItems = [],
  humanExceptions = [],
  loading,
  view,
  onOpen,
  groupBy,
  sortBy,
  sortDir,
  visibleFields,
  executors,
  adventurers,
  onCreateQuest,
  onConfigureAgents,
  onCreateAdventurer,
  onOpenAdventurer,
  onOpenInbox,
  onSwitchView,
  activeProject,
  feedSearchQuery,
}: Props) {
  const { t } = useTranslation()
  const sorted = sortQuests(quests, sortBy, sortDir)
  const groups = buildGroups(sorted, groupBy, view)
  const attentionCount = sorted.filter(
    (q) => q.status === 'user_review' || q.status === 'waiting_input' || q.status === 'blocked' || hasApplyFailed(q)
  ).length
  // failed/cancelled 是终态历史，默认折叠让首页 focus 在需处理的
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(new Set(['failed', 'cancelled']))

  function toggleGroup(key: string) {
    setCollapsedGroups((prev) => {
      const next = new Set(prev)
      if (next.has(key)) next.delete(key)
      else next.add(key)
      return next
    })
  }

  // Loading / empty states
  if (loading && sorted.length === 0) {
    return <QuestLoadingState />
  }
  if (!loading && sorted.length === 0) {
    return (
      <QuestEmptyState
        executors={executors}
        adventurers={adventurers}
        onCreateQuest={onCreateQuest}
        onConfigureAgents={onConfigureAgents}
        onCreateAdventurer={onCreateAdventurer}
      />
    )
  }

  // Feed view (首页循环运行反馈流)
  if (view === 'feed') {
    return (
      <FeedView
        sorted={sorted}
        inboxItems={inboxItems}
        humanExceptions={humanExceptions}
        onOpen={onOpen}
        onCreateQuest={onCreateQuest}
        onOpenInbox={onOpenInbox}
        onOpenAdventurer={onOpenAdventurer}
        activeProject={activeProject}
        feedSearchQuery={feedSearchQuery}
      />
    )
  }

  // Board view (classic kanban)
  if (view === 'board') {
    const isStatusGroup = groupBy === 'status'
    if (groupBy === 'phase') {
      const lanes = buildPhaseLanes(sorted)
      return (
        <div className="phase-board-scroll">
          <div className="phase-board">
            {lanes.map((lane, laneIndex) => (
              <section className={'phase-lane' + (lane.legacy ? ' legacy' : '') + (laneIndex >= 3 ? ' collapsed-lite' : '')} key={lane.key}>
                <header className="phase-lane-head">
                  <div>
                    <h2>{lane.title}</h2>
                    <span>{lane.subtitle}</span>
                  </div>
                  {laneIndex >= 3 && <em>{t('quest.board.collapsedLowPriority')}</em>}
                </header>
                <div className="phase-lane-columns">
                  {lane.columns.map((column) => (
                    <section className={'board-column phase-column role-' + (column.role || 'unknown')} key={column.key}>
                      <header>
                        <span className={'column-dot dot-' + (column.role === 'review' ? 'reviewing' : column.role === 'execute' ? 'running' : 'pending')} />
                        <h2>{column.title}</h2>
                        <span className="mono">{column.items.length}</span>
                      </header>
                      <div className="column-stack">
                        {column.items.map((quest) => (
                          <QuestCard key={quest.id} quest={quest} onOpen={onOpen} visibleFields={visibleFields} />
                        ))}
                      </div>
                    </section>
                  ))}
                </div>
              </section>
            ))}
          </div>
        </div>
      )
    }
    return (
      <div className="board-scroll">
        <div className="board">
          {isStatusGroup && attentionCount > 0 && (
            <div className="mobile-section-label attention">
              <Icon name="bell-ring" />
              <h2>{t('quest.board.needAttention')}</h2>
              <span className="mono">{attentionCount}</span>
            </div>
          )}
          {groups.map((group) => {
            const wide = isStatusGroup && (group.key === 'user_review' || group.key === 'waiting_input' || group.key === 'apply_failed')
            return (
              <section
                className={'board-column' + (wide ? ' wide' : '')}
                style={isStatusGroup ? { ['--mobile-order' as string]: GROUP_MOBILE_ORDER[group.key] ?? MOBILE_ORDER[group.key as QuestStatus] ?? 10 } : undefined}
                key={group.key}
              >
                <header>
                  {group.dot && <span className={'column-dot dot-' + group.dot} />}
                  <h2>{group.title}</h2>
                  <span className="mono">{group.items.length}</span>
                </header>
                <div className="column-stack">
                  {group.items.map((quest) => (
                    <QuestCard key={quest.id} quest={quest} onOpen={onOpen} visibleFields={visibleFields} />
                  ))}
                </div>
              </section>
            )
          })}
        </div>
      </div>
    )
  }

  // List view (preserved)
  return (
    <div className="list-table">
      {sorted.length > 0 && groups.map((group) => (
        <div className={'list-group' + (group.attention ? ' list-group-attention' : '')} key={group.key}>
          <header className="list-group-head" onClick={() => toggleGroup(group.key)} role="button" tabIndex={0} onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggleGroup(group.key) } }} aria-expanded={!collapsedGroups.has(group.key)}>
            {group.dot && <span className={'column-dot dot-' + group.dot} />}
            <h3>{group.title}</h3>
            <span className="mono">{group.items.length}</span>
            {group.attention && <span className="attention-chip">{t('quest.board.attentionChip')}</span>}
            <Icon name="chevron-down" size={14} className={'list-fold-icon' + (collapsedGroups.has(group.key) ? ' folded' : '')} />
          </header>
          {!collapsedGroups.has(group.key) && (
          <div className="list-group-wrap">
            {group.items.map((quest) => {
              const applyFailed = hasApplyFailed(quest)
              const hasReviewScore = quest.status === 'user_review' && quest.mage_score != null
              return (
                <button className="quest-row" key={quest.id} onClick={() => onOpen(quest.id)} title={quest.query}>
                  <div className="quest-row-left">
                    <span className="mono quest-row-id">#{quest.short_id || shortId(quest.id)}</span>
                    {visibleFields.has('type') && (
                      <span className="quest-row-type">{typeLabel(quest.type)}</span>
                    )}
                    <QuestSummary query={quest.query} variant="row" />
                  </div>
                  <div className="quest-row-right">
                    {visibleFields.has('statusHint') && (
                      <>
                        {hasReviewScore && (
                          <span className={'row-status-hint ' + (quest.final_verdict === 'pass' ? 'success' : quest.final_verdict === 'reject' ? 'danger' : 'amber')}>
                            <Icon name={quest.final_verdict === 'pass' ? 'circle-check' : quest.final_verdict === 'reject' ? 'x' : 'rotate-ccw'} size={13} />
                            {quest.final_verdict === 'pass' ? t('quest.row.reviewPass') : quest.final_verdict === 'reject' ? t('quest.row.reviewReject') : t('quest.row.reviewNeedChanges')} {quest.mage_score}/10
                          </span>
                        )}
                        {quest.status === 'user_review' && !hasReviewScore && (
                          <span className="row-status-hint amber">
                            <Icon name="bell-ring" size={12} />
                            {t('quest.row.needHuman')}
                          </span>
                        )}
                        {quest.status === 'reviewing' && (
                          <span className="row-status-hint violet">
                            <Icon name="wand-sparkles" size={12} />
                            {t('quest.row.mageReviewing')}
                          </span>
                        )}
                        {quest.status === 'blocked' && (
                          <span className="row-status-hint danger">
                            <Icon name="triangle-alert" size={13} />
                            {failureSummary(quest)?.title || fmtBlockedReason(quest) || t('quest.row.blockedFallback')}
                          </span>
                        )}
                        {quest.status === 'running' && (
                          <span className="row-status-hint running">
                            <Icon name="play" size={12} />
                            {t('quest.row.running')} · {questTurnUsageLabel(quest)}
                          </span>
                        )}
                        {quest.status === 'success' && quest.applied && (
                          <span className="row-status-hint success">
                            <Icon name="git-merge" size={12} />
                            {t('quest.row.applied')}
                          </span>
                        )}
                        {applyFailed && (
                          <span className="row-status-hint danger">
                            <Icon name="triangle-alert" size={13} />
                            {t('quest.row.applyFailed')}
                          </span>
                        )}
                      </>
                    )}
                    {visibleFields.has('mageScore') && quest.mage_score != null && (
                      <span className="row-status-hint muted">
                        {t('quest.row.mageScore', { score: quest.mage_score })}
                      </span>
                    )}
                    {visibleFields.has('adventurer') && (
                      <span className="row-status-hint muted">
                        <Icon name="sword" size={12} />
                        {quest.warrior_id || quest.mage_id || t('quest.row.warriorFallback')}
                      </span>
                    )}
                    {visibleFields.has('reworkCount') && quest.rework_count > 0 && (
                      <span className="row-status-hint muted">
                        {t('quest.row.rework', { count: quest.rework_count })}
                      </span>
                    )}
                    {visibleFields.has('diffStat') && quest.diff_changed_files != null && (
                      <span className="row-status-hint muted">
                        {t('quest.row.files', { count: quest.diff_changed_files })}
                      </span>
                    )}
                    {visibleFields.has('applied') && quest.applied && (
                      <span className="row-status-hint success">
                        <Icon name="git-merge" size={12} />
                        {t('quest.row.applied')}
                      </span>
                    )}
                    {visibleFields.has('time') && (
                      <span className="mono row-time">{relativeTime(questActivityTime(quest))}</span>
                    )}
                  </div>
                </button>
              )
            })}
          </div>
          )}
        </div>
      ))}
    </div>
  )
}
