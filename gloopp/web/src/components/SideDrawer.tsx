import { useCallback, useEffect, useRef, useState } from 'react'
import Icon from './Icon'
import Logo from './Logo'
import { useTranslation } from 'react-i18next'

type Tab = 'quests' | 'inbox' | 'automations' | 'knowledge' | 'adventurers' | 'resources' | 'stats' | 'settings'

type NavItem = { key: Tab; label: string; icon: string; count?: number }

type Props = {
  open: boolean
  onClose: () => void
  onOpen: () => void
  currentTab: Tab
  items: NavItem[]
  onNavigate: (tab: Tab) => void
  inboxCount?: number
}

const DRAWER_WIDTH = 280
const EDGE_SWIPE_ZONE = 24 // px from left edge to start swipe-to-open
const DISMISS_THRESHOLD = 0.3 // dismiss when dragged past 30% of width
const VELOCITY_THRESHOLD = 0.3 // px/ms
const TOUCH_SLOP = 8 // minimum px movement to count as a drag

export default function SideDrawer({ open, onClose, onOpen, currentTab, items, onNavigate }: Props) {
  const { t } = useTranslation()
  const drawerRef = useRef<HTMLDivElement>(null)
  const scrimRef = useRef<HTMLDivElement>(null)
  const [translateX, setTranslateX] = useState(0)
  const [isDragging, setIsDragging] = useState(false)
  const dragState = useRef({
    startX: 0,
    startY: 0,
    currentX: 0,
    startTime: 0,
    lastX: 0,
    lastTime: 0,
    tracking: false,
    axisLocked: false,
    axis: null as 'x' | 'y' | null,
    startedOpen: false, // snapshot of `open` at drag start
  })

  // Reset position when open state changes programmatically
  useEffect(() => {
    if (!isDragging) {
      setTranslateX(0)
    }
  }, [open, isDragging])

  const onTouchStart = useCallback(
    (e: TouchEvent) => {
      const state = dragState.current
      const touch = e.touches[0]
      const drawer = drawerRef.current

      // Two ways to start a drag:
      // 1. Drawer is open and touch starts inside the drawer → drag to dismiss
      // 2. Drawer is closed and touch starts near left edge → swipe to open
      const fromInside = open && drawer?.contains(e.target as Node)
      const fromEdge = !open && touch.clientX < EDGE_SWIPE_ZONE

      if (!fromInside && !fromEdge) return

      state.startX = touch.clientX
      state.startY = touch.clientY
      state.currentX = open ? 0 : -DRAWER_WIDTH
      state.startTime = performance.now()
      state.lastX = touch.clientX
      state.lastTime = state.startTime
      state.tracking = true
      state.axisLocked = false
      state.axis = null
      state.startedOpen = open

      // Don't call onOpen / setIsDragging here — wait for a real drag
      // This prevents a simple tap on the left edge from opening the drawer
    },
    [open],
  )

  const onTouchMove = useCallback(
    (e: TouchEvent) => {
      const state = dragState.current
      if (!state.tracking) return

      const touch = e.touches[0]
      const dx = touch.clientX - state.startX
      const dy = touch.clientY - state.startY

      // Determine scroll axis on first significant movement
      if (!state.axisLocked && Math.abs(dx) > TOUCH_SLOP) {
        if (Math.abs(dx) > Math.abs(dy)) {
          state.axis = 'x'
          state.axisLocked = true
          // Start the drag
          setIsDragging(true)
          if (!state.startedOpen) {
            onOpen()
          }
        } else if (Math.abs(dy) > Math.abs(dx)) {
          state.axis = 'y'
          state.axisLocked = true
          state.tracking = false
          return
        }
      }

      if (state.axis !== 'x') return

      // Calculate translation based on drag start state
      let nextX: number
      if (state.startedOpen) {
        // Dragging closed: start at 0, go negative
        nextX = Math.min(0, dx)
      } else {
        // Dragging open: start from -DRAWER_WIDTH, go toward 0
        nextX = -DRAWER_WIDTH + Math.max(0, dx)
      }

      state.currentX = nextX
      state.lastX = touch.clientX
      state.lastTime = performance.now()
      setTranslateX(nextX)

      e.preventDefault()
    },
    [onOpen],
  )

  const onTouchEnd = useCallback(() => {
    const state = dragState.current
    if (!state.tracking) return
    state.tracking = false

    // If axis was never locked (tap / movement was too small), treat as no-op
    if (!state.axisLocked || state.axis !== 'x') {
      return
    }

    setIsDragging(false)

    const dx = state.lastX - state.startX
    const dt = Math.max(1, state.lastTime - state.startTime)
    const velocity = dx / dt

    if (state.startedOpen) {
      // Dismiss if dragged left past threshold OR flung left fast enough
      if (dx < -DRAWER_WIDTH * DISMISS_THRESHOLD || velocity < -VELOCITY_THRESHOLD) {
        onClose()
      } else {
        setTranslateX(0)
      }
    } else {
      // Open if dragged right past threshold OR flung right fast enough
      const openedAmount = dx + EDGE_SWIPE_ZONE
      if (openedAmount > DRAWER_WIDTH * DISMISS_THRESHOLD || velocity > VELOCITY_THRESHOLD) {
        setTranslateX(0)
      } else {
        onClose()
      }
    }
  }, [onClose])

  // Attach global touch listeners for edge-swipe-to-open
  useEffect(() => {
    const handleStart = onTouchStart
    const handleMove = onTouchMove
    const handleEnd = onTouchEnd
    document.addEventListener('touchstart', handleStart, { passive: true })
    document.addEventListener('touchmove', handleMove, { passive: false })
    document.addEventListener('touchend', handleEnd)
    document.addEventListener('touchcancel', handleEnd)
    return () => {
      document.removeEventListener('touchstart', handleStart)
      document.removeEventListener('touchmove', handleMove)
      document.removeEventListener('touchend', handleEnd)
      document.removeEventListener('touchcancel', handleEnd)
    }
  }, [onTouchStart, onTouchMove, onTouchEnd])

  // Handle tab click — close drawer after navigation
  const handleNav = useCallback(
    (tab: Tab) => {
      onNavigate(tab)
      onClose()
    },
    [onNavigate, onClose],
  )

  // Compute scrim opacity based on drawer position
  const progress = open ? (DRAWER_WIDTH + translateX) / DRAWER_WIDTH : 0
  const scrimOpacity = isDragging ? Math.max(0, Math.min(1, progress)) : open ? 1 : 0
  const drawerTranslate = isDragging ? translateX : open ? 0 : -DRAWER_WIDTH

  // Group nav items: primary (quests, inbox, automations) and secondary
  const primaryKeys: Tab[] = ['quests', 'inbox', 'automations']
  const secondaryKeys: Tab[] = ['knowledge', 'adventurers', 'resources', 'stats']
  const settingsKey: Tab = 'settings'

  const primaryItems = items.filter((i) => primaryKeys.includes(i.key))
  const secondaryItems = items.filter((i) => secondaryKeys.includes(i.key))
  const settingsItem = items.find((i) => i.key === settingsKey)

  return (
    <>
      {/* Scrim / backdrop */}
      <div
        ref={scrimRef}
        className="drawer-scrim"
        style={{
          opacity: scrimOpacity,
          pointerEvents: open || isDragging ? 'auto' : 'none',
          transition: isDragging ? 'none' : 'opacity 0.25s ease',
        }}
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Drawer panel */}
      <aside
        ref={drawerRef}
        className="side-drawer"
        style={{
          transform: `translateX(${drawerTranslate}px)`,
          transition: isDragging ? 'none' : 'transform 0.3s cubic-bezier(0.2, 0.8, 0.2, 1)',
          visibility: open || isDragging ? 'visible' : 'hidden',
          willChange: isDragging ? 'transform' : undefined,
        }}
        role="dialog"
        aria-modal="true"
        aria-label={t('sideDrawer.ariaNav')}
      >
        <div className="drawer-header">
          <div className="drawer-brand">
            <Logo variant="official" size={32} />
            <div>
              <strong>Gloop</strong>
              <span>{t('sideDrawer.tagline')}</span>
            </div>
          </div>
          <button className="icon-button drawer-close" onClick={onClose} aria-label={t('sideDrawer.closeAria')}>
            <Icon name="x" />
          </button>
        </div>

        <nav className="drawer-nav">
          <div className="drawer-nav-section">
            <div className="drawer-nav-label">{t('nav.workbench')}</div>
            {primaryItems.map((item) => (
              <button
                key={item.key}
                className={'drawer-nav-item' + (currentTab === item.key ? ' active' : '')}
                onClick={() => handleNav(item.key)}
              >
                <span className="drawer-nav-icon">
                  <Icon name={item.icon} />
                </span>
                <span>{item.label}</span>
                {item.count !== undefined && item.count > 0 && <em>{item.count}</em>}
              </button>
            ))}
          </div>

          <div className="drawer-nav-section">
            <div className="drawer-nav-label">{t('sideDrawer.config')}</div>
            {secondaryItems.map((item) => (
              <button
                key={item.key}
                className={'drawer-nav-item' + (currentTab === item.key ? ' active' : '')}
                onClick={() => handleNav(item.key)}
              >
                <span className="drawer-nav-icon">
                  <Icon name={item.icon} />
                </span>
                <span>{item.label}</span>
                {item.count !== undefined && item.count > 0 && <em>{item.count}</em>}
              </button>
            ))}
          </div>

          <div className="drawer-nav-spacer" />

          {settingsItem && (
            <div className="drawer-nav-section">
              <button
                className={'drawer-nav-item' + (currentTab === settingsItem.key ? ' active' : '')}
                onClick={() => handleNav(settingsItem.key)}
              >
                <span className="drawer-nav-icon">
                  <Icon name={settingsItem.icon} />
                </span>
                <span>{settingsItem.label}</span>
              </button>
              <a
                className="drawer-nav-item external"
                href="https://bytedance.larkoffice.com/docx/KIJ7dcrjZoQ0WqxqKoBcSb9wnVJ"
                target="_blank"
                rel="noopener noreferrer"
              >
                <span className="drawer-nav-icon">
                  <Icon name="file-text" />
                </span>
                <span>{t('nav.docs')}</span>
                <Icon name="open" />
              </a>
            </div>
          )}
        </nav>
      </aside>
    </>
  )
}
