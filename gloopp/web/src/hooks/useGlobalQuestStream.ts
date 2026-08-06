import { useMemo, useRef } from 'react'
import { useEventStream } from '../api/events'
import { debounce } from '../components/util'

type UseGlobalQuestStreamOptions = {
  enabled: boolean
  refreshQuests: () => void
  refreshAutomations: () => void
}

export function useGlobalQuestStream({
  enabled,
  refreshQuests,
  refreshAutomations,
}: UseGlobalQuestStreamOptions) {
  const refreshRef = useRef(refreshQuests)
  refreshRef.current = refreshQuests

  const automationRefreshRef = useRef(refreshAutomations)
  automationRefreshRef.current = refreshAutomations

  const debouncedRefresh = useMemo(() => debounce(() => refreshRef.current(), 250), [])
  const debouncedAutomationRefresh = useMemo(() => debounce(() => automationRefreshRef.current(), 250), [])

  return useEventStream((event) => {
    if (event.type.startsWith('automation.')) {
      debouncedAutomationRefresh()
      return
    }
    if (event.type.startsWith('quest.') || event.type.startsWith('workspace.') || event.type.startsWith('runtime.')) {
      debouncedRefresh()
    }
  }, undefined, enabled)
}
