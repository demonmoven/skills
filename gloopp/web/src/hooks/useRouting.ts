import { useCallback, useEffect, useRef, useState } from 'react'

type Tab = 'quests' | 'inbox' | 'automations' | 'knowledge' | 'adventurers' | 'resources' | 'stats' | 'settings'

const TAB_PATHS: Record<Tab, string> = {
  quests: '/',
  inbox: '/inbox',
  automations: '/automations',
  knowledge: '/knowledge',
  adventurers: '/adventurers',
  resources: '/resources',
  stats: '/stats',
  settings: '/settings',
}

const PATH_TO_TAB: Record<string, Tab> = {}
for (const [tab, path] of Object.entries(TAB_PATHS)) {
  PATH_TO_TAB[path] = tab as Tab
}
// Legacy redirects: /skills and /prompts now live under /resources
PATH_TO_TAB['/skills'] = 'resources'
PATH_TO_TAB['/prompts'] = 'resources'

interface RouteState {
  tab: Tab
  questId: string | null
}

function parseLocation(): RouteState {
  const path = window.location.pathname
  if (path.startsWith('/quests/')) {
    const qid = path.slice('/quests/'.length)
    return { tab: 'quests', questId: qid || null }
  }
  const tab = PATH_TO_TAB[path]
  return { tab: tab || 'quests', questId: null }
}

export function useRouting() {
  const [state, setState] = useState<RouteState>(parseLocation)
  const skipNextPush = useRef(false)

  useEffect(() => {
    const onPop = () => {
      skipNextPush.current = true
      setState(parseLocation())
    }
    window.addEventListener('popstate', onPop)
    return () => window.removeEventListener('popstate', onPop)
  }, [])

  useEffect(() => {
    if (skipNextPush.current) {
      skipNextPush.current = false
      return
    }
    const target = state.questId
      ? `/quests/${state.questId}`
      : TAB_PATHS[state.tab]
    if (window.location.pathname !== target) {
      window.history.pushState(null, '', target)
    }
  }, [state])

  const setTab = useCallback((tab: Tab) => {
    setState({ tab, questId: null })
  }, [])

  const setQuestId = useCallback((qid: string | null) => {
    setState((prev) => ({ ...prev, questId: qid }))
  }, [])

  const openQuest = useCallback((qid: string) => {
    setState({ tab: 'quests', questId: qid })
  }, [])

  return {
    tab: state.tab,
    questId: state.questId,
    setTab,
    setQuestId,
    openQuest,
  }
}
