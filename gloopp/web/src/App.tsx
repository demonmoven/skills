import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type {
  AutomationConfig,
  PersonalDrilldown,
  QuestStatus,
  QuestType,
} from './api/types'
import AppSidebar from './components/AppSidebar'
import AppTopbar from './components/AppTopbar'
import CommandPalette from './components/CommandPalette'
import CreateAdventurerSheet from './components/CreateAdventurerSheet'
import CreateAutomationSheet from './components/CreateAutomationSheet'
import CreateQuestSheet from './components/CreateQuestSheet'
import ErrorBoundary from './components/ErrorBoundary'
import type { FieldKey } from './components/FieldsPopover'
import type { GroupBy } from './components/GroupPopover'
import Icon from './components/Icon'
import QuestFilterBar from './components/QuestFilterBar'
import SideDrawer from './components/SideDrawer'
import type { SortBy, SortDir } from './components/SortPopover'
import UpgradeBanner from './components/UpgradeBanner'
import i18n from './i18n'
import { formatPageTitle } from './pageTitle'
import { hasApplyFailed } from './components/util'
import QuestsBoard from './pages/QuestsBoard'
import { useDashboardData, type QuestRange } from './hooks/useDashboardData'
import { useGlobalQuestStream } from './hooks/useGlobalQuestStream'
import { useRouting } from './hooks/useRouting'
import { useToasts } from './hooks/useToasts'
import { buildCommandItems, type CommandPaletteTab } from './features/commandPalette/buildCommandItems'

type Tab = 'quests' | 'inbox' | 'automations' | 'knowledge' | 'adventurers' | 'resources' | 'stats' | 'settings'
type View = 'feed' | 'board' | 'list'
type ResourceTab = 'skills' | 'prompts'

const Adventurers = lazy(() => import('./pages/Adventurers'))
const Automations = lazy(() => import('./pages/Automations'))
const Inbox = lazy(() => import('./pages/Inbox'))
const Knowledge = lazy(() => import('./pages/Knowledge'))
const QuestDetail = lazy(() => import('./pages/QuestDetail'))
const Resources = lazy(() => import('./pages/Resources'))
const Settings = lazy(() => import('./pages/Settings'))
const Stats = lazy(() => import('./pages/Stats'))
const ThreadView = lazy(() => import('./pages/ThreadView'))

const NAV: { key: Tab; label: string; icon: string }[] = [
  { key: 'quests', label: i18n.t('nav.quests'), icon: 'grid' },
  { key: 'inbox', label: i18n.t('nav.inbox'), icon: 'inbox' },
  { key: 'automations', label: i18n.t('nav.automations'), icon: 'bolt' },
  { key: 'knowledge', label: i18n.t('nav.knowledge'), icon: 'file-text' },
  { key: 'adventurers', label: i18n.t('nav.adventurers'), icon: 'users' },
  { key: 'resources', label: i18n.t('nav.resources'), icon: 'package' },
  { key: 'stats', label: i18n.t('nav.stats'), icon: 'board' },
  { key: 'settings', label: i18n.t('nav.settings'), icon: 'settings' },
]

const MOBILE_NAV: { key: Tab; label: string; icon: string }[] = NAV

type SheetKind = 'quest' | 'automation' | 'adventurer' | null

const PAGE_TITLES: Record<Tab, string> = {
  quests: i18n.t('nav.quests'),
  inbox: i18n.t('nav.inbox'),
  automations: i18n.t('nav.automations'),
  knowledge: i18n.t('nav.knowledge'),
  adventurers: i18n.t('nav.adventurers'),
  resources: i18n.t('nav.resources'),
  stats: i18n.t('nav.stats'),
  settings: i18n.t('nav.settings'),
}

