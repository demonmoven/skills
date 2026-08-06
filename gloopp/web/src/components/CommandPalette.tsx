import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { CommandItem, CommandSection } from '../features/commandPalette/types'
import Icon from './Icon'

export type { CommandItem, CommandSection } from '../features/commandPalette/types'

type Props = {
  open: boolean
  onClose: () => void
  items: CommandItem[]
}

function fuzzyScore(query: string, text: string): number {
  if (!query) return 1
  const q = query.toLowerCase()
  const t = text.toLowerCase()
  if (t.includes(q)) return 100 - t.indexOf(q)
  let qi = 0
  let score = 0
  let lastMatch = -1
  for (let i = 0; i < t.length && qi < q.length; i++) {
    if (t[i] === q[qi]) {
      score += 10
      if (lastMatch >= 0 && i - lastMatch === 1) score += 5
      lastMatch = i
      qi++
    }
  }
  return qi === q.length ? score : 0
}

function itemTextScore(query: string, item: CommandItem): number {
  if (!query) return 0
  let best = fuzzyScore(query, item.title)
  if (item.subtitle) best = Math.max(best, fuzzyScore(query, item.subtitle))
  if (item.keywords) {
    for (const kw of item.keywords) {
      best = Math.max(best, fuzzyScore(query, kw))
    }
  }
  return best
}

export function orderCommandItemsForPalette(items: CommandItem[], query: string): CommandItem[] {
  const q = query.trim()
  if (!q) {
    return [...items].sort((a, b) => a.priority - b.priority)
  }
  return items
    .map((item) => ({ item, score: itemTextScore(q, item) }))
    .filter(({ score }) => score > 0)
    .sort((a, b) => {
      if (a.item.priority !== b.item.priority) return a.item.priority - b.item.priority
      return b.score - a.score
    })
    .map(({ item }) => item)
}

export default function CommandPalette({ open, onClose, items }: Props) {
  const { t } = useTranslation()
  const [query, setQuery] = useState('')
  const [activeIndex, setActiveIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const previousFocusRef = useRef<HTMLElement | null>(null)

  useEffect(() => {
    if (open) {
      previousFocusRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null
      setQuery('')
      setActiveIndex(0)
      setTimeout(() => inputRef.current?.focus(), 0)
    } else {
      previousFocusRef.current?.focus()
      previousFocusRef.current = null
    }
  }, [open])

  const sorted = useMemo(() => {
    return orderCommandItemsForPalette(items, query)
  }, [items, query])

  const grouped = useMemo(() => {
    const rows: Array<{ section?: CommandSection; items: CommandItem[] }> = []
    for (const item of sorted) {
      const last = rows[rows.length - 1]
      if (last && last.section === item.section) {
        last.items.push(item)
      } else {
        rows.push({ section: item.section, items: [item] })
      }
    }
    return rows
  }, [sorted])

  const flatList = sorted

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      } else if (e.key === 'ArrowDown') {
        e.preventDefault()
        setActiveIndex((i) => Math.min(i + 1, flatList.length - 1))
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setActiveIndex((i) => Math.max(i - 1, 0))
      } else if (e.key === 'Enter') {
        e.preventDefault()
        const cmd = flatList[activeIndex]
        if (cmd && !cmd.disabledReason) {
          cmd.run()
          onClose()
        }
      }
    },
    [flatList, activeIndex, onClose],
  )

  useEffect(() => {
    if (activeIndex >= flatList.length) setActiveIndex(Math.max(0, flatList.length - 1))
  }, [flatList.length, activeIndex])

  useEffect(() => {
    if (!listRef.current) return
    const el = listRef.current.querySelector('[data-active="true"]') as HTMLElement | null
    el?.scrollIntoView({ block: 'nearest' })
  }, [activeIndex])

  if (!open) return null

  const sectionI18n: Record<CommandSection, string> = {
    Decide: t('palette.section.decide'),
    Open: t('palette.section.open'),
    Create: t('palette.section.create'),
    Navigate: t('palette.section.navigate'),
    Configure: t('palette.section.configure'),
    Inspect: t('palette.section.inspect'),
  }

  const renderItem = (cmd: CommandItem, idx: number) => {
    const isActive = idx === activeIndex
    return (
      <button
        key={cmd.id}
        type="button"
        className={'palette-item' + (isActive ? ' active' : '') + (cmd.disabledReason ? ' disabled' : '')}
        data-active={isActive}
        title={cmd.disabledReason}
        onMouseEnter={() => setActiveIndex(idx)}
        onClick={() => {
          if (!cmd.disabledReason) {
            cmd.run()
            onClose()
          }
        }}
      >
        <span className="palette-item-icon">
          <Icon name={cmd.icon || 'square'} />
        </span>
        <span className="palette-item-text">
          <strong>{cmd.title}</strong>
          {cmd.subtitle && <em>{cmd.subtitle}</em>}
        </span>
        {cmd.disabledReason && (
          <span className="palette-item-disabled">{cmd.disabledReason}</span>
        )}
      </button>
    )
  }

  let globalIdx = 0

  return (
    <div className="palette-scrim" onClick={onClose} role="dialog" aria-modal="true" aria-label={t('palette.aria')}>
      <div className="palette-card" onClick={(e) => e.stopPropagation()}>
        <div className="palette-input-wrap">
          <Icon name="search" />
          <input
            ref={inputRef}
            type="text"
            className="palette-input"
            placeholder={t('palette.placeholder')}
            value={query}
            onChange={(e) => {
              setQuery(e.target.value)
              setActiveIndex(0)
            }}
            onKeyDown={handleKeyDown}
            autoComplete="off"
            spellCheck={false}
          />
          {query && (
            <button className="palette-clear" onClick={() => setQuery('')} aria-label={t('aria.clear')}>
              <Icon name="x" size={14} />
            </button>
          )}
        </div>
        <div className="palette-list" ref={listRef}>
          {flatList.length === 0 ? (
            <div className="palette-empty">{t('palette.noResults')}</div>
          ) : (
            <>
              {grouped.map((group, groupIdx) => (
                <div key={(group.section || 'ungrouped') + '-' + groupIdx} className="palette-group">
                  {group.section && <div className="palette-group-label">{sectionI18n[group.section]}</div>}
                  {group.items.map((cmd) => renderItem(cmd, globalIdx++))}
                </div>
              ))}
            </>
          )}
        </div>
        <div className="palette-foot">
          <span><kbd>↑</kbd><kbd>↓</kbd> {t('palette.hint.navigate')}</span>
          <span><kbd>↵</kbd> {t('palette.hint.select')}</span>
          <span><kbd>esc</kbd> {t('palette.hint.close')}</span>
        </div>
      </div>
    </div>
  )
}
