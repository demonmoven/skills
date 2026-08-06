import { useTranslation } from 'react-i18next'
import type { QuestRange } from '../hooks/useDashboardData'
import type { FieldKey } from './FieldsPopover'
import FieldsPopover from './FieldsPopover'
import type { GroupBy } from './GroupPopover'
import GroupPopover from './GroupPopover'
import Icon from './Icon'
import type { SortBy, SortDir } from './SortPopover'
import SortPopover from './SortPopover'

export type AppView = 'feed' | 'board' | 'list'
export type AppTopbarTab = 'quests' | 'inbox' | 'automations' | 'knowledge' | 'adventurers' | 'resources' | 'stats' | 'settings'
export type AppTopbarPopover = 'group' | 'fields' | 'sort' | null

type Props = {
  tab: AppTopbarTab
  title: string
  questOpen: boolean
  view: AppView
  searchQuery: string
  streamState: string
  statsRange: QuestRange
  activePopover: AppTopbarPopover
  groupBy: GroupBy
  sortBy: SortBy
  sortDir: SortDir
  visibleFields: Set<FieldKey>
  showSearchHeader: boolean
  showStatsToolbar: boolean
  fabHidden: boolean
  hasGroupValue: boolean
  hasFieldsValue: boolean
  hasFilterValue: boolean
  hasSortValue: boolean
  onOpenDrawer: () => void
  onSearchQueryChange: (query: string) => void
  onStatsRangeChange: (range: QuestRange) => void
  onActivePopoverChange: (popover: AppTopbarPopover) => void
  onGroupByChange: (groupBy: GroupBy) => void
  onSortByChange: (sortBy: SortBy) => void
  onSortDirChange: (sortDir: SortDir) => void
  onVisibleFieldsChange: (fields: Set<FieldKey>) => void
  onToggleFilterBar: () => void
  onViewChange: (view: AppView) => void
  onRefreshStats: () => void
  onPrimaryAction: () => void
  feedProjects?: string[]
  activeProject?: string
  onActiveProjectChange?: (project: string) => void
  feedSearchQuery?: string
  onFeedSearchChange?: (query: string) => void
  topbarHidden?: boolean
}

function searchPlaceholder(tab: AppTopbarTab, t: (key: string, opts?: Record<string, unknown>) => string): string {
  const targetKey =
    tab === 'inbox' ? 'topbar.search.target.inbox'
      : tab === 'automations' ? 'topbar.search.target.automations'
        : tab === 'adventurers' ? 'topbar.search.target.adventurers'
          : tab === 'resources' ? 'topbar.search.target.resources'
            : ''
  if (!targetKey) return t('topbar.search.quest')
  return t('topbar.search.placeholder', { target: t(targetKey) })
}

function primaryActionLabel(tab: AppTopbarTab, t: (key: string) => string): string {
  if (tab === 'automations') return t('topbar.action.newAutomation')
  if (tab === 'adventurers') return t('topbar.action.recruitAdventurer')
  return t('topbar.action.createQuest')
}

