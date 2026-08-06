import { useCallback, useMemo, useState } from 'react'
import { api } from '../api/client'
import i18n from '../i18n'
import type {
  AdventurerFile,
  AutomationConfig,
  ExecutorInfo,
  HumanExceptionItem,
  PersonalStats,
  PromptTemplate,
  QuestMeta,
  QuestStatus,
  QuestType,
  SkillManifest,
} from '../api/types'

export type QuestRange = 'today' | '7d' | '30d' | 'all'

type QuestFilters = {
  statuses: QuestStatus[]
  types: QuestType[]
  questIds: string[]
  projects: string[]
  failureReasons: string[]
  failureCategories: string[]
}

type UseDashboardDataOptions = {
  showError: (msg: string) => void
}

const emptyQuestFilters = (): QuestFilters => ({
  statuses: [],
  types: [],
  questIds: [],
  projects: [],
  failureReasons: [],
  failureCategories: [],
})

function buildQuestQuery(questFilters: QuestFilters, questRange: QuestRange): string {
  const params = new URLSearchParams()
  for (const status of questFilters.statuses) params.append('status', status)
  for (const type of questFilters.types) params.append('type', type)
  for (const questId of questFilters.questIds) params.append('quest_id', questId)
  for (const project of questFilters.projects) params.append('project', project)
  for (const reason of questFilters.failureReasons) params.append('failure_reason', reason)
  for (const category of questFilters.failureCategories) params.append('failure_category', category)
  if (questRange !== 'all') params.set('range', questRange)
  const raw = params.toString()
  return raw ? '?' + raw : ''
}

function useQuestList(showError: (msg: string) => void) {
  const [quests, setQuests] = useState<QuestMeta[]>([])
  const [questsLoading, setQuestsLoading] = useState(true)
  const [questFilters, setQuestFilters] = useState<QuestFilters>(emptyQuestFilters)
  const [questRange, setQuestRange] = useState<QuestRange>('7d')

  const questQuery = useMemo(
    () => buildQuestQuery(questFilters, questRange),
    [questFilters, questRange],
  )

  const loadQuests = useCallback(async () => {
    try {
      const res = await api.get<{ items: QuestMeta[] }>('/api/quests' + questQuery)
      setQuests(res.items || [])
    } catch (e) {
      showError(e instanceof Error ? e.message : i18n.t('dashboard.error.loadQuests'))
    } finally {
      setQuestsLoading(false)
    }
  }, [questQuery, showError])

  const toggleFilterStatus = useCallback((status: QuestStatus) => {
    setQuestFilters((prev) => ({
      ...prev,
      statuses: prev.statuses.includes(status)
        ? prev.statuses.filter((item) => item !== status)
        : [...prev.statuses, status],
    }))
  }, [])

  const toggleFilterType = useCallback((type: QuestType) => {
    setQuestFilters((prev) => ({
      ...prev,
      types: prev.types.includes(type)
        ? prev.types.filter((item) => item !== type)
        : [...prev.types, type],
    }))
  }, [])

  const resetQuestFilters = useCallback(() => {
    setQuestFilters(emptyQuestFilters())
    setQuestRange('7d')
  }, [])

  return {
    quests,
    questsLoading,
    questFilters,
    setQuestFilters,
    questRange,
    setQuestRange,
    loadQuests,
    toggleFilterStatus,
    toggleFilterType,
    resetQuestFilters,
  }
}

function useInboxList(showError: (msg: string) => void) {
  const [inbox, setInbox] = useState<QuestMeta[]>([])
  const [humanExceptions, setHumanExceptions] = useState<HumanExceptionItem[]>([])
  const [inboxLoading, setInboxLoading] = useState(true)

  const loadInbox = useCallback(async () => {
    try {
      const [res, exceptionRes] = await Promise.all([
        api.get<{ items: QuestMeta[] }>('/api/inbox'),
        api.get<{ items: HumanExceptionItem[] }>('/api/human-exceptions'),
      ])
      setInbox(res.items || [])
      setHumanExceptions(exceptionRes.items || [])
    } catch (e) {
      setInbox([])
      setHumanExceptions([])
      showError(e instanceof Error ? e.message : i18n.t('dashboard.error.loadInbox'))
    } finally {
      setInboxLoading(false)
    }
  }, [showError])

  const removeInboxItem = useCallback((id: string) => {
    setInbox((prev) => prev.filter((item) => item.id !== id))
  }, [])

  return { inbox, humanExceptions, inboxLoading, loadInbox, removeInboxItem }
}

