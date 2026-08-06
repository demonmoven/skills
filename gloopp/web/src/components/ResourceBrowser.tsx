import { useEffect, useMemo, useState, type ReactNode } from 'react'
import Icon from './Icon'

export type ResourceBrowserProps<T, D> = {
  items: T[]
  selected: string | null
  onSelect: (name: string) => void
  searchQuery: string
  filterFn: (item: T, normalizedQuery: string) => boolean
  iconFor: (item: T) => string
  nameOf: (item: T) => string
  subtitleFor: (item: T) => string
  fetchDetail: (name: string) => Promise<D>
  renderDetail: (detail: D, loading: boolean) => ReactNode
  emptyListText: string
  selectOneText: string
  loadingDetailText: string
  pageClass?: string
  displayNameOf?: (item: T) => string
}

export default function ResourceBrowser<T, D>({
  items,
  selected,
  onSelect,
  searchQuery,
  filterFn,
  iconFor,
  nameOf,
  subtitleFor,
  fetchDetail,
  renderDetail,
  emptyListText,
  selectOneText,
  loadingDetailText,
  pageClass = '',
  displayNameOf,
}: ResourceBrowserProps<T, D>) {
  const activeName = useMemo(
    () => selected || (items.length > 0 ? nameOf(items[0]) : null),
    [selected, items, nameOf],
  )
  const [detail, setDetail] = useState<D | null>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!activeName) {
      setDetail(null)
      return
    }
    let cancelled = false
    setLoading(true)
    fetchDetail(activeName)
      .then((d) => {
        if (!cancelled) setDetail(d)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => { cancelled = true }
  }, [activeName, fetchDetail])

  const query = searchQuery.trim().toLowerCase()
  const filtered = useMemo(
    () => (query ? items.filter((item) => filterFn(item, query)) : items),
    [items, query, filterFn],
  )

  const displayName = displayNameOf || nameOf

  return (
    <div className={'configured-page ' + pageClass}>
      <div className="mobile-pill-switcher">
        {filtered.map((item) => (
          <button
            key={nameOf(item)}
            className={'pill-item' + (nameOf(item) === activeName ? ' active' : '')}
            onClick={() => onSelect(nameOf(item))}
          >
            <Icon name={iconFor(item)} size={12} />
            <span>{displayName(item)}</span>
          </button>
        ))}
      </div>
      <aside className="configured-list">
        {filtered.map((item) => (
          <button
            type="button"
            className={
              'configured-list-item ' +
              (nameOf(item) === activeName ? 'active ' : '')
            }
            key={nameOf(item)}
            onClick={() => onSelect(nameOf(item))}
          >
            <Icon name={iconFor(item)} />
            <span>
              <strong>{nameOf(item)}</strong>
              <small>{subtitleFor(item)}</small>
            </span>
          </button>
        ))}
        {filtered.length === 0 && <div className="empty-page">{emptyListText}</div>}
      </aside>

      <main className="configured-detail">
        {detail ? (
          renderDetail(detail, loading)
        ) : (
          <div className="empty-page">
            {loading ? loadingDetailText : selectOneText}
          </div>
        )}
      </main>
    </div>
  )
}
