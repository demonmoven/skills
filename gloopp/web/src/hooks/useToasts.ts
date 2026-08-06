import { useCallback, useRef, useState } from 'react'

export type ToastItem = {
  id: number
  kind: 'info' | 'error' | 'success'
  msg: string
}

export function useToasts() {
  const [toasts, setToasts] = useState<ToastItem[]>([])
  const toastIdRef = useRef(0)

  const pushToast = useCallback((kind: ToastItem['kind'], msg: string, ttlMs: number) => {
    const id = ++toastIdRef.current
    setToasts((prev) => [...prev, { id, kind, msg }])
    window.setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id))
    }, ttlMs)
  }, [])

  const showError = useCallback((msg: string) => {
    pushToast('error', msg, 4800)
  }, [pushToast])

  const showInfo = useCallback((msg: string) => {
    pushToast('info', msg, 3200)
  }, [pushToast])

  const showSuccess = useCallback((msg: string) => {
    pushToast('success', msg, 2600)
  }, [pushToast])

  return { toasts, showError, showInfo, showSuccess }
}
