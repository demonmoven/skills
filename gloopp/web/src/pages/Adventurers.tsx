import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api/client'
import type { AdventurerFile, ExecutorInfo, QuestMeta } from '../api/types'
import CapabilityTierBadge from '../components/CapabilityTierBadge'
import DesignSelect from '../components/DesignSelect'
import Icon from '../components/Icon'
import Logo from '../components/Logo'
import { adventurerModelPolicyLabel, agentKey, agentLabel, isSelectableAgentExecutor, relativeTime, shortId } from '../components/util'
import useDragToDismiss from '../hooks/useDragToDismiss'

const defaultSkills: Record<string, string[]> = {
  warrior: ['gloop-quest-execution', 'gloop-note-keeping', 'gloop-self-awareness', 'gloop-user-context'],
  mage: ['gloop-quest-review', 'gloop-note-keeping', 'gloop-self-awareness', 'gloop-user-context'],
}

export default function Adventurers({
  items,
  executors,
  selected,
  onSelect,
  onError,
  onSuccess,
  onChanged,
  searchQuery,
  onOpenQuest,
}: {
  items: AdventurerFile[]
  executors: ExecutorInfo[]
  selected: string | null
  onSelect: (id: string) => void
  onError: (msg: string) => void
  onSuccess: (msg: string) => void
  onChanged: () => void
  searchQuery: string
  onOpenQuest: (qid: string) => void
}) {
  const { t } = useTranslation()
  const filtered = useMemo(() => {
    if (!searchQuery.trim()) return items
    const q = searchQuery.toLowerCase()
    return items.filter(
      (a) =>
        a.name.toLowerCase().includes(q) ||
        (a.description || '').toLowerCase().includes(q) ||
        (a.class || '').toLowerCase().includes(q),
    )
  }, [items, searchQuery])
  const warriors = filtered.filter((item) => item.class === 'warrior')
  const mages = filtered.filter((item) => item.class === 'mage')
  const active = useMemo(
    () => items.find((a) => a.id === selected) || null,
    [items, selected],
  )

  // controlled detail form state (re-sync when selected changes)
  const [form, setForm] = useState<Partial<AdventurerFile>>({})
  const [saving, setSaving] = useState(false)
  const [retiring, setRetiring] = useState(false)
  const [retireConfirm, setRetireConfirm] = useState(false)
  const { sheetRef: retireSheetRef, sheetStyle: retireSheetStyle, scrimStyle: retireScrimStyle } =
    useDragToDismiss({ onDismiss: () => setRetireConfirm(false), disabled: !retireConfirm })

  const agentOptions = useMemo(() => {
    const seen = new Set<string>()
    return executors
      .map((executor) => ({ executor, agent: agentKey(executor) }))
      .filter(({ executor, agent }) => {
        if (!isSelectableAgentExecutor(executor)) return false
        if (!agent || seen.has(agent)) return false
        seen.add(agent)
        return true
      })
      .sort((a, b) => agentLabel(a.agent).localeCompare(agentLabel(b.agent)))
  }, [executors])

  const [recentQuests, setRecentQuests] = useState<QuestMeta[]>([])
  const [defaultWarriorId, setDefaultWarriorId] = useState('')
  const [defaultMageId, setDefaultMageId] = useState('')
  const [settingDefault, setSettingDefault] = useState(false)

  useEffect(() => {
    api.get<{ config?: { default_warrior_id?: string; default_mage_id?: string } }>('/api/settings')
      .then((res) => {
        setDefaultWarriorId(res.config?.default_warrior_id || '')
        setDefaultMageId(res.config?.default_mage_id || '')
      })
      .catch(() => {})
  }, [onChanged])

  useEffect(() => {
    if (!active?.id) {
      setRecentQuests([])
      return
    }
    api.get<{ ok: boolean; items: QuestMeta[] }>(`/api/quests?adventurer_id=${encodeURIComponent(active.id)}`)
      .then((res) => {
        const sorted = (res.items || []).sort((a, b) => (b.updated_at_ms || b.created_at_ms) - (a.updated_at_ms || a.created_at_ms))
        setRecentQuests(sorted.slice(0, 10))
      })
      .catch(() => setRecentQuests([]))
  }, [active?.id])

  useEffect(() => {
    setForm(active ? { ...active } : {})
    setRetireConfirm(false)
  }, [active?.id])

  const mergedForm: Partial<AdventurerFile> = useMemo(() => {
    if (!active) return {}
    return { ...active, ...form }
  }, [active, form])

  const selectedExecutor = useMemo(() => {
    const agent = mergedForm.agent
    if (!agent) return null
    return agentOptions.find(({ agent: a }) => a === agent)?.executor || null
  }, [mergedForm.agent, agentOptions])

  const hasSelectableFormAgent = !!mergedForm.agent && agentOptions.some(({ agent }) => agent === mergedForm.agent)

  const winRate = useMemo(() => {
    const total = (active?.win_count || 0) + (active?.lose_count || 0)
    if (total === 0) return 0
    return Math.round(((active?.win_count || 0) / total) * 1000) / 10
  }, [active])

  if (items.length === 0) {
    return (
      <div className="configured-page adventurer-page">
        <aside className="configured-list" />
        <main className="configured-detail">
          <div className="empty-page">{t('adventurer.empty')}</div>
        </main>
      </div>
    )
  }

  function patch<K extends keyof AdventurerFile>(k: K, v: AdventurerFile[K]) {
    setForm((prev) => ({ ...prev, [k]: v }))
  }

  async function save() {
    if (!active) return
    if (!hasSelectableFormAgent) {
      onError(t('adventurer.error.needAgent'))
      return
    }
    setSaving(true)
    try {
      const toolsRaw: unknown = form.tools
      const toolsArr =
        typeof toolsRaw === 'string'
          ? toolsRaw
            .split(',')
            .map((s: string) => s.trim())
            .filter(Boolean)
          : Array.isArray(toolsRaw)
            ? (toolsRaw as string[])
            : undefined
      // 若 agent 填全且当前 pending，API 会自动激活；模型由 agent 默认模型决定。
      const payload: Record<string, any> = {
        name: form.name,
        class: form.class,
        agent: form.agent,
        model: typeof form.model === 'string' ? form.model.trim() : '',
        description: form.description,
        custom_prompt: form.custom_prompt,
        tools: toolsArr,
        level: form.level,
        status: form.status,
        exp: form.exp,
        win_count: form.win_count,
        lose_count: form.lose_count,
      }
      await api.post(`/api/adventurers/${active.id}`, payload)
      onChanged()
      onSuccess(t('adventurer.success.saved'))
    } catch (e) {
      onError(e instanceof Error ? e.message : t('adventurer.error.saveFail'))
    } finally {
      setSaving(false)
    }
  }

  async function activate() {
    if (!active) return
    if (!hasSelectableFormAgent) {
      onError(t('adventurer.error.needAgent'))
      return
    }
    setSaving(true)
    try {
      await api.post(`/api/adventurers/${active.id}/activate`, {
        name: form.name,
        agent: form.agent,
        model: typeof form.model === 'string' ? form.model.trim() : '',
        description: form.description,
        custom_prompt: form.custom_prompt,
        tools: (() => {
          const raw: unknown = form.tools
          return typeof raw === 'string'
            ? raw
              .split(',')
              .map((s: string) => s.trim())
              .filter(Boolean)
            : Array.isArray(raw)
              ? (raw as string[])
              : undefined
        })(),
      })
      onChanged()
      onSuccess(t('adventurer.success.activated'))
    } catch (e) {
      onError(e instanceof Error ? e.message : t('adventurer.error.activateFail'))
    } finally {
      setSaving(false)
    }
  }

  async function retire() {
    if (!active) return
    setRetiring(true)
    try {
      await api.post(`/api/adventurers/${active.id}/retire`)
      setRetireConfirm(false)
      onChanged()
    } catch (e) {
      onError(e instanceof Error ? e.message : t('adventurer.error.retireFail'))
    } finally {
      setRetiring(false)
    }
  }

  async function toggleDefault() {
    if (!active) return
    const key = active.class === 'warrior' ? 'default_warrior_id' : 'default_mage_id'
    const current = active.class === 'warrior' ? defaultWarriorId : defaultMageId
    const next = current === active.id ? '' : active.id
    setSettingDefault(true)
    try {
      await api.patch('/api/settings', { [key]: next })
      if (active.class === 'warrior') setDefaultWarriorId(next)
      else setDefaultMageId(next)
      const successKey = next
        ? active.class === 'warrior' ? 'adventurer.success.setDefaultWarrior' : 'adventurer.success.setDefaultMage'
        : active.class === 'warrior' ? 'adventurer.success.unsetDefaultWarrior' : 'adventurer.success.unsetDefaultMage'
      onSuccess(t(successKey))
    } catch (e) {
      onError(e instanceof Error ? e.message : t('adventurer.error.setDefaultFail'))
    } finally {
      setSettingDefault(false)
    }
  }

  const isPending = active?.status === 'pending_setup'
  const isRetired = active?.status === 'retired'

  return (
    <div className="configured-page adventurer-page">
      <div className="mobile-pill-switcher">
        {[...warriors, ...mages].map((item) => (
          <button
            key={item.id}
            className={'pill-item' + (item.id === active?.id ? ' active' : '')}
            onClick={() => onSelect(item.id)}
          >
            <Logo variant={item.class === 'mage' ? 'mage' : 'knight'} size={20} />
            <span>{item.name}</span>
          </button>
        ))}
      </div>
      <aside className="configured-list">
        <div className="list-group-label">
          <Icon name="swords" />
          <span>{t('world.warrior')}</span>
        </div>
        {warriors.map((item) => (
          <button
            type="button"
            className={
              'configured-list-item agent ' +
              (item.id === active?.id ? 'active ' : '') +
              (item.status === 'pending_setup'
                ? 'pending '
                : item.status === 'retired'
                  ? 'retired '
                  : '') +
              (item.id === defaultWarriorId ? 'is-default ' : '')
            }
            key={item.id}
            onClick={() => onSelect(item.id)}
          >
            <Logo variant="knight" size={28} />
            <span>
              <strong>
                {item.name}
                {item.status === 'active' && <i className="status-dot ok" />}
                {item.status === 'pending_setup' && (
                  <i className="status-dot amber" />
                )}
                {item.id === defaultWarriorId && (
                  <i className="default-mark" title={t('adventurer.list.defaultWarrior')}><Icon name="star" size={12} /></i>
                )}
              </strong>
              <small>
                {item.status === 'pending_setup'
                  ? t('adventurer.status.pending')
                  : item.status === 'retired'
                    ? t('adventurer.status.retired')
                    : item.agent || t('adventurer.status.noAgent')}
              </small>
            </span>
          </button>
        ))}
        <div className="list-group-label">
          <Icon name="wand-sparkles" />
          <span>{t('world.mage')}</span>
        </div>
        {mages.map((item) => (
          <button
            type="button"
            className={
              'configured-list-item agent ' +
              (item.id === active?.id ? 'active ' : '') +
              (item.status === 'pending_setup'
                ? 'pending '
                : item.status === 'retired'
                  ? 'retired '
                  : '') +
              (item.id === defaultMageId ? 'is-default ' : '')
            }
            key={item.id}
            onClick={() => onSelect(item.id)}
          >
            <Logo variant="mage" size={28} />
            <span>
              <strong>
                {item.name}
                {item.status === 'active' && <i className="status-dot ok" />}
                {item.status === 'pending_setup' && (
                  <i className="status-dot amber" />
                )}
                {item.id === defaultMageId && (
                  <i className="default-mark" title={t('adventurer.list.defaultMage')}><Icon name="star" size={12} /></i>
                )}
              </strong>
              <small>
                {item.status === 'pending_setup'
                  ? t('adventurer.status.pending')
                  : item.status === 'retired'
                    ? t('adventurer.status.retired')
                    : item.agent || t('adventurer.status.noAgent')}
              </small>
            </span>
          </button>
        ))}
        {filtered.length === 0 && searchQuery && (
          <div className="empty-page">{t('adventurer.noMatch')}</div>
        )}
      </aside>

      <main className="configured-detail adventurer-detail three-pane">
        {active ? (
          <>
            <div className="form-surface">
              <div className="detail-title">
                <span className="agent-avatar large">
                  <Logo variant={active.class === 'mage' ? 'mage' : 'knight'} size={48} />
                </span>
                <div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <h2>{mergedForm.name || active.name}</h2>
                    {active.status === 'active' && (
                      <span className="chip ok">{t('adventurer.status.active')}</span>
                    )}
                    {active.status === 'pending_setup' && (
                      <span className="chip amber">{t('adventurer.status.pending')}</span>
                    )}
                    {active.status === 'retired' && (
                      <span className="chip">{t('adventurer.status.retired')}</span>
                    )}
                  </div>
                  <span className="subtle-line">
                    {active.class === 'mage' ? t('world.mage') : t('world.warrior')} / <span className="mono">Lv.{active.level}</span> {active.title || ''}
                  </span>
                </div>
              </div>

              {isPending && (
                <div className="bottom-banner amber">
                  <Icon name="alert-triangle" /> {t('adventurer.banner.notActivated')}
                </div>
              )}

              <label className="field">
                <span>{t('adventurer.field.systemPrompt')}</span>
                <textarea
                  rows={3}
                  className="mono"
                  value={mergedForm.custom_prompt || t('adventurer.field.systemPromptDefault')}
                  onChange={(e) => patch('custom_prompt', e.target.value)}
                  placeholder={t('adventurer.field.systemPromptPlaceholder')}
                />
              </label>

              <div className="row-actions left sticky-actions">
                {isPending ? (
                  <button className="button primary" onClick={activate} disabled={saving}>
                    {saving ? t('adventurer.action.activating') : t('adventurer.action.activate')}
                  </button>
                ) : (
                  <button className="button primary" onClick={save} disabled={saving || isRetired}>
                    {saving ? t('adventurer.action.saving') : t('adventurer.action.save')}
                  </button>
                )}
                {!isPending && !isRetired && (() => {
                  const isDefault = active.class === 'warrior'
                    ? defaultWarriorId === active.id
                    : defaultMageId === active.id
                  return (
                    <button
                      className={'button ghost' + (isDefault ? ' default-active' : '')}
                      onClick={toggleDefault}
                      disabled={settingDefault}
                    >
                      <Icon name="star" size={14} />
                      {settingDefault ? t('adventurer.action.settingDefault') : isDefault ? t('adventurer.action.isDefault') : t('adventurer.action.setDefault')}
                    </button>
                  )
                })()}
                <button
                  className="button danger ghost"
                  onClick={() => setRetireConfirm(true)}
                  disabled={retiring || isRetired}
                >
                  {retiring ? t('adventurer.action.retiring') : t('adventurer.action.retire')}
                </button>
              </div>

              <div className="recent-runs-section">
                <h4>{t('adventurer.recent.title')}</h4>
                {recentQuests.length === 0 ? (
                  <p className="muted">{t('adventurer.recent.empty')}</p>
                ) : (
                  <div className="activity-summary">
                    <div className="activity-status-line">
                      {(() => {
                        const running = recentQuests.find((q) => q.status === 'running' || q.status === 'reviewing')
                        if (running) return <><span className="activity-dot running" /><span>{t('adventurer.recent.running')}</span><em className="mono faint">#{running.short_id || shortId(running.id)}</em></>
                        const last = recentQuests[0]
                        return <><span className="activity-dot idle" /><span>{t('adventurer.recent.idle')}</span><em className="mono faint">{t('adventurer.recent.lastCompleted')}: {relativeTime(last.updated_at_ms || last.created_at_ms)}</em></>
                      })()}
                    </div>
                    <div className="activity-trend">
                      <span className="trend-label">{t('adventurer.recent.recentCount', { count: recentQuests.length })}</span>
                      <span className="trend-dots">
                        {recentQuests.slice().reverse().map((q, i) => (
                          <i key={i} className={'trend-dot ' + (q.status === 'success' ? 'ok' : q.status === 'failed' ? 'fail' : 'other')} title={q.query} />
                        ))}
                      </span>
                      <em className="mono faint">{t('adventurer.recent.successRate', { success: recentQuests.filter((q) => q.status === 'success').length, total: recentQuests.length })}</em>
                    </div>
                    {(() => {
                      const lastFail = recentQuests.find((q) => q.status === 'failed')
                      if (!lastFail) return null
                      return (
                        <div className="activity-last-fail" onClick={() => onOpenQuest(lastFail.id)}>
                          <span className="activity-dot fail" />
                          <span className="fail-query">{lastFail.query}</span>
                          <em className="mono faint">#{lastFail.short_id || shortId(lastFail.id)}</em>
                        </div>
                      )
                    })()}
                  </div>
                )}
              </div>
            </div>

            <aside className="adventurer-meta-panel">
              <div className="meta-section capability-matrix-panel">
                <div className="meta-section-label">{t('adventurer.field.capabilityMatrix')}</div>
                <div className="capability-matrix-list">
                  {agentOptions.map(({ agent, executor }) => (
                    <div className={'capability-matrix-row' + (agent === mergedForm.agent ? ' active' : '')} key={agent}>
                      <strong>{agentLabel(agent)}</strong>
                      <CapabilityTierBadge tier={executor.capability_tier} compact />
                      <span>{executor.capabilities?.includes('stream') || executor.capabilities?.includes('streaming') ? 'streaming' : 'observability varies'}</span>
                    </div>
                  ))}
                </div>
              </div>

              <div className="meta-section">
                <div className="meta-section-label">{t('adventurer.field.metadata')}</div>
                <div className="meta-kv-block">
                  <span>Agent</span>
                  <DesignSelect
                    compact
                    value={mergedForm.agent || ''}
                    options={[
                      { value: '', label: t('adventurer.field.agentSelectPlaceholder') },
                      ...agentOptions.map(({ agent }) => ({
                        value: agent,
                        label: agentLabel(agent),
                      })),
                    ]}
                    onChange={(v) => patch('agent', v)}
                    ariaLabel={t('aria.agent')}
                  />
                </div>
                <div className="meta-kv-block">
                  <span>{t('adventurer.field.capability')}</span>
                  <div className="meta-field-stack">
                    <CapabilityTierBadge tier={selectedExecutor?.capability_tier} />
                    <small className="field-hint">{t('adventurer.field.capabilityHint')}</small>
                  </div>
                </div>
                <div className="meta-kv-block">
                  <span>{t('adventurer.field.modelPolicy')}</span>
                  <div className="meta-field-stack">
                    <input
                      value={typeof mergedForm.model === 'string' ? mergedForm.model : ''}
                      onChange={(e) => patch('model', e.target.value)}
                      placeholder={t('adventurer.field.modelPlaceholder')}
                    />
                    <small className="field-hint">{adventurerModelPolicyLabel(typeof mergedForm.model === 'string' ? mergedForm.model : '', selectedExecutor)}</small>
                  </div>
                </div>
                <div className="meta-kv-block">
                  <span>{t('adventurer.field.class')}</span>
                  <em>{active.class === 'mage' ? t('world.mage') : t('world.warrior')}</em>
                </div>
                <div className="meta-kv-block">
                  <span>{t('adventurer.field.winRate')}</span>
                  <em className="mono">{winRate}% <span className="faint">({active.win_count}/{active.win_count + active.lose_count})</span></em>
                </div>
                <div className="meta-kv-block">
                  <span>{t('adventurer.field.exp')}</span>
                  <div className="xp-progress">
                    <div className="xp-row">
                      <span className="mono">{active.exp ?? 0} / {Math.ceil(((active.exp ?? 0) + 100) / 100) * 100} XP</span>
                      <span className="mono faint">Lv.{active.level} → Lv.{active.level + 1}</span>
                    </div>
                    <div className="xp-bar">
                      <i style={{ width: `${((active.exp ?? 0) % 100)}%` }} />
                    </div>
                  </div>
                </div>
              </div>

              <div className="meta-section border-top">
                <div className="meta-section-label">{t('adventurer.field.skills')}</div>
                <em className="mono meta-readonly">{
                  (() => {
                    const tools = Array.isArray(active.tools) && active.tools.length > 0
                      ? active.tools
                      : defaultSkills[active.class] || []
                    return tools.join(',\n')
                  })()
                }</em>
              </div>

              <div className="meta-section border-top">
                <div className="meta-kv">
                  <span>{t('adventurer.field.created')}</span>
                  <em className="mono">{active.created_at_ms ? relativeTime(active.created_at_ms) : '—'}</em>
                </div>
                <div className="meta-kv">
                  <span>{t('adventurer.field.updated')}</span>
                  <em className="mono">{active.updated_at_ms ? relativeTime(active.updated_at_ms as number) : '—'}</em>
                </div>
              </div>
            </aside>
          </>
        ) : (
          <div className="empty-page">{t('adventurer.selectPrompt')}</div>
        )}
      </main>
      {retireConfirm && active && (
        <div className="sheet-scrim" style={retireScrimStyle} role="dialog" aria-modal="true" aria-label={t('adventurer.retireDialog.title')}>
          <div className="sheet-card review-dialog" ref={retireSheetRef} style={retireSheetStyle} onClick={(e) => e.stopPropagation()}>
            <div className="sheet-head">
              <span className="sheet-icon warning">
                <Icon name="trash" />
              </span>
              <div>
                <h2>{t('adventurer.retireDialog.title')}</h2>
                <p>{t('adventurer.retireDialog.body')}</p>
              </div>
              <button className="icon-button" onClick={() => setRetireConfirm(false)} aria-label={t('aria.close')}>
                <Icon name="x" size={14} />
              </button>
            </div>
            <div className="review-dialog-body">
              <div className="settings-box">
                <strong>{active.name}</strong>
                <span className="dim-meta mono">{active.id}</span>
              </div>
            </div>
            <div className="sheet-foot">
              <button className="button ghost" onClick={() => setRetireConfirm(false)} disabled={retiring}>
                {t('adventurer.action.cancel')}
              </button>
              <button className="button danger" onClick={retire} disabled={retiring}>
                <Icon name="trash" />
                {retiring ? t('adventurer.action.retiring') : t('adventurer.action.confirmRetire')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div >
  )
}
