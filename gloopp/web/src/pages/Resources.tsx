import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import type { SkillManifest, PromptTemplate } from '../api/types'
import Icon from '../components/Icon'
import Logo from '../components/Logo'
import MarkdownRenderer from '../components/MarkdownRenderer'
import ResourceBrowser from '../components/ResourceBrowser'
import { useTranslation } from 'react-i18next'

type ResourceTab = 'skills' | 'prompts'

type Props = {
  skills: SkillManifest[]
  prompts: PromptTemplate[]
  selectedSkill: string | null
  selectedPrompt: string | null
  onSelectSkill: (name: string) => void
  onSelectPrompt: (name: string) => void
  onError: (msg: string) => void
  searchQuery: string
  activeResourceTab: ResourceTab
  onResourceTabChange: (tab: ResourceTab) => void
}

function skillIcon(item: SkillManifest): string {
  const name = item.name.toLowerCase()
  if (name.includes('review')) return 'wand-sparkles'
  if (name.includes('note')) return 'sticky-note'
  if (name.includes('aware') || name.includes('context')) return 'eye'
  if (name.includes('test')) return 'test-tube'
  if (name.includes('refactor') || name.includes('code')) return 'code'
  return 'swords'
}

function promptIcon(item: PromptTemplate): string {
  const name = item.name.toLowerCase()
  if (name.includes('role')) return 'user-cog'
  if (name.includes('review')) return 'wand-sparkles'
  if (name.includes('block') || name.includes('intensity')) return 'puzzle'
  if (name.includes('system')) return 'settings'
  if (name.includes('user')) return 'message-circle'
  return 'file-text'
}