export default function AppTopbar({
  tab,
  title,
  questOpen,
  view,
  searchQuery,
  streamState,
  statsRange,
  activePopover,
  groupBy,
  sortBy,
  sortDir,
  visibleFields,
  showSearchHeader,
  showStatsToolbar,
  fabHidden,
  hasGroupValue,
  hasFieldsValue,
  hasFilterValue,
  hasSortValue,
  onOpenDrawer,
  onSearchQueryChange,
  onStatsRangeChange,
  onActivePopoverChange,
  onGroupByChange,
  onSortByChange,
  onSortDirChange,
  onVisibleFieldsChange,
  onToggleFilterBar,
  onViewChange,
  onRefreshStats,
  onPrimaryAction,
  feedProjects,
  activeProject,
  onActiveProjectChange,
  feedSearchQuery,
  onFeedSearchChange,
  topbarHidden,
}: Props) {
  const { t } = useTranslation()
  return (
    <header
      className={
        'topbar' +
        (showSearchHeader ? ' search-topbar' : '') +
        (showStatsToolbar ? ' stats-topbar' : '') +
        (topbarHidden ? ' topbar-hidden' : '')
      }
    >
      <div className="mobile-brand">
        <button className="hamburger-button" onClick={onOpenDrawer} aria-label={t('topbar.menu.open')}>
          <Icon name="menu" />
        </button>
      </div>
      {showStatsToolbar ? (
        <div className="segmented wide range-segmented" aria-label={t('topbar.stats.rangeLabel')}>
          {(['today', '7d', '30d', 'all'] as const).map((r) => (
            <button
              key={r}
              type="button"
              className={statsRange === r ? 'active' : ''}
              onClick={() => onStatsRangeChange(r)}
            >
              {t(`topbar.stats.range.${r}`)}
            </button>
          ))}
        </div>
      ) : tab === 'quests' && !questOpen ? (
        <>
          {view === 'feed' && feedProjects && feedProjects.length > 0 && onActiveProjectChange ? (
            <div className="topbar-project-tabs">
              <button
                className={'topbar-project-tab' + (!activeProject ? ' active' : '')}
                onClick={() => onActiveProjectChange('')}
                type="button"
              >
                {t('common.action.all')}
              </button>
              {feedProjects.map((p) => (
                <button
                  key={p}
                  className={'topbar-project-tab' + (activeProject === p ? ' active' : '')}
                  onClick={() => onActiveProjectChange(p)}
                  type="button"
                >
                  {p}
                </button>
              ))}
            </div>
          ) : (
            <h1 className="mobile-only">{title}</h1>
          )}
          {view === 'feed' && onFeedSearchChange && (
            <div className="search-placeholder desktop-only">
              <Icon name="search" />
              <input
                type="text"
                placeholder={t('topbar.search.quest')}
                value={feedSearchQuery || ''}
                onChange={(e) => onFeedSearchChange(e.target.value)}
              />
              {feedSearchQuery && (
                <button className="search-clear" onClick={() => onFeedSearchChange('')}>
                  <Icon name="x" size={14} />
                </button>
              )}
            </div>
          )}
          {view !== 'feed' && (
            <div className="topbar-tools desktop-only">
              <div className="popover-anchor">
                <button className={'topbar-tool' + (hasGroupValue ? ' has-value' : '')} onClick={() => onActivePopoverChange(activePopover === 'group' ? null : 'group')}>
                  <Icon name="layers" /><span>{t('topbar.tools.group')}</span>
                </button>
                <GroupPopover open={activePopover === 'group'} onClose={() => onActivePopoverChange(null)} value={groupBy} onChange={onGroupByChange} />
              </div>
              <div className="popover-anchor">
                <button className={'topbar-tool' + (hasFieldsValue ? ' has-value' : '')} onClick={() => onActivePopoverChange(activePopover === 'fields' ? null : 'fields')}>
                  <Icon name="columns" /><span>{t('topbar.tools.fields')}</span>
                </button>
                <FieldsPopover open={activePopover === 'fields'} onClose={() => onActivePopoverChange(null)} visible={visibleFields} onChange={onVisibleFieldsChange} />
              </div>
              <button className={'topbar-tool' + (hasFilterValue ? ' has-value' : '')} onClick={onToggleFilterBar}>
                <Icon name="sliders-horizontal" /><span>{t('topbar.tools.filter')}</span>
              </button>
              <div className="popover-anchor">
                <button className={'topbar-tool' + (hasSortValue ? ' has-value' : '')} onClick={() => onActivePopoverChange(activePopover === 'sort' ? null : 'sort')}>
                  <Icon name="arrow-up-down" /><span>{t('topbar.tools.sort')}</span>
                </button>
                <SortPopover open={activePopover === 'sort'} onClose={() => onActivePopoverChange(null)} sortBy={sortBy} sortDir={sortDir} onChangeBy={onSortByChange} onChangeDir={onSortDirChange} />
              </div>
            </div>
          )}
        </>
      ) : questOpen || tab === 'settings' || tab === 'knowledge' ? (
        <h1 className="mobile-only">{title}</h1>
      ) : (
        <div className="search-placeholder">
          <Icon name="search" />
          <input
            type="text"
            placeholder={searchPlaceholder(tab, t)}
            value={searchQuery}
            onChange={(e) => onSearchQueryChange(e.target.value)}
          />
          {searchQuery && (
            <button className="search-clear" onClick={() => onSearchQueryChange('')}>
              <Icon name="x" size={14} />
            </button>
          )}
        </div>
      )}
      <div className="toolbar">
        {!questOpen && streamState !== 'open' && (
          <span className={'stream-indicator stream-' + streamState} title={t('topbar.stream.label', { state: streamState })}>
            <Icon name="signal" />
            {streamState === 'reconnecting' ? t('topbar.stream.reconnecting') : streamState === 'connecting' ? t('topbar.stream.connecting') : streamState === 'disconnected' ? t('topbar.stream.disconnected') : streamState}
          </span>
        )}
        {!questOpen && tab === 'quests' && (
          <div className="segmented" aria-label={t('topbar.view.label')}>
            <button
              className={view === 'feed' ? 'active' : ''}
              onClick={() => onViewChange('feed')}
              aria-label={t('topbar.view.feed')}
            >
              <Icon name="radio" />
            </button>
            <button
              className={'desktop-only' + (view === 'board' ? ' active' : '')}
              onClick={() => onViewChange('board')}
              aria-label={t('topbar.view.board')}
            >
              <Icon name="board" />
            </button>
            <button
              className={view === 'list' ? 'active' : ''}
              onClick={() => onViewChange('list')}
              aria-label={t('topbar.view.list')}
            >
              <Icon name="list" />
            </button>
          </div>
        )}
        {showStatsToolbar && (
          <button className="button ghost" type="button" onClick={onRefreshStats}>
            <Icon name="refresh" />
            {t('common.action.refresh')}
          </button>
        )}
        {!fabHidden && (
          <button
            className={'button' + (view === 'feed' ? ' icon-only' : ' primary')}
            onClick={onPrimaryAction}
            title={primaryActionLabel(tab, t)}
            aria-label={primaryActionLabel(tab, t)}
          >
            <Icon name="plus" />
            {view !== 'feed' && primaryActionLabel(tab, t)}
          </button>
        )}
        {tab === 'settings' && !questOpen && (
          <button
            className="button primary"
            form="settings-form"
            type="submit"
          >
            <Icon name="save" />
            {t('topbar.action.saveSettings')}
          </button>
        )}
      </div>
    </header>
  )
}
