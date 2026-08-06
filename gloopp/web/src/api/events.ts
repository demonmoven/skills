import { useEffect, useRef, useState } from 'react'
import { withToken } from './client'
import type { QuestEvent } from './types'

type Handler = (event: QuestEvent) => void
export type StreamState = 'connecting' | 'open' | 'reconnecting' | 'disconnected'

/** 重连超过该时长后，状态从 reconnecting 升级为 disconnected（仅显示层，底层 EventSource 仍由浏览器自动重连）。 */
const RECONNECT_TIMEOUT_MS = 30000

export function useEventStream(handler: Handler, qid?: string, enabled = true): StreamState {
  const handlerRef = useRef(handler)
  const [state, setState] = useState<StreamState>('connecting')
  const reconnectSinceRef = useRef<number | null>(null)
  handlerRef.current = handler

  useEffect(() => {
    if (!enabled) {
      setState('connecting')
      reconnectSinceRef.current = null
      return
    }
    setState('connecting')
    reconnectSinceRef.current = null
    const path = qid ? withToken('/api/quests/' + qid + '/stream?level=semantic') : withToken('/api/stream')
    const es = new EventSource(path)

    es.onopen = () => {
      reconnectSinceRef.current = null
      setState('open')
    }
    es.onmessage = (ev) => {
      try {
        const event = JSON.parse(ev.data) as QuestEvent
        if (event.type === 'hello') return
        handlerRef.current(event)
      } catch {
        // Ignore malformed stream frames.
      }
    }
    es.onerror = () => {
      if (reconnectSinceRef.current === null) reconnectSinceRef.current = Date.now()
      setState('reconnecting')
    }

    return () => es.close()
  }, [enabled, qid])

  // reconnecting → disconnected 升级：进入 reconnecting 后若持续未恢复，标记断开。
  useEffect(() => {
    if (state !== 'reconnecting') return
    const startedAt = reconnectSinceRef.current ?? Date.now()
    const remaining = Math.max(0, RECONNECT_TIMEOUT_MS - (Date.now() - startedAt))
    const timer = setTimeout(() => {
      setState((prev) => (prev === 'reconnecting' ? 'disconnected' : prev))
    }, remaining)
    return () => clearTimeout(timer)
  }, [state])

  return state
}

export function useMicroStream(handler: Handler, qid?: string, enabled = true): StreamState {
  const handlerRef = useRef(handler)
  const [state, setState] = useState<StreamState>('connecting')
  handlerRef.current = handler

  useEffect(() => {
    if (!enabled || !qid) {
      setState('connecting')
      return
    }
    setState('connecting')
    const path = withToken('/api/quests/' + qid + '/stream')
    const es = new EventSource(path)

    es.onopen = () => setState('open')
    es.onmessage = (ev) => {
      try {
        const event = JSON.parse(ev.data) as QuestEvent
        if (event.type === 'hello') return
        if (!event.type.startsWith('micro.')) return
        handlerRef.current(event)
      } catch {
        // Ignore malformed stream frames.
      }
    }
    es.onerror = () => setState('reconnecting')

    return () => es.close()
  }, [enabled, qid])

  return state
}
