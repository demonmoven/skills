import { useEffect, useMemo, useState } from 'react'
import { api } from '../api/client'
import type { PromptTemplate } from '../api/types'
import Icon from '../components/Icon'
import Logo from '../components/Logo'
import MarkdownRenderer from '../components/MarkdownRenderer'
import { useTranslation } from 'react-i18next'

export default function Prompts({
  items,
  selected,
  onSelect,
  onError,
  searchQuery,
}: {
  items: PromptTemplate[]
  selected: string | null
  onSelect: (name: string) => void
  onError: (msg: string) => void
  searchQuery: string
}) {
  const { t } = useTranslation()
  const activeName = useMemo(
    () => selected || items[0]?.name || null,
    [selected, items],
  )
  const [detail, setDetail] = useState<PromptTemplate | null>(null)
  const [mode, setMode] = useState<'render' | 'raw'>('raw')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!activeName) {
      setDetail(null)
      return
    }
    let cancelled = false
    async function fetchDetail() {
      if (!activeName) return
      setLoading(true)
      try {
        const res = await api.get<{ item: PromptTemplate }>(
          '/api/prompts/' + encodeURIComponent(activeName),
        )
        if (cancelled) return
        setDetail(res.item)
      } catch (e) {
        onError(e instanceof Error ? e.message : t('prompts.error.fetchDetail'))
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    fetchDetail()
    return () => {
      cancelled = true
    }
  }, [activeName, onError])

  const filtered = items.filter(
    (p) =>
      !searchQuery.trim() ||
      p.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (p.preview || '').toLowerCase().includes(searchQuery.toLowerCase()),
  )

  function iconFor(item: PromptTemplate): string {
    const name = item.name.toLowerCase()
    if (name.includes('role')) return 'user-cog'
    if (name.includes('review')) return 'wand-sparkles'
    if (name.includes('block') || name.includes('intensity')) return 'puzzle'
    if (name.includes('system')) return 'settings'
    if (name.includes('user')) return 'message-circle'
    return 'file-text'
  }

  function sourceLabel(source: string): string {
    if (source === 'builtin') return t('prompts.source.builtin')
    if (source === 'user') return t('prompts.source.user')
    return source
  }

  return (
    <div className="configured-page prompts-page">
      <div className="mobile-pill-switcher">
        {filtered.map((item) => (
          <button
            key={item.name}
            className={'pill-item' + (item.name === activeName ? ' active' : '')}
            onClick={() => onSelect(item.name)}
          >
            <Icon name={iconFor(item)} size={12} />
            <span>{item.name.split('/').pop() || item.name}</span>
          </button>
        ))}
      </div>
      <aside className="configured-list">
        {filtered.map((item) => (
          <button
            type="button"
            className={
              'configured-list-item ' +
              (item.name === activeName ? 'active ' : '')
            }
            key={item.name}
            onClick={() => onSelect(item.name)}
          >
            <Icon name={iconFor(item)} />
            <span>
              <strong>{item.name}</strong>
              <small>
                {sourceLabel(item.source)}
                {item.is_overridden && t('prompts.overridden')}
                {' · '}
                {item.size_bytes}B
              </small>
            </span>
          </button>
        ))}
        {filtered.length === 0 && <div className="empty-page">{t('prompts.emptyList')}</div>}
      </aside>

      <main className="configured-detail prompts-detail">
        {detail ? (
          <div className="skill-detail-surface">
            <header className="skill-detail-head">
              <div>
                <h2>{detail.name}</h2>
                <span>
                  {sourceLabel(detail.source)}
                  {detail.is_overridden && t('prompts.overriddenDetail')}
                </span>
              </div>
            </header>

            <section className="settings-section">
              <h3>{t('prompts.section.overview')}</h3>
              <p className="review-copy">{detail.preview || t('prompts.emptyTemplate')}</p>
            </section>

            <section className="settings-section">
              <h3>{t('prompts.section.content')}</h3>
              <div className="skill-file">
                <header>
                  <div>
                    <strong>{detail.name.split('/').pop() || detail.name}</strong>
                    <small>{detail.size_bytes} {t('prompts.bytes')}</small>
                  </div>
                  <div className="segmented file-mode">
                    <button
                      type="button"
                      className={mode === 'render' ? 'active' : ''}
                      onClick={() => setMode('render')}
                    >
                      {t('prompts.mode.render')}
                    </button>
                    <button
                      type="button"
                      className={mode === 'raw' ? 'active' : ''}
                      onClick={() => setMode('raw')}
                    >
                      {t('prompts.mode.raw')}
                    </button>
                  </div>
                </header>
                {loading ? (
                  <div className="empty-page small">{t('prompts.loadingContent')}</div>
                ) : mode === 'raw' ? (
                  <pre className="skill-markdown mono raw">{detail.content || t('prompts.emptyContent')}</pre>
                ) : (
                  <MarkdownRenderer
                    source={detail.content || ''}
                    className="skill-markdown rendered"
                  />
                )}
              </div>
            </section>

            <section className="settings-section">
              <h3>{t('prompts.section.source')}</h3>
              <div className="origin-card">
                <Logo size={28} variant="official" />
                <div>
                  <strong>{detail.source === 'builtin' ? t('prompts.origin.builtinTitle') : t('prompts.origin.userTitle')}</strong>
                  <span>
                    {detail.source === 'builtin'
                      ? t('prompts.origin.builtinDesc')
                      : t('prompts.origin.userDesc')}
                  </span>
                </div>
                <em className="badge">
                  {detail.source === 'builtin' ? t('prompts.origin.badgeBuiltin') : t('prompts.origin.badgeCustom')}
                </em>
              </div>
            </section>

            <div className="automation-chiprow mobile-only">
              <span className="chip">
                <Icon name="hard-drive" /> {detail.size_bytes} B
              </span>
              {detail.is_overridden && (
                <span className="chip ok">
                  <Icon name="check-circle" /> {t('prompts.chip.overridden')}
                </span>
              )}
            </div>
          </div>
        ) : (
          <div className="empty-page">
            {loading ? t('prompts.loadingDetail') : t('prompts.selectOne')}
          </div>
        )}
      </main>
    </div>
  )
}
