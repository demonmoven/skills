import { useEffect, useMemo, useState } from 'react'
import { api } from '../api/client'
import type { SkillManifest } from '../api/types'
import Icon from '../components/Icon'
import Logo from '../components/Logo'
import MarkdownRenderer from '../components/MarkdownRenderer'
import { useTranslation } from 'react-i18next'

type Detail = {
  manifest: SkillManifest
  body: string
}

export default function Skills({
  items,
  selected,
  onSelect,
  onError,
  searchQuery,
}: {
  items: SkillManifest[]
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
  const [detail, setDetail] = useState<Detail | null>(null)
  const [mode, setMode] = useState<'render' | 'raw'>('render')
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
        const res = await api.get<{ item: Detail['manifest'] & { body: string; source: string } }>(
          '/api/skills/' + encodeURIComponent(activeName),
        )
        if (cancelled) return
        const { body, source, ...m } = res.item
        setDetail({
          manifest: { ...m },
          body: body || '',
        })
      } catch (e) {
        onError(e instanceof Error ? e.message : t('skills.error.fetchDetail'))
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
    (s) =>
      !searchQuery.trim() ||
      s.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (s.description || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
      (s.category || '').toLowerCase().includes(searchQuery.toLowerCase()),
  )

  function iconFor(item: SkillManifest): string {
    const name = item.name.toLowerCase()
    if (name.includes('review')) return 'wand-sparkles'
    if (name.includes('note')) return 'sticky-note'
    if (name.includes('aware') || name.includes('context')) return 'eye'
    if (name.includes('test')) return 'test-tube'
    if (name.includes('refactor') || name.includes('code')) return 'code'
    return 'swords'
  }

  return (
    <div className="configured-page skills-page">
      <div className="mobile-pill-switcher">
        {filtered.map((item) => (
          <button
            key={item.name}
            className={'pill-item' + (item.name === activeName ? ' active' : '')}
            onClick={() => onSelect(item.name)}
          >
            <Icon name={iconFor(item)} size={12} />
            <span>{item.name}</span>
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
                {t('skills.sourcePrefix')}{item.warrior_available && !item.mage_available ? t('skills.class.warrior') : !item.warrior_available && item.mage_available ? t('skills.class.mage') : item.category || t('skills.class.general')}
              </small>
            </span>
          </button>
        ))}
        {filtered.length === 0 && <div className="empty-page">{t('skills.emptyList')}</div>}
      </aside>

      <main className="configured-detail skills-detail">
        {detail ? (
          <div className="skill-detail-surface">
            <header className="skill-detail-head">
              <div>
                <h2>{detail.manifest.name}</h2>
                <span>{t('skills.builtinPreset')}</span>
              </div>
            </header>

            <section className="settings-section">
              <h3>{t('skills.section.overview')}</h3>
              <p className="review-copy">{detail.manifest.description || t('skills.noDescription')}</p>
            </section>

            <section className="settings-section">
              <h3>{t('skills.section.files')}</h3>
              <div className="skill-file">
                <header>
                  <div>
                    <strong>SKILL.md</strong>
                  </div>
                  <div className="segmented file-mode">
                    <button
                      type="button"
                      className={mode === 'render' ? 'active' : ''}
                      onClick={() => setMode('render')}
                    >
                      {t('skills.mode.render')}
                    </button>
                    <button
                      type="button"
                      className={mode === 'raw' ? 'active' : ''}
                      onClick={() => setMode('raw')}
                    >
                      {t('skills.mode.raw')}
                    </button>
                  </div>
                </header>
                {loading ? (
                  <div className="empty-page small">{t('skills.loadingBody')}</div>
                ) : mode === 'raw' ? (
                  <pre className="skill-markdown mono raw">{detail.body || t('skills.empty')}</pre>
                ) : (
                  <MarkdownRenderer
                    source={detail.body || ''}
                    className="skill-markdown rendered"
                    stripFrontmatter
                  />
                )}
              </div>
            </section>

            <section className="settings-section">
              <h3>{t('skills.section.source')}</h3>
              <div className="origin-card">
                <Logo size={28} variant="official" />
                <div>
                  <strong>{t('skills.builtinPreset')}</strong>
                  <span>{t('skills.origin.builtinDesc')}</span>
                </div>
                <em className="badge">{t('skills.origin.badge')}</em>
              </div>
            </section>

            <div className="automation-chiprow mobile-only">
              {detail.manifest.warrior_available && (
                <span className="chip ok"><Icon name="sword" /> {t('skills.available.warrior')}</span>
              )}
              {detail.manifest.mage_available && (
                <span className="chip violet"><Icon name="wand" /> {t('skills.available.mage')}</span>
              )}
              {detail.manifest.version && (
                <span className="chip">v{detail.manifest.version}</span>
              )}
            </div>
          </div>
        ) : (
          <div className="empty-page">
            {loading ? t('skills.loadingDetail') : t('skills.selectOne')}
          </div>
        )}
      </main>
    </div>
  )
}