function useInventoryData() {
  const [adventurers, setAdventurers] = useState<AdventurerFile[]>([])
  const [automations, setAutomations] = useState<AutomationConfig[]>([])
  const [executors, setExecutors] = useState<ExecutorInfo[]>([])
  const [skills, setSkills] = useState<SkillManifest[]>([])
  const [prompts, setPrompts] = useState<PromptTemplate[]>([])
  const [selectedAdv, setSelectedAdv] = useState<string | null>(null)
  const [selectedAuto, setSelectedAuto] = useState<string | null>(null)
  const [selectedSkill, setSelectedSkill] = useState<string | null>(null)
  const [selectedPrompt, setSelectedPrompt] = useState<string | null>(null)

  const loadAdventurers = useCallback(async () => {
    try {
      const res = await api.get<{ items: AdventurerFile[] }>('/api/adventurers')
      const list = res.items || []
      setAdventurers(list)
      setSelectedAdv((prev) => (prev && list.find((a) => a.id === prev) ? prev : list[0]?.id ?? null))
    } catch {
      setAdventurers([])
    }
  }, [])

  const loadAutomations = useCallback(async () => {
    try {
      const res = await api.get<{ items: AutomationConfig[] }>('/api/automations')
      const list = res.items || []
      setAutomations(list)
      setSelectedAuto((prev) => (prev && list.find((a) => a.id === prev) ? prev : list[0]?.id ?? null))
    } catch {
      setAutomations([])
    }
  }, [])

  const loadSkills = useCallback(async () => {
    try {
      const res = await api.get<{ items: SkillManifest[] }>('/api/skills')
      const list = res.items || []
      setSkills(list)
      setSelectedSkill((prev) => (prev && list.find((s) => s.name === prev) ? prev : list[0]?.name ?? null))
    } catch {
      setSkills([])
    }
  }, [])

  const loadPrompts = useCallback(async () => {
    try {
      const res = await api.get<{ items: PromptTemplate[] }>('/api/prompts')
      const list = res.items || []
      setPrompts(list)
      setSelectedPrompt((prev) => (prev && list.find((p) => p.name === prev) ? prev : list[0]?.name ?? null))
    } catch {
      setPrompts([])
    }
  }, [])

  const loadExecutors = useCallback(async () => {
    try {
      const res = await api.get<{ items: ExecutorInfo[] }>('/api/executors')
      setExecutors(res.items || [])
    } catch {
      setExecutors([])
    }
  }, [])

  const loadCoreInventory = useCallback(async () => {
    await Promise.all([
      loadAdventurers(),
      loadExecutors(),
    ])
  }, [loadAdventurers, loadExecutors])

  const upsertAutomation = useCallback((item: AutomationConfig) => {
    setAutomations((prev) => {
      const idx = prev.findIndex((a) => a.id === item.id)
      if (idx === -1) return [item, ...prev]
      const next = [...prev]
      next[idx] = item
      return next
    })
    setSelectedAuto((prev) => prev || item.id)
  }, [])

  return {
    adventurers,
    automations,
    executors,
    skills,
    prompts,
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
    loadAdventurers,
    loadAutomations,
    loadSkills,
    loadPrompts,
    loadExecutors,
    loadCoreInventory,
  }
}

function useStatsData() {
  const [stats, setStats] = useState<PersonalStats | null>(null)
  const [statsLoading, setStatsLoading] = useState(false)
  const [statsRange, setStatsRange] = useState<QuestRange>('7d')

  const loadStats = useCallback(async (range: string = statsRange) => {
    setStatsLoading(true)
    try {
      const res = await api.get<{ stats: PersonalStats }>(`/api/stats?range=${range}`)
      setStats(res.stats || null)
    } catch {
      /* stats 是辅助界面 */
    } finally {
      setStatsLoading(false)
    }
  }, [statsRange])

  return {
    stats,
    statsLoading,
    statsRange,
    setStatsRange,
    loadStats,
  }
}

export function useDashboardData({ showError }: UseDashboardDataOptions) {
  const questsData = useQuestList(showError)
  const inboxData = useInboxList(showError)
  const inventoryData = useInventoryData()
  const statsData = useStatsData()
  const { loadQuests } = questsData
  const { loadInbox } = inboxData
  const { loadCoreInventory } = inventoryData

  const loadAll = useCallback(async () => {
    loadQuests()
    loadInbox()
    await loadCoreInventory()
  }, [loadCoreInventory, loadInbox, loadQuests])

  return {
    ...questsData,
    ...inboxData,
    ...inventoryData,
    ...statsData,
    loadAll,
  }
}
