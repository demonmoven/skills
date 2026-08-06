import { useMemo, useRef } from 'react'
import { useEventStream } from '../api/events'
import { debounce } from '../components/util'

type UseContextRefreshStreamOptions = {
  enabled: boolean
  refreshContext: () => void
}

export function useContextRefreshStream({
  enabled,
  refreshContext,
}: UseContextRefreshStreamOptions) {
  const refreshRef = useRef(refreshContext)
  refreshRef.current = refreshContext

  const debouncedRefresh = useMemo(() => debounce(() => refreshRef.current(), 250), [])

  return useEventStream((event) => {
    if (event.type === 'context.updated') {
      debouncedRefresh()
    }
  }, undefined, enabled)
}
