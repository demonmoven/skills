import { useCallback, useEffect, useRef, useState } from 'react'

type Options = {
  onDismiss: () => void
  threshold?: number // dismiss when dragged beyond this pixels (default: 120)
  velocityThreshold?: number // velocity-based dismiss threshold in px/ms
  disabled?: boolean
}

/**
 * Hook for bottom-sheet drag-to-dismiss behavior on touch devices.
 *
 * 关键设计：
 * 1. 拖拽关闭只从 handle 起步。内容区域一律交给原生滚动，即使滚到顶部
 *    也不拦截 —— 否则用户从顶部向下滑想看内容会被 preventDefault 吞掉。
 * 2. 非被动 touchmove 监听器只在确认从 handle 起步时动态注册，内容区域
 *    的普通触摸完全不经过它，浏览器可原生滚动，零阻碍。
 *
 * Usage:
 *   const { sheetRef, handleRef, style, isDragging } = useDragToDismiss({ onDismiss: onClose })
 *   <div className="sheet-card" ref={sheetRef} style={style}>
 *     <i className="sheet-handle" ref={handleRef} />
 *     ...
 *   </div>
 */
export default function useDragToDismiss({
  onDismiss,
  threshold = 120,
  velocityThreshold = 0.4,
  disabled = false,
}: Options) {
  const sheetRef = useRef<HTMLDivElement | null>(null)
  const handleRef = useRef<HTMLElement | null>(null)
  const stateRef = useRef({
    isDragging: false,
    startY: 0,
    currentY: 0,
    startTime: 0,
    lastY: 0,
    lastTime: 0,
  })
  const [translateY, setTranslateY] = useState(0)
  const [isDragging, setIsDragging] = useState(false)

  const onTouchMove = useCallback(
    (e: TouchEvent) => {
      const state = stateRef.current
      if (!state.isDragging) return

      const touch = e.touches[0]
      const delta = touch.clientY - state.startY

      // Track velocity
      state.lastY = touch.clientY
      state.lastTime = performance.now()

      // Damping when pulling upward (above starting position)
      if (delta < 0) {
        state.currentY = delta * 0.25
      } else {
        // Slight resistance for downward drag
        state.currentY = delta * 0.85
      }

      setTranslateY(state.currentY)

      // Prevent background scroll when dragging
      if (Math.abs(delta) > 5) {
        e.preventDefault()
      }
    },
    [],
  )

  const onTouchStart = useCallback(
    (e: TouchEvent) => {
      if (disabled) return
      const state = stateRef.current
      const target = e.target as Node

      // 拖拽关闭只从 handle 起步。内容区域一律交给原生滚动，
      // 即使滚到顶部也不拦截 —— 否则用户从顶部向下滑想看内容
      // 会被 preventDefault 吞掉，导致"划不动"。
      const fromHandle = handleRef.current?.contains(target)
      if (!fromHandle) return

      const touch = e.touches[0]
      state.isDragging = true
      state.startY = touch.clientY
      state.currentY = 0
      state.startTime = performance.now()
      state.lastY = touch.clientY
      state.lastTime = state.startTime
      setIsDragging(true)
      // 确认要拖拽，才动态注册非被动 touchmove 监听器
      sheetRef.current?.addEventListener('touchmove', onTouchMove, { passive: false })
    },
    [disabled, onTouchMove],
  )

  const onTouchEnd = useCallback(() => {
    const state = stateRef.current
    if (!state.isDragging) return
    state.isDragging = false
    setIsDragging(false)
    // 拖拽结束，移除非被动 touchmove 监听器
    sheetRef.current?.removeEventListener('touchmove', onTouchMove)

    const delta = state.currentY
    const dt = Math.max(1, state.lastTime - state.startTime)
    const velocity = delta / dt

    // Dismiss if dragged past threshold OR flung down with enough velocity
    if (delta > threshold || velocity > velocityThreshold) {
      // Animate off-screen
      setTranslateY(window.innerHeight)
      setTimeout(onDismiss, 220)
    } else {
      // Spring back
      setTranslateY(0)
    }
  }, [threshold, velocityThreshold, onDismiss, onTouchMove])

  const setSheetRef = useCallback(
    (node: HTMLDivElement | null) => {
      const prev = sheetRef.current
      if (prev) {
        prev.removeEventListener('touchstart', onTouchStart)
        // touchmove 不再常驻绑定，这里只是兜底清理
        prev.removeEventListener('touchmove', onTouchMove)
        prev.removeEventListener('touchend', onTouchEnd)
        prev.removeEventListener('touchcancel', onTouchEnd)
      }
      sheetRef.current = node
      if (node) {
        // touchstart 用 passive，不阻碍原生滚动决策
        node.addEventListener('touchstart', onTouchStart, { passive: true })
        // touchmove 不在这里常驻绑定 —— 由 onTouchStart 动态注册
        node.addEventListener('touchend', onTouchEnd)
        node.addEventListener('touchcancel', onTouchEnd)
      }
    },
    [onTouchStart, onTouchMove, onTouchEnd],
  )

  // Reset translate when disabled changes
  useEffect(() => {
    if (disabled) {
      setTranslateY(0)
      setIsDragging(false)
    }
  }, [disabled])

  const sheetStyle: React.CSSProperties = {
    transform: translateY !== 0 ? `translateY(${translateY}px)` : undefined,
    transition: isDragging ? 'none' : 'transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1)',
    willChange: isDragging ? 'transform' : undefined,
  }

  // Scrim fades out as sheet is dragged down
  const scrimOpacity = isDragging ? Math.max(0.2, 1 - Math.max(0, translateY) / 400) : 1
  const scrimStyle: React.CSSProperties = {
    opacity: scrimOpacity,
    transition: isDragging ? 'none' : 'opacity 0.2s ease',
  }

  return { sheetRef: setSheetRef, handleRef, sheetStyle, scrimStyle, isDragging, translateY }
}