export default function App() {
  const { tab, questId: openQid, setTab, setQuestId: setOpenQid } = useRouting()
  const [view, setView] = useState<View>(() => {
    try {
      const saved = localStorage.getItem('quests-view')
      if (saved === 'board' || saved === 'list') return saved
    } catch { /* ignore */ }
    return 'feed' // 旧用户存过 'focus' 的也 fallback 到 feed
  })
  const [sheet, setSheet] = useState<SheetKind>(null)
  const [paletteOpen, setPaletteOpen] = useState(false)
  const [automationDraft, setAutomationDraft] = useState<Partial<AutomationConfig> | null>(null)
  const [activeResourceTab, setActiveResourceTab] = useState<ResourceTab>('skills')
  const { toasts, showError, showSuccess } = useToasts()

  // Selection state for master-detail pages
  const [searchQuery, setSearchQuery] = useState('')
  const [activeProject, setActiveProject] = useState('')
  const [feedSearchQuery, setFeedSearchQuery] = useState('')
  const [topbarHidden, setTopbarHidden] = useState(false)
  const [questViewMode, setQuestViewMode] = useState<'thread' | 'detail'>('thread')
  const contentRef = useRef<HTMLDivElement>(null)
  const lastScrollTop = useRef(0)

  // Quest toolbar state
  const [groupBy, setGroupBy] = useState<GroupBy>('status')
  const [sortBy, setSortBy] = useState<SortBy>('updated_at_ms')
  const [sortDir, setSortDir] = useState<SortDir>('desc')
  const [visibleFields, setVisibleFields] = useState<Set<FieldKey>>(() => new Set(['type', 'statusHint', 'time']))
  const [showFilterBar, setShowFilterBar] = useState(false)
  const [activePopover, setActivePopover] = useState<'group' | 'fields' | 'sort' | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    try {
      return localStorage.getItem('sidebar-collapsed') === '1'
    } catch {
      return false
    }
  })

  useEffect(() => {
    try {
      localStorage.setItem('sidebar-collapsed', sidebarCollapsed ? '1' : '0')
    } catch {
      /* ignore */
    }
  }, [sidebarCollapsed])

  useEffect(() => {
    const mql = window.matchMedia('(max-width: 1280px)')
    const handler = (e: MediaQueryListEvent | MediaQueryList) => {
      setSidebarCollapsed(e.matches)
    }
    handler(mql)
    mql.addEventListener('change', handler)
    return () => mql.removeEventListener('change', handler)
  }, [])

  useEffect(() => {
    const isMobile = window.matchMedia('(max-width: 768px)').matches
    const el = isMobile ? window : contentRef.current
    if (!el) return
    const onScroll = () => {
      const st = isMobile ? window.scrollY : (contentRef.current?.scrollTop ?? 0)
      if (st > lastScrollTop.current && st > 56) {
        setTopbarHidden(true)
      } else if (st < lastScrollTop.current) {
        setTopbarHidden(false)
      }
      lastScrollTop.current = st
    }
    el.addEventListener("scroll", onScroll, { passive: true })
    return () => el.removeEventListener("scroll", onScroll)
  }, [])

  useEffect(() => {
    try {
      localStorage.setItem('quests-view', view)
    } catch {
      /* ignore */
    }
  }, [view])

  const {
    quests,
    questsLoading,
    inbox,
    humanExceptions,
    inboxLoading,
    adventurers,
    automations,
    executors,
    skills,
    prompts,
    stats,
    statsLoading,
    selectedAdv,
    setSelectedAdv,
    selectedAuto,
    setSelectedAuto,
    selectedSkill,
    setSelectedSkill,
    selectedPrompt,
    setSelectedPrompt,
    setAdventurers,
    upsertAutomation,
    questFilters,
    setQuestFilters,
    questRange,
    setQuestRange,
    statsRange,
    setStatsRange,
    loadQuests,
    loadInbox,
    removeInboxItem,
    loadAutomations,
    loadSkills,
    loadPrompts,
    loadAdventurers,
    loadExecutors,
    loadStats,
    loadAll,
    toggleFilterStatus,
    toggleFilterType,
    resetQuestFilters,
  } = useDashboardData({ showError })

  const DEFAULT_FIELDS = useMemo(() => new Set<FieldKey>(['type', 'statusHint', 'time']), [])
  const hasStatsDrilldownFilter =
    questFilters.questIds.length > 0 ||
    questFilters.projects.length > 0 ||
    questFilters.failureReasons.length > 0 ||
    questFilters.failureCategories.length > 0
  const hasFilterValue = questFilters.statuses.length > 0 || questFilters.types.length > 0 || hasStatsDrilldownFilter || questRange !== '7d'
  const hasSortValue = sortBy !== 'updated_at_ms' || sortDir !== 'desc'
  const hasGroupValue = groupBy !== 'none'
  const hasFieldsValue = visibleFields.size !== DEFAULT_FIELDS.size || [...visibleFields].some(k => !DEFAULT_FIELDS.has(k))

  useEffect(() => {
    loadAll()
  }, [loadAll])

  useEffect(() => {
    if (tab === 'stats') {
      loadStats(statsRange)
    }
  }, [loadStats, statsRange, tab])

  useEffect(() => {
    if (tab === 'automations') {
      loadAutomations()
    } else if (tab === 'resources') {
      loadSkills()
      loadPrompts()
    } else if (tab === 'adventurers' || tab === 'settings') {
      loadAdventurers()
      loadExecutors()
    }
  }, [loadAdventurers, loadAutomations, loadExecutors, loadPrompts, loadSkills, tab])

  useEffect(() => {
    if (tab === 'quests' && !openQid) {
      loadQuests()
    }
  }, [loadQuests, openQid, tab])

  // Reset to ThreadView when opening a different quest (v0.6: thread is default reading surface)
  useEffect(() => {
    if (openQid) {
      setQuestViewMode('thread')
    }
  }, [openQid])

  const streamState = useGlobalQuestStream({
    enabled: !openQid,
    refreshQuests: () => {
      loadQuests()
      loadInbox()
    },
    refreshAutomations: () => {
      loadQuests()
      loadInbox()
      loadAutomations()
      if (tab === 'stats') loadStats()
    },
  })

  function go(next: Tab) {
    setTab(next)
    setSearchQuery('')
  }

  const navigateFromCommand = useCallback((next: CommandPaletteTab) => {
    setSheet(null)
    setSearchQuery('')
    setOpenQid(null)
    setTab(next)
  }, [setOpenQid, setTab])

  const openAgentSettings = useCallback(() => {
    setSheet(null)
    setSearchQuery('')
    setTab('settings')
    window.setTimeout(() => {
      window.location.hash = 'settings-agents'
      document.getElementById('settings-agents')?.scrollIntoView({ block: 'start' })
    }, 0)
  }, [setTab])

  const openPrimary = useCallback(() => {
    // 按当前 tab 打开对应表单
    setAutomationDraft(null)
    if (tab === 'automations') setSheet('automation')
    else if (tab === 'adventurers') setSheet('adventurer')
    else setSheet('quest')
  }, [tab])

  const openAutomationFromCommand = useCallback((id: string) => {
    setSheet(null)
    setSearchQuery('')
    setOpenQid(null)
    setTab('automations')
    setSelectedAuto(id)
  }, [setOpenQid, setSelectedAuto, setTab])

  const openAdventurerFromCommand = useCallback((id: string) => {
    setSheet(null)
    setSearchQuery('')
    setOpenQid(null)
    setTab('adventurers')
    setSelectedAdv(id)
  }, [setOpenQid, setSelectedAdv, setTab])

  const openSkillFromCommand = useCallback((name: string) => {
    setSheet(null)
    setSearchQuery('')
    setOpenQid(null)
    setActiveResourceTab('skills')
    setTab('resources')
    setSelectedSkill(name)
  }, [setOpenQid, setSelectedSkill, setTab])

  const openPromptFromCommand = useCallback((name: string) => {
    setSheet(null)
    setSearchQuery('')
    setOpenQid(null)
    setActiveResourceTab('prompts')
    setTab('resources')
    setSelectedPrompt(name)
  }, [setOpenQid, setSelectedPrompt, setTab])

  const paletteItems = useMemo(() => buildCommandItems({
    quests,
    inbox,
    humanExceptions,
    automations,
    adventurers,
    skills,
    prompts,
    actions: {
      navigate: navigateFromCommand,
      createQuest: () => {
        setAutomationDraft(null)
        setOpenQid(null)
        setSearchQuery('')
        setTab('quests')
        setSheet('quest')
      },
      createAutomation: () => {
        setAutomationDraft(null)
        setOpenQid(null)
        setSearchQuery('')
        setTab('automations')
        setSheet('automation')
      },
      createAdventurer: () => {
        setOpenQid(null)
        setSearchQuery('')
        setTab('adventurers')
        setSheet('adventurer')
      },
      openQuest: (qid) => {
        setSheet(null)
        setSearchQuery('')
        setTab('quests')
        setOpenQid(qid)
      },
      openAutomation: openAutomationFromCommand,
      openAdventurer: openAdventurerFromCommand,
      openSkill: openSkillFromCommand,
      openPrompt: openPromptFromCommand,
    },
  }), [
    adventurers,
    automations,
    humanExceptions,
    inbox,
    navigateFromCommand,
    openAdventurerFromCommand,
    openAutomationFromCommand,
    openPromptFromCommand,
    openSkillFromCommand,
    prompts,
    quests,
    setOpenQid,
    setTab,
    skills,
  ])

  // Command Palette: Cmd/Ctrl+K global shortcut
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setPaletteOpen((v) => !v)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  const attentionCount = quests.filter((q) => q.status === 'user_review' || q.status === 'waiting_input' || q.status === 'blocked' || hasApplyFailed(q)).length
  const feedProjects = useMemo(() => Array.from(new Set(quests.map((q) => q.base_working_dir?.split('/').pop()).filter((p): p is string => !!p))).sort(), [quests])
  const openQuest = openQid ? quests.find((q) => q.id === openQid) : null
  const navCount = useCallback(
    (key: Tab) => {
      if (key === 'quests') return attentionCount
      if (key === 'inbox') return inbox.length
      return 0
    },
    [attentionCount, inbox.length],
  )

  const title = openQid
    ? i18n.t('nav.questDetail')
    : tab === 'quests'
      ? (view === 'feed' ? i18n.t('nav.quests') : view === 'board' ? i18n.t('nav.questBoard') : i18n.t('nav.questList'))
      : NAV.find((n) => n.key === tab)?.label || PAGE_TITLES[tab]
  const browserTitle = openQid
    ? formatPageTitle(i18n.t('nav.quests') + ' #' + (openQuest?.short_id || openQid.slice(0, 8)))
    : formatPageTitle(PAGE_TITLES[tab])

  useEffect(() => {
    document.title = browserTitle
  }, [browserTitle])

  const showSearchHeader =
    !openQid && (tab === 'inbox' || tab === 'automations' || tab === 'adventurers' || tab === 'resources')
  const showStatsToolbar = !openQid && tab === 'stats'

  const openContextQuest = useCallback((qid: string) => {
    if (!qid) return
    setTab('quests')
    setOpenQid(qid)
  }, [setTab, setOpenQid])

  const openAdventurer = useCallback((advId: string) => {
    if (!advId) return
    setTab('adventurers')
    setSelectedAdv(advId)
  }, [setTab, setSelectedAdv])

  const openStatsDrilldown = useCallback((drilldown: PersonalDrilldown, _label?: string) => {
    const query = drilldown.query || {}
    setOpenQid(null)
    setTab('quests')
    setView('list')
    setGroupBy('none')
    setShowFilterBar(true)
    setQuestRange(statsRange)
    setQuestFilters({
      statuses: (query.status || []).filter(Boolean) as QuestStatus[],
      types: (query.type || []).filter(Boolean) as QuestType[],
      questIds: (query.quest_id || []).filter(Boolean),
      projects: (query.project || []).filter(Boolean),
      failureReasons: (query.failure_reason || []).filter(Boolean),
      failureCategories: (query.failure_category || []).filter(Boolean),
    })
  }, [setQuestFilters, setQuestRange, setOpenQid, setTab, statsRange])

  const body = useMemo(() => {
    if (openQid) {
      const threadView = (
        <ThreadView
          qid={openQid}
          onBack={() => setOpenQid(null)}
          onOpenThread={(qid) => {
            setOpenQid(qid)
            setQuestViewMode('thread')
          }}
          onOpenDetail={(qid) => {
            setOpenQid(qid)
            setQuestViewMode('detail')
          }}
          onError={showError}
          onChanged={loadAll}
        />
      )
      if (questViewMode === 'detail') {
        return (
          <div className="quest-drawer-layout">
            <div className="quest-drawer-backdrop" onClick={() => setQuestViewMode('thread')} />
            {threadView}
            <div className="quest-drawer-panel">
              <div className="quest-drawer-header">
                <button className="icon-button" onClick={() => setQuestViewMode('thread')} aria-label="返回 Thread">
                  <Icon name="x" size={18} />
                </button>
                <span>管理面板</span>
              </div>
              <div className="quest-drawer-body">
                <QuestDetail
                  qid={openQid}
                  onBack={() => setQuestViewMode('thread')}
                  onError={showError}
                  onChanged={loadAll}
                  onOpenQuest={setOpenQid}
                />
              </div>
            </div>
          </div>
        )
      }
      return threadView
    }
    if (tab === 'quests') {
      return (
        <QuestsBoard
          quests={quests}
          inboxItems={inbox}
          humanExceptions={humanExceptions}
          loading={questsLoading}
          view={view}
          onOpen={setOpenQid}
          groupBy={groupBy}
          sortBy={sortBy}
          sortDir={sortDir}
          visibleFields={visibleFields}
          executors={executors}
          adventurers={adventurers}
          onCreateQuest={() => setSheet('quest')}
          onConfigureAgents={openAgentSettings}
          onCreateAdventurer={() => {
            setTab('adventurers')
            setSheet('adventurer')
          }}
          onOpenAdventurer={openAdventurer}
          onOpenInbox={() => setTab('inbox')}
          onSwitchView={setView}
          activeProject={activeProject}
          feedSearchQuery={feedSearchQuery}
        />
      )
    }
    if (tab === 'inbox') {
      return (
        <Inbox
          items={inbox}
          humanExceptions={humanExceptions}
          loading={inboxLoading}
          searchQuery={searchQuery}
          onOpen={setOpenQid}
          onError={showError}
          onChanged={loadAll}
          onRemoveItem={removeInboxItem}
        />
      )
    }
    if (tab === 'automations') {
      return (
        <Automations
          items={automations}
          selected={selectedAuto}
          onSelect={setSelectedAuto}
          onError={showError}
          onChanged={loadAll}
          onAutomationUpdated={upsertAutomation}
          adventurers={adventurers}
          executors={executors}
          searchQuery={searchQuery}
          onUseTemplate={(template) => {
            setAutomationDraft({
              ...template,
              id: undefined,
              source: 'user',
              name: template.name ? template.name + ' ' + i18n.t('app.copySuffix') : '',
              enabled: true,
              tags: (template.tags || []).filter((t) => t !== 'official' && t !== 'template'),
            })
            setSheet('automation')
          }}
        />
      )
    }
    if (tab === 'adventurers') {
      return (
        <Adventurers
          items={adventurers}
          executors={executors}
          selected={selectedAdv}
          onSelect={setSelectedAdv}
          onError={showError}
          onSuccess={showSuccess}
          onChanged={loadAll}
          searchQuery={searchQuery}
          onOpenQuest={setOpenQid}
        />
      )
    }
    if (tab === 'resources') {
      return (
        <Resources
          skills={skills}
          prompts={prompts}
          selectedSkill={selectedSkill}
          selectedPrompt={selectedPrompt}
          onSelectSkill={setSelectedSkill}
          onSelectPrompt={setSelectedPrompt}
          onError={showError}
          searchQuery={searchQuery}
          activeResourceTab={activeResourceTab}
          onResourceTabChange={setActiveResourceTab}
        />
      )
    }
    if (tab === 'stats') {
      return (
        <Stats
          stats={stats}
          loading={statsLoading}
          range={statsRange}
          onOpenQuest={openContextQuest}
          onDrilldown={openStatsDrilldown}
        />
      )
    }
    if (tab === 'knowledge') {
      return (
        <Knowledge
          onError={showError}
          onOpenQuest={openContextQuest}
        />
      )
    }
    return (
      <Settings
        adventurers={adventurers}
        onError={showError}
        onSuccess={showSuccess}
        onChanged={loadAll}
      />
    )
  }, [
    adventurers,
    automations,
    executors,
    groupBy,
    humanExceptions,
    inbox,
    inboxLoading,
    loadAll,
    loadStats,
    openAdventurer,
    openContextQuest,
    openStatsDrilldown,
    openQid,
    quests,
    questsLoading,
    removeInboxItem,
    searchQuery,
    selectedAdv,
    selectedAuto,
    selectedSkill,
    selectedPrompt,
    showError,
    skills,
    prompts,
    sortBy,
    sortDir,
    stats,
    statsRange,
    tab,
    upsertAutomation,
    view,
    activeProject,
    feedSearchQuery,
    visibleFields,
    questViewMode,
    setQuestViewMode,
  ])

  const lazyBody = (
    <ErrorBoundary>
      <Suspense fallback={<div className="empty-state">{i18n.t('app.loading')}</div>}>
        {body}
      </Suspense>
    </ErrorBoundary>
  )

  // FAB: 根据 tab 显示对应操作；收件箱 / 设置 / 统计不显示 FAB
  const fabHidden =
    !!openQid || tab === 'inbox' || tab === 'stats' || tab === 'settings' || tab === 'resources' || tab === 'knowledge'
  const fabText =
    tab === 'automations'
      ? i18n.t('fab.automation')
      : tab === 'adventurers'
        ? i18n.t('fab.adventurer')
        : i18n.t('fab.quest')

  return (
    <div className="app-shell">
      <AppSidebar
        items={NAV}
        currentTab={tab}
        questOpen={!!openQid}
        collapsed={sidebarCollapsed}
        version={__GLOOP_VERSION__}
        navCount={navCount}
        onNavigate={go}
        onToggleCollapsed={() => setSidebarCollapsed((v) => !v)}
      />

      <main className="main-panel">
        <UpgradeBanner />
        <AppTopbar
          tab={tab}
          title={title}
          questOpen={!!openQid}
          view={view}
          searchQuery={searchQuery}
          streamState={streamState}
          statsRange={statsRange}
          activePopover={activePopover}
          groupBy={groupBy}
          sortBy={sortBy}
          sortDir={sortDir}
          visibleFields={visibleFields}
          showSearchHeader={showSearchHeader}
          showStatsToolbar={showStatsToolbar}
          fabHidden={fabHidden}
          hasGroupValue={hasGroupValue}
          hasFieldsValue={hasFieldsValue}
          hasFilterValue={hasFilterValue}
          hasSortValue={hasSortValue}
          onOpenDrawer={() => setDrawerOpen(true)}
          onSearchQueryChange={setSearchQuery}
          onStatsRangeChange={setStatsRange}
          onActivePopoverChange={setActivePopover}
          onGroupByChange={setGroupBy}
          onSortByChange={setSortBy}
          onSortDirChange={setSortDir}
          onVisibleFieldsChange={setVisibleFields}
          onToggleFilterBar={() => {
            setShowFilterBar(!showFilterBar)
            setActivePopover(null)
          }}
          onViewChange={setView}
          onRefreshStats={() => loadStats()}
          onPrimaryAction={openPrimary}
          feedProjects={feedProjects}
          activeProject={activeProject}
          onActiveProjectChange={setActiveProject}
          feedSearchQuery={feedSearchQuery}
          onFeedSearchChange={setFeedSearchQuery}
          topbarHidden={topbarHidden}
        />

        {showFilterBar && tab === 'quests' && !openQid && (
          <QuestFilterBar
            questRange={questRange}
            questFilters={questFilters}
            hasStatsDrilldownFilter={hasStatsDrilldownFilter}
            hasFilterValue={hasFilterValue}
            onQuestRangeChange={setQuestRange}
            onToggleStatus={toggleFilterStatus}
            onToggleType={toggleFilterType}
            onReset={resetQuestFilters}
          />
        )}

        <div className="content" ref={contentRef}>{lazyBody}</div>
      </main>

      {!fabHidden && (
        <button className="fab" onClick={openPrimary} aria-label={fabText}>
          <Icon name="plus" />
          <span>{fabText}</span>
        </button>
      )}

      <SideDrawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        onOpen={() => setDrawerOpen(true)}
        currentTab={tab}
        items={MOBILE_NAV.map((item) => ({ ...item, count: navCount(item.key as Tab) }))}
        onNavigate={go}
      />

      {sheet === 'quest' && (
        <CreateQuestSheet
          adventurers={adventurers}
          executors={executors}
          onClose={() => setSheet(null)}
          onError={showError}
          onConfigureAgents={openAgentSettings}
          onCreated={(quest) => {
            setSheet(null)
            loadAll()
            setTab('quests')
            setOpenQid(quest.id)
          }}
          onAdventurerCreated={(adv) => {
            setAdventurers((prev) => {
              const next = prev.filter((item) => item.id !== adv.id)
              return [adv, ...next]
            })
            setSelectedAdv(adv.id)
          }}
        />
      )}
      {sheet === 'automation' && (
        <CreateAutomationSheet
          adventurers={adventurers}
          executors={executors}
          initial={automationDraft || undefined}
          onClose={() => {
            setSheet(null)
            setAutomationDraft(null)
          }}
          onError={showError}
          onCreated={(auto) => {
            setSheet(null)
            setAutomationDraft(null)
            loadAll()
            setTab('automations')
            setSelectedAuto(auto.id)
          }}
        />
      )}
      {sheet === 'adventurer' && (
        <CreateAdventurerSheet
          executors={executors}
          onClose={() => setSheet(null)}
          onError={showError}
          onConfigureAgents={openAgentSettings}
          onCreated={async (adv) => {
            setSheet(null)
            setSelectedAdv(adv.id)
            await loadAll()
            setTab('adventurers')
          }}
        />
      )}
      {toasts.length > 0 && (
        <div className="toast-stack">
          {toasts.map((t) => (
            <div key={t.id} className={'toast toast-' + t.kind}>
              <Icon name={t.kind === 'error' ? 'error' : t.kind === 'success' ? 'check' : 'info'} />
              <span>{t.msg}</span>
            </div>
          ))}
        </div>
      )}

      <CommandPalette
        open={paletteOpen}
        onClose={() => setPaletteOpen(false)}
        items={paletteItems}
      />
    </div>
  )
}