export default function Resources({
  skills,
  prompts,
  selectedSkill,
  selectedPrompt,
  onSelectSkill,
  onSelectPrompt,
  onError,
  searchQuery,
  activeResourceTab,
  onResourceTabChange,
}: Props) {
  const { t } = useTranslation()
  const [tab, setTab] = useState<ResourceTab>(activeResourceTab)
  const [skillMode, setSkillMode] = useState<'render' | 'raw'>('render')
  const [promptMode, setPromptMode] = useState<'render' | 'raw'>('raw')

  useEffect(() => {
    setTab(activeResourceTab)
  }, [activeResourceTab])

  function selectTab(next: ResourceTab) {
    setTab(next)
    onResourceTabChange(next)
  }

  const fetchSkillDetail = useCallback(async (name: string) => {
    const res = await api.get<{ item: SkillManifest & { body: string; source: string } }>(
      '/api/skills/' + encodeURIComponent(name),
    )
    return res.item
  }, [])

  const fetchPromptDetail = useCallback(async (name: string) => {
    const res = await api.get<{ item: PromptTemplate }>(
      '/api/prompts/' + encodeURIComponent(name),
    )
    return res.item
  }, [])

  function renderSkillDetail(detail: SkillManifest & { body?: string; source?: string }, loading: boolean) {
    const body = (detail as any).body || ''
    return (
      <div className="skill-detail-surface">
        <header className="skill-detail-head">
          <div>
            <h2>{detail.name}</h2>
            <span>{t('skills.builtinPreset')}</span>
          </div>
        </header>
        <section className="settings-section">
          <h3>{t('skills.section.overview')}</h3>
          <p className="review-copy">{detail.description || t('skills.noDescription')}</p>
        </section>
        <section className="settings-section">
          <h3>{t('skills.section.files')}</h3>
          <div className="skill-file">
            <header>
              <div><strong>SKILL.md</strong></div>
              <div className="segmented file-mode">
                <button type="button" className={skillMode === 'render' ? 'active' : ''} onClick={() => setSkillMode('render')}>{t('skills.mode.render')}</button>
                <button type="button" className={skillMode === 'raw' ? 'active' : ''} onClick={() => setSkillMode('raw')}>{t('skills.mode.raw')}</button>
              </div>
            </header>
            {loading ? (
              <div className="empty-page small">{t('skills.loadingBody')}</div>
            ) : skillMode === 'raw' ? (
              <pre className="skill-markdown mono raw">{body || t('skills.empty')}</pre>
            ) : (
              <MarkdownRenderer source={body} className="skill-markdown rendered" stripFrontmatter />
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
          {detail.warrior_available && <span className="chip ok"><Icon name="sword" /> {t('skills.available.warrior')}</span>}
          {detail.mage_available && <span className="chip violet"><Icon name="wand" /> {t('skills.available.mage')}</span>}
          {detail.version && <span className="chip">v{detail.version}</span>}
        </div>
      </div>
    )
  }

  function renderPromptDetail(detail: PromptTemplate, loading: boolean) {
    function sourceLabel(source: string): string {
      if (source === 'builtin') return t('prompts.source.builtin')
      if (source === 'user') return t('prompts.source.user')
      return source
    }
    return (
      <div className="skill-detail-surface">
        <header className="skill-detail-head">
          <div>
            <h2>{detail.name}</h2>
            <span>{sourceLabel(detail.source)}{detail.is_overridden && t('prompts.overriddenDetail')}</span>
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
                <button type="button" className={promptMode === 'render' ? 'active' : ''} onClick={() => setPromptMode('render')}>{t('prompts.mode.render')}</button>
                <button type="button" className={promptMode === 'raw' ? 'active' : ''} onClick={() => setPromptMode('raw')}>{t('prompts.mode.raw')}</button>
              </div>
            </header>
            {loading ? (
              <div className="empty-page small">{t('prompts.loadingContent')}</div>
            ) : promptMode === 'raw' ? (
              <pre className="skill-markdown mono raw">{detail.content || t('prompts.emptyContent')}</pre>
            ) : (
              <MarkdownRenderer source={detail.content || ''} className="skill-markdown rendered" />
            )}
          </div>
        </section>
        <section className="settings-section">
          <h3>{t('prompts.section.source')}</h3>
          <div className="origin-card">
            <Logo size={28} variant="official" />
            <div>
              <strong>{detail.source === 'builtin' ? t('prompts.origin.builtinTitle') : t('prompts.origin.userTitle')}</strong>
              <span>{detail.source === 'builtin' ? t('prompts.origin.builtinDesc') : t('prompts.origin.userDesc')}</span>
            </div>
            <em className="badge">{detail.source === 'builtin' ? t('prompts.origin.badgeBuiltin') : t('prompts.origin.badgeCustom')}</em>
          </div>
        </section>
        <div className="automation-chiprow mobile-only">
          <span className="chip"><Icon name="hard-drive" /> {detail.size_bytes} B</span>
          {detail.is_overridden && <span className="chip ok"><Icon name="check-circle" /> {t('prompts.chip.overridden')}</span>}
        </div>
      </div>
    )
  }

  return (
    <div className="resources-page">
      <div className="resources-tab-bar">
        <button
          type="button"
          className={'resources-tab' + (tab === 'skills' ? ' active' : '')}
          onClick={() => selectTab('skills')}
        >
          <Icon name="skills" size={14} />
          {t('nav.skills')}
          <em className="resources-count">{skills.length}</em>
        </button>
        <button
          type="button"
          className={'resources-tab' + (tab === 'prompts' ? ' active' : '')}
          onClick={() => selectTab('prompts')}
        >
          <Icon name="message-square" size={14} />
          {t('nav.prompts')}
          <em className="resources-count">{prompts.length}</em>
        </button>
      </div>

      {tab === 'skills' ? (
        <ResourceBrowser<SkillManifest, SkillManifest & { body?: string; source?: string }>
          items={skills}
          selected={selectedSkill}
          onSelect={onSelectSkill}
          searchQuery={searchQuery}
          filterFn={(s, q) =>
            s.name.toLowerCase().includes(q) ||
            (s.description || '').toLowerCase().includes(q) ||
            (s.category || '').toLowerCase().includes(q)
          }
          iconFor={skillIcon}
          nameOf={(s) => s.name}
          subtitleFor={(s) => {
            const prefix = t('skills.sourcePrefix')
            const cls = s.warrior_available && !s.mage_available
              ? t('skills.class.warrior')
              : !s.warrior_available && s.mage_available
                ? t('skills.class.mage')
                : s.category || t('skills.class.general')
            return prefix + cls
          }}
          fetchDetail={fetchSkillDetail}
          renderDetail={renderSkillDetail}
          emptyListText={t('skills.emptyList')}
          selectOneText={t('skills.selectOne')}
          loadingDetailText={t('skills.loadingDetail')}
          pageClass="skills-page"
        />
      ) : (
        <ResourceBrowser<PromptTemplate, PromptTemplate>
          items={prompts}
          selected={selectedPrompt}
          onSelect={onSelectPrompt}
          searchQuery={searchQuery}
          filterFn={(p, q) =>
            p.name.toLowerCase().includes(q) ||
            (p.preview || '').toLowerCase().includes(q)
          }
          iconFor={promptIcon}
          nameOf={(p) => p.name}
          displayNameOf={(p) => p.name.split('/').pop() || p.name}
          subtitleFor={(p) => {
            const src = p.source === 'builtin' ? t('prompts.source.builtin') : p.source === 'user' ? t('prompts.source.user') : p.source
            return src + (p.is_overridden ? t('prompts.overridden') : '') + ' · ' + p.size_bytes + 'B'
          }}
          fetchDetail={fetchPromptDetail}
          renderDetail={renderPromptDetail}
          emptyListText={t('prompts.emptyList')}
          selectOneText={t('prompts.selectOne')}
          loadingDetailText={t('prompts.loadingDetail')}
          pageClass="prompts-page"
        />
      )}
    </div>
  )
}
