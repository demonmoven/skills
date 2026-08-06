import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api/client'
import type { AdventurerFile, AutomationConfig, AutomationDiscoveryArchive, ExecutorInfo } from '../api/types'
import AutomationPolicySummary, {
  AutomationFlow,
  automationFlowDescription,
  automationFlowIcon,
  automationFlowLabel,
  buildAutomationUpdatePayload,
  flowFromAutomationConfig,
  patchFromAutomationFlow,
} from '../components/AutomationPolicySummary'
import CronField from '../components/CronField'
import DesignSelect from '../components/DesignSelect'
import Icon from '../components/Icon'
import { agentKey, isRunnableAdventurer, relativeTime } from '../components/util'
import useDragToDismiss from '../hooks/useDragToDismiss'

type Props = {
  items: AutomationConfig[]
  selected: string | null
  onSelect: (id: string) => void
  onError: (msg: string) => void
  onChanged: () => void
  onAutomationUpdated: (item: AutomationConfig) => void
  adventurers: AdventurerFile[]
  executors: ExecutorInfo[]
  searchQuery: string
  onUseTemplate: (template: AutomationConfig) => void
}

function isContextAutomation(item: AutomationConfig | null | undefined) {
  return item?.id === 'auto_context_refresh'
}

function isKnowledgeAutomation(item: AutomationConfig | null | undefined) {
  return item?.id === 'auto_knowledge_pack_refresh' || !!item?.tags?.includes('knowledge')
}

function isOfficialAutomation(item: AutomationConfig | null | undefined) {
  return item?.source === 'official'
}

// Guarded automations have special runtime constraints (no auto_apply, no allow_l2)
// regardless of source — this is a functional/behavioral constraint, not a protection one.
function isGuardedAutomation(item: AutomationConfig | null | undefined) {
  return isContextAutomation(item) || isKnowledgeAutomation(item)
}

export default function Automations({
  items,
  selected,
  onSelect,
  onError,
  onChanged,
  onAutomationUpdated,
  adventurers,
  executors,
  searchQuery,
  onUseTemplate,
}: Props) {
  const { t } = useTranslation()
  const active = useMemo(
    () => items.find((a) => a.id === selected) || null,
    [items, selected],
  )
  const filtered = useMemo(() => {
    const visible = items
    const q = searchQuery.toLowerCase()
    if (!q.trim()) return visible
    return visible.filter(
      (a) =>
        a.name.toLowerCase().includes(q) ||
        (a.description || '').toLowerCase().includes(q),
    )
  }, [items, searchQuery])
  const [form, setForm] = useState<Partial<AutomationConfig>>({})
  const [enabledConnectors, setEnabledConnectors] = useState<string[]>([])
  const [discoveryArchive, setDiscoveryArchive] = useState<AutomationDiscoveryArchive[]>([])
  const [saving, setSaving] = useState(false)
  const [running, setRunning] = useState<string | null>(null)
  const [toggling, setToggling] = useState(false)
  const [selectMode, setSelectMode] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [batchRunning, setBatchRunning] = useState(false)
  const [batchResult, setBatchResult] = useState<{ action: 'enable' | 'disable'; success: number; failed: number } | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const { sheetRef: deleteSheetRef, sheetStyle: deleteSheetStyle, scrimStyle: deleteScrimStyle } =
    useDragToDismiss({ onDismiss: () => setDeleteConfirm(false), disabled: !deleteConfirm })

  useEffect(() => {
    setForm(active ? { ...active } : {})
    setDeleteConfirm(false)
  }, [active])

  useEffect(() => {
    let cancelled = false
    Promise.all([
      api.get<{ config?: { connectors?: { git?: { enabled?: boolean } } } }>('/api/settings'),
      api.get<{ items?: AutomationDiscoveryArchive[] }>('/api/automations/discovery-archive'),
    ]).then(([settings, archive]) => {
      if (cancelled) return
      const list: string[] = []
      if (settings.config?.connectors?.git?.enabled) list.push('git')
      setEnabledConnectors(list)
      setDiscoveryArchive(archive.items || [])
    }).catch(() => {})
    return () => {
      cancelled = true
    }
  }, [])

  function patch<K extends keyof AutomationConfig>(k: K, v: AutomationConfig[K]) {
    setForm((prev) => ({ ...prev, [k]: v }))
  }

  function patchFlow(flow: AutomationFlow) {
    const patch = patchFromAutomationFlow(flow)
    setForm((prev) => ({
      ...prev,
      quest_type: patch.quest_type as 'execute' | 'design',
      intensity: patch.intensity as 'quick' | 'standard',
      with_design_phase: patch.with_design_phase,
    }))
  }

  async function loadDiscoveryArchive() {
    try {
      const res = await api.get<{ items?: AutomationDiscoveryArchive[] }>('/api/automations/discovery-archive')
      setDiscoveryArchive(res.items || [])
    } catch {
      /* discovery archive is auxiliary */
    }
  }

  async function toggleEnabled(item: AutomationConfig) {
    const nextEnabled = !item.enabled
    setToggling(true)
    try {
      const res = await api.post<{ item?: AutomationConfig; enabled?: boolean }>(
        '/api/automations/' + item.id + (item.enabled ? '/disable' : '/enable'),
      )
      const updated = res.item
      const nextItem = updated || { ...item, enabled: res.enabled ?? nextEnabled }
      onAutomationUpdated(nextItem)
      setForm((prev) => ({
        ...prev,
        ...nextItem,
      }))
    } catch (e) {
      onError(e instanceof Error ? e.message : t('automation.error.update'))
    } finally {
      setToggling(false)
    }
  }

  const toggleSelectId = useCallback((id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])

  const selectAllVisible = useCallback(() => {
    setSelectedIds(new Set(filtered.map((a) => a.id)))
  }, [filtered])

  const clearSelection = useCallback(() => {
    setSelectedIds(new Set())
  }, [])

  const exitSelectMode = useCallback(() => {
    setSelectMode(false)
    setSelectedIds(new Set())
    setBatchResult(null)
  }, [])

  async function batchToggle(action: 'enable' | 'disable') {
    const targets = filtered.filter((a) => {
      if (!selectedIds.has(a.id)) return false
      if (action === 'enable') return !a.enabled
      return a.enabled
    })
    if (targets.length === 0) return

    setBatchRunning(true)
    setBatchResult(null)
    let success = 0
    let failed = 0

    await Promise.allSettled(
      targets.map(async (item) => {
        try {
          const endpoint = action === 'enable' ? '/enable' : '/disable'
          const res = await api.post<{ item?: AutomationConfig; enabled?: boolean }>(
            '/api/automations/' + item.id + endpoint,
          )
          const updated = res.item
          const nextItem = updated || { ...item, enabled: res.enabled ?? (action === 'enable') }
          onAutomationUpdated(nextItem)
          success++
        } catch {
          failed++
        }
      }),
    )

    setBatchRunning(false)
    setBatchResult({ action, success, failed })
    if (failed === 0) {
      setSelectedIds(new Set())
    }
  }

  async function runNow(item: AutomationConfig) {
    setRunning(item.id)
    try {
      await api.post('/api/automations/' + item.id + '/run')
      await loadDiscoveryArchive()
      onChanged()
    } catch (e) {
      onError(e instanceof Error ? e.message : t('automation.error.run'))
    } finally {
      setRunning(null)
    }
  }

  async function removeActive() {
    if (!active) return
    setDeleting(true)
    try {
      await api.delete('/api/automations/' + active.id)
      setDeleteConfirm(false)
      onChanged()
    } catch (e) {
      onError(e instanceof Error ? e.message : t('automation.error.delete'))
    } finally {
      setDeleting(false)
    }
  }

  async function save() {
    if (!active) return
    setSaving(true)
    const guardedAutomation = isGuardedAutomation(active)
    const previousFlow = flowFromAutomationConfig({
      quest_type: active?.quest_type,
      intensity: active?.intensity,
      with_design_phase: active?.with_design_phase,
    })
    try {
      const payload = buildAutomationUpdatePayload(
        {
          name: form.name,
          description: form.description,
          query: form.query,
          quest_type: form.quest_type,
          with_design_phase: form.with_design_phase,
          intensity: form.intensity,
          working_dir: form.working_dir,
          trigger: form.trigger,
          cron: form.cron,
          auto_start: form.auto_start,
          auto_apply: form.auto_apply,
          allow_l2: form.allow_l2,
          auto_spawn_execute: form.auto_spawn_execute,
          warrior_id: form.warrior_id,
          mage_id: form.mage_id,
          connectors: form.connectors,
          priority: form.priority,
        },
        activeFlow,
        previousFlow,
        active.auto_spawn_execute,
        canPlanOnlyToggle,
        guardedAutomation,
        active.auto_start,
      )
      await api.patch(`/api/automations/${active.id}`, payload)
      onChanged()
    } catch (e) {
      onError(e instanceof Error ? e.message : t('automation.error.save'))
    } finally {
      setSaving(false)
    }
  }

  const trigger = form.trigger || active?.trigger || 'manual'
  const activeIsContextAutomation = isContextAutomation(active)
  const activeIsKnowledgeAutomation = isKnowledgeAutomation(active)
  const activeIsOfficial = isOfficialAutomation(active)
  const activeIsGuardedAutomation = isGuardedAutomation(active)
  const activeFlow: AutomationFlow = flowFromAutomationConfig({
    quest_type: form.quest_type || active?.quest_type,
    intensity: form.intensity || active?.intensity,
    with_design_phase: form.with_design_phase ?? active?.with_design_phase,
  })
  const canPlanOnlyToggle = activeFlow === 'goal' && (form.intensity || active?.intensity || 'standard') !== 'quick'
  const warriors = adventurers.filter((a) => a.class === 'warrior' && isRunnableAdventurer(a, executors))
  const mages = adventurers.filter((a) => a.class === 'mage' && isRunnableAdventurer(a, executors))
  const selectedWarriorID = form.warrior_id || active?.warrior_id || ''
  const selectedWarrior = adventurers.find((item) => item.id === selectedWarriorID)
  const selectedExecutor = selectedWarrior ? executors.find((executor) => agentKey(executor) === selectedWarrior.agent) : null
  const activeDiscoveryArchive = discoveryArchive
    .filter((item) => item.automation_id === active?.id)
    .slice(-5)
    .reverse()

  return (
    <div className="configured-page automation-page">
      {selectMode && (
        <div className="automation-list-toolbar mobile-only">
          <span className="batch-info">
            {t('automation.batch.selectedCount', { count: selectedIds.size, total: filtered.length })}
          </span>
          <button type="button" className="button ghost compact" onClick={selectAllVisible} disabled={batchRunning}>
            {t('automation.batch.selectAll')}
          </button>
          <button type="button" className="button ghost compact" onClick={exitSelectMode} disabled={batchRunning}>
            {t('automation.batch.exitSelect')}
          </button>
        </div>
      )}
      <div className="mobile-pill-switcher">
        {filtered.map((item) => (
          <button
            key={item.id}
            className={'pill-item' + (item.id === active?.id ? ' active' : '') + (selectMode && selectedIds.has(item.id) ? ' selected' : '')}
            onClick={() => {
              if (selectMode) {
                toggleSelectId(item.id)
              } else {
                onSelect(item.id)
              }
            }}
          >
            {selectMode && (
              <span className={'batch-check' + (selectedIds.has(item.id) ? ' checked' : '')}>
                <Icon name="check" size={10} />
              </span>
            )}
            <Icon name={item.enabled ? 'zap' : 'shield'} size={12} />
            <span>{item.name}</span>
          </button>
        ))}
      </div>
      <aside className="configured-list">
        <div className="automation-list-toolbar desktop-only">
          {!selectMode ? (
            <button
              type="button"
              className="button ghost compact"
              onClick={() => setSelectMode(true)}
              disabled={filtered.length === 0}
            >
              <Icon name="check-square" />
              {t('automation.batch.enterSelect')}
            </button>
          ) : (
            <>
              <span className="batch-info">
                {t('automation.batch.selectedCount', { count: selectedIds.size, total: filtered.length })}
              </span>
              <button type="button" className="button ghost compact" onClick={selectAllVisible} disabled={batchRunning}>
                {t('automation.batch.selectAll')}
              </button>
              <button type="button" className="button ghost compact" onClick={clearSelection} disabled={batchRunning || selectedIds.size === 0}>
                {t('automation.batch.clear')}
              </button>
              <button type="button" className="button ghost compact" onClick={exitSelectMode} disabled={batchRunning}>
                {t('automation.batch.exitSelect')}
              </button>
            </>
          )}
        </div>
        {filtered.map((item) => (
          <button
            type="button"
            className={
              'configured-list-item ' +
              (item.id === active?.id ? 'active ' : '') +
              (!item.enabled ? 'disabled ' : '') +
              (selectMode && selectedIds.has(item.id) ? 'row-selected ' : '')
            }
            key={item.id}
            onClick={() => {
              if (selectMode) {
                toggleSelectId(item.id)
              } else {
                onSelect(item.id)
              }
            }}
          >
            {selectMode && (
              <span className={'batch-check' + (selectedIds.has(item.id) ? ' checked' : '')}>
                <Icon name="check" size={12} />
              </span>
            )}
            <Icon name={item.enabled ? 'zap' : 'shield'} />
            <span>
              <strong>
                {item.name}
                {item.enabled ? (
                  <i className="status-dot ok" />
                ) : (
                  <i className="status-dot ink" />
                )}
              </strong>
              <small>
                {item.trigger === 'schedule' ? t('automation.meta.schedule') : t('automation.meta.manual')} · {item.enabled ? t('automation.meta.enabled') : t('automation.meta.disabled')}
                {isContextAutomation(item) ? ` · ${t('automation.meta.context')}` : ''}
                {isKnowledgeAutomation(item) ? ` · ${t('automation.meta.knowledge')}` : ''}
                {isOfficialAutomation(item) ? ` · ${t('automation.meta.official')}` : ''}
              </small>
            </span>
            {(item.run_count ?? 0) > 0 && (
              <em className="automation-run-count mono">{t('automation.meta.runCount', { count: item.run_count })}</em>
            )}
          </button>
        ))}
        {filtered.length === 0 && <div className="empty-page">{searchQuery ? t('automation.list.noMatch') : t('automation.list.empty')}</div>}
      </aside>

      <main className="configured-detail automation-detail">
        {active ? (
          <div className="form-surface">
            <header className="form-head">
              <h2>
                {form.name || active.name}
                {activeIsContextAutomation && <span className="automation-title-meta">{t('automation.meta.contextRefresh')}</span>}
                {activeIsKnowledgeAutomation && <span className="automation-title-meta">{t('automation.meta.knowledge')}</span>}
              </h2>
              <div className="row-actions">
                {activeIsOfficial && (
                  <button
                    className="button"
                    onClick={() => onUseTemplate(active)}
                  >
                    <Icon name="copy" />
                    {t('automation.edit.createCopy')}
                  </button>
                )}
                <button
                  className="button"
                  onClick={() => runNow(active)}
                  disabled={running === active.id || !active?.enabled}
                  title={
                    activeIsKnowledgeAutomation
                      ? t('automation.edit.runTooltipKnowledge')
                      : activeIsOfficial && !activeIsContextAutomation
                        ? t('automation.edit.runTooltipOfficial')
                      : activeIsContextAutomation
                        ? t('automation.edit.runTooltipContext')
                        : undefined
                  }
                >
                  <Icon name="play" />
                  {running === active.id ? t('automation.edit.running') : t('automation.edit.runNow')}
                </button>
                <button className="button primary" onClick={save} disabled={saving || activeIsOfficial}>
                  {saving ? t('automation.edit.saving') : t('automation.edit.save')}
                </button>
                <button
                  className="button danger ghost"
                  onClick={() => setDeleteConfirm(true)}
                  disabled={deleting || activeIsOfficial}
                  title={activeIsOfficial ? t('automation.edit.deleteTooltip') : undefined}
                >
                  <Icon name="trash" />
                  {t('automation.edit.delete')}
                </button>
              </div>
            </header>

            <div className="enable-row">
              <div>
                <strong>{t('automation.edit.enable')}</strong>
                <span>{t('automation.edit.enableDesc')}</span>
              </div>
              <button
                type="button"
                className={'toggle-switch ' + ((form.enabled ?? active.enabled) ? 'on' : '')}
                onClick={() => toggleEnabled(active)}
                disabled={toggling}
                aria-label={t('automation.edit.enableAria')}
              />
            </div>
            {activeIsOfficial && (
              <div className="form-banner">
                <Icon name="shield" />
                <span>
                  {activeIsGuardedAutomation
                    ? t('automation.edit.officialGuarded')
                    : t('automation.edit.officialReadonly')}
                </span>
              </div>
            )}
            <AutomationPolicySummary
              input={{
                triageMode: form.triage_mode || active.triage_mode,
                autoStart: form.auto_start ?? active.auto_start,
                autoApply: activeIsGuardedAutomation ? false : (form.auto_apply ?? active.auto_apply),
                allowL2: activeIsGuardedAutomation ? false : (form.allow_l2 ?? active.allow_l2),
                trustTier: form.trust_tier || active.trust_tier,
                connectors: form.connectors || active.connectors || [],
                enabledConnectors,
                capabilityTier: selectedExecutor?.capability_tier,
              }}
            />

            <label className="field">
              <span>{t('automation.edit.query')}</span>
              <textarea
                rows={8}
                value={form.query || ''}
                onChange={(e) => patch('query', e.target.value)}
                placeholder={t('automation.edit.queryPlaceholder')}
                disabled={activeIsOfficial}
              />
            </label>

            <div className="grid-2">
              <label className="field flow-field">
                <span>{t('automation.edit.flow')}</span>
                <div className="option-cards flow-options">
                  <button
                    type="button"
                    className={'option-card ' + (activeFlow === 'direct' ? 'selected' : '')}
                    onClick={() => patchFlow('direct')}
                    disabled={activeIsOfficial}
                  >
                    <Icon name={automationFlowIcon('direct')} />
                    <div>
                      <strong>{automationFlowLabel('direct')}</strong>
                      <em>{automationFlowDescription('direct')}</em>
                    </div>
                  </button>
                  <button
                    type="button"
                    className={'option-card ' + (activeFlow === 'checked' ? 'selected' : '')}
                    onClick={() => patchFlow('checked')}
                    disabled={activeIsOfficial}
                  >
                    <Icon name={automationFlowIcon('checked')} />
                    <div>
                      <strong>{automationFlowLabel('checked')}</strong>
                      <em>{automationFlowDescription('checked')}</em>
                    </div>
                  </button>
                  <button
                    type="button"
                    className={'option-card ' + (activeFlow === 'goal' ? 'selected' : '')}
                    onClick={() => patchFlow('goal')}
                    disabled={activeIsOfficial}
                  >
                    <Icon name={automationFlowIcon('goal')} />
                    <div>
                      <strong>{automationFlowLabel('goal')}</strong>
                      <em>{automationFlowDescription('goal')}</em>
                    </div>
                  </button>
                </div>
              </label>
              <label className="field">
                <span>{t('automation.edit.workingDir')}</span>
                <input
                  value={form.working_dir || ''}
                  onChange={(e) => patch('working_dir', e.target.value)}
                  placeholder="/path/to/project"
                  className="mono"
                  disabled={activeIsOfficial}
                />
              </label>

            </div>

            <div className="grid-2">
              <label className="field choice-field">
                <span>{t('automation.edit.warrior')}</span>
                <DesignSelect
                  compact
                  value={form.warrior_id || active.warrior_id || ''}
                  options={[
                    { value: '', label: t('automation.edit.autoAssign') },
                    ...warriors.map((a) => ({ value: a.id, label: a.name })),
                  ]}
                  onChange={(v) => patch('warrior_id', v)}
                  ariaLabel={t('aria.warrior')}
                  disabled={activeIsOfficial}
                />
              </label>
              <label className="field choice-field">
                <span>{t('automation.edit.mage')}</span>
                <DesignSelect
                  compact
                  value={form.mage_id || active.mage_id || ''}
                  options={[
                    { value: '', label: t('automation.edit.autoAssign') },
                    ...mages.map((a) => ({ value: a.id, label: a.name })),
                  ]}
                  onChange={(v) => patch('mage_id', v)}
                  ariaLabel={t('aria.mage')}
                  disabled={activeIsOfficial}
                />
              </label>
            </div>

            <label className="field">
              <span>{t('automation.policy.triage')}</span>
              <div className="option-cards two-col">
                <button
                  type="button"
                  className={
                    'option-card ' +
                    (!(form.auto_start ?? active.auto_start) ? 'selected' : '')
                  }
                  onClick={() => patch('auto_start', false)}
                  disabled={activeIsOfficial}
                >
                  <Icon name="inbox" />
                  <div>
                    <strong>{t('automation.value.triageCandidate')}</strong>
                    <em>{activeIsContextAutomation ? t('automation.edit.triageCandidateDescContext') : activeIsKnowledgeAutomation ? t('automation.edit.triageCandidateDescKnowledge') : t('automation.edit.triageCandidateDescDefault')}</em>
                  </div>
                </button>
                <button
                  type="button"
                  className={
                    'option-card ' +
                    ((form.auto_start ?? active.auto_start) ? 'selected' : '')
                  }
                  onClick={() => patch('auto_start', true)}
                  disabled={activeIsOfficial}
                >
                  <Icon name="layout-grid" />
                  <div>
                    <strong>{activeIsContextAutomation ? t('automation.edit.triageDirectContext') : activeIsKnowledgeAutomation ? t('automation.edit.triageDirectKnowledge') : t('automation.edit.triageDirectDefault')}</strong>
                    <em>{activeIsContextAutomation ? t('automation.edit.triageDirectDescContext') : activeIsKnowledgeAutomation ? t('automation.edit.triageDirectDescKnowledge') : t('automation.edit.triageDirectDescDefault')}</em>
                  </div>
                </button>
              </div>
            </label>

            <div className="auto-apply-row">
              <div>
                <strong>{t('automation.edit.autoApply')}</strong>
                <em>
                  {activeIsGuardedAutomation
                    ? t('automation.edit.autoApplyDescGuarded')
                    : t('automation.edit.autoApplyDesc')}
                </em>
              </div>
              <button
                type="button"
                className={'toggle-switch ' + (!activeIsGuardedAutomation && (form.auto_apply ?? active.auto_apply) ? 'on' : '')}
                onClick={() => {
                  if (!activeIsGuardedAutomation) {
                    const next = !(form.auto_apply ?? active.auto_apply)
                    patch('auto_apply', next)
                    if (next) patch('allow_l2', false)
                  }
                }}
                disabled={activeIsOfficial || activeIsGuardedAutomation}
                aria-label={t('automation.edit.autoApplyAria')}
              />
            </div>

            {canPlanOnlyToggle && (
              <div className="auto-apply-row">
                <div>
                  <strong>{t('automation.edit.planOnly')}</strong>
                  <em>{t('automation.edit.planOnlyDesc')}</em>
                </div>
                <button
                  type="button"
                  className={'toggle-switch ' + (!(form.auto_spawn_execute ?? active.auto_spawn_execute) ? 'on' : '')}
                  onClick={() => patch('auto_spawn_execute', !(form.auto_spawn_execute ?? active.auto_spawn_execute))}
                  disabled={activeIsOfficial}
                  aria-label={t('automation.edit.planOnlyAria')}
                />
              </div>
            )}

            <div className="auto-apply-row">
              <div>
                <strong>{t('automation.edit.allowL2')}</strong>
                <em>{t('automation.edit.allowL2Desc')}</em>
              </div>
              <button
                type="button"
                className={'toggle-switch ' + (!activeIsGuardedAutomation && (form.allow_l2 ?? active.allow_l2) ? 'on' : '')}
                onClick={() => {
                  if (!activeIsGuardedAutomation) {
                    const next = !(form.allow_l2 ?? active.allow_l2)
                    patch('allow_l2', next)
                    if (next) patch('auto_apply', false)
                  }
                }}
                disabled={activeIsOfficial || activeIsGuardedAutomation}
                aria-label={t('automation.edit.allowL2Aria')}
              />
            </div>

            <label className="field">
              <span>{t('automation.edit.trigger')}</span>
              <div className="option-cards two-col">
                <button
                  type="button"
                  className={'option-card ' + (trigger === 'schedule' ? 'selected' : '')}
                  onClick={() => patch('trigger', 'schedule')}
                  disabled={activeIsOfficial}
                >
                  <Icon name="clock" />
                  <div>
                    <strong>{t('automation.edit.triggerSchedule')}</strong>
                    <em>{t('automation.edit.triggerScheduleDesc')}</em>
                  </div>
                </button>
                <button
                  type="button"
                  className={'option-card ' + (trigger === 'manual' ? 'selected' : '')}
                  onClick={() => patch('trigger', 'manual')}
                  disabled={activeIsOfficial}
                >
                  <Icon name="hand" />
                  <div>
                    <strong>{t('automation.edit.triggerManual')}</strong>
                    <em>{t('automation.edit.triggerManualDesc')}</em>
                  </div>
                </button>
              </div>
            </label>

            {trigger === 'schedule' && (
              <CronField
                value={form.cron || ''}
                onChange={(value) => patch('cron', value)}
                disabled={activeIsOfficial}
              />
            )}

            <label className="field inline-field">
              <span>{t('automation.edit.priority')}</span>
              <input
                type="number"
                value={form.priority ?? 0}
                onChange={(e) => patch('priority', Number(e.target.value))}
                disabled={activeIsOfficial}
              />
            </label>

            {enabledConnectors.length > 0 && (
              <div className="auto-apply-row connectors-row">
                <div>
                  <strong>{t('automation.edit.connectorsLabel')}</strong>
                  <em>{t('automation.edit.connectorsDesc')}</em>
                </div>
                <div className="connectors-options">
                  {enabledConnectors.map((name) => (
                    <label key={name} className="connector-chip">
                      <input
                        type="checkbox"
                        checked={(form.connectors || []).includes(name)}
                        disabled={activeIsOfficial}
                        onChange={(e) => {
                          const prev = form.connectors || []
                          patch(
                            'connectors',
                            e.target.checked ? [...prev, name] : prev.filter((c) => c !== name),
                          )
                        }}
                      />
                      <Icon name="git-merge" size={13} />
                      {name === 'git' ? t('automation.edit.connectorGit') : name}
                    </label>
                  ))}
                </div>
              </div>
            )}
            <section className="automation-discovery-panel">
              <div className="panel-title">
                <h2>{t('automation.discovery.title')}</h2>
                <span>{t('automation.discovery.count', { count: activeDiscoveryArchive.length })}</span>
              </div>
              {activeDiscoveryArchive.length > 0 ? (
                <div className="automation-discovery-list">
                  {activeDiscoveryArchive.map((item) => (
                    <div className="automation-discovery-row" key={item.id || String(item.created_at_ms)}>
                      <strong>{item.summary || item.reason || t('automation.discovery.noFinding')}</strong>
                      <span className="mono">{item.outcome}</span>
                      {item.created_at_ms && <em>{relativeTime(item.created_at_ms)}</em>}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="automation-discovery-empty">{t('automation.discovery.empty')}</div>
              )}
            </section>
          </div>
        ) : (
          <div className="empty-page">{t('automation.list.selectOne')}</div>
        )}
      </main>
      {deleteConfirm && active && (
        <div className="sheet-scrim" style={deleteScrimStyle} role="dialog" aria-modal="true" aria-label={t('automation.delete.aria')}>
          <div className="sheet-card review-dialog" ref={deleteSheetRef} style={deleteSheetStyle} onClick={(e) => e.stopPropagation()}>
            <div className="sheet-head">
              <span className="sheet-icon warning">
                <Icon name="trash" />
              </span>
              <div>
                <h2>{t('automation.delete.title')}</h2>
                <p>{t('automation.delete.body')}</p>
              </div>
              <button className="icon-button" onClick={() => setDeleteConfirm(false)} aria-label={t('aria.close')}>
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
              <button className="button ghost" onClick={() => setDeleteConfirm(false)} disabled={deleting}>
                {t('automation.delete.cancel')}
              </button>
              <button className="button danger" onClick={removeActive} disabled={deleting}>
                <Icon name="trash" />
                {deleting ? t('automation.delete.deleting') : t('automation.delete.confirm')}
              </button>
            </div>
          </div>
        </div>
      )}
      {selectMode && (
        <div className="batch-action-bar">
          <div className="batch-action-info">
            <strong>{t('automation.batch.selectedCount', { count: selectedIds.size, total: filtered.length })}</strong>
            {batchResult && (
              <span className={'batch-result ' + (batchResult.failed > 0 ? 'has-error' : 'ok')}>
                {batchResult.failed > 0
                  ? t('automation.batch.resultPartial', { action: t(batchResult.action === 'enable' ? 'automation.batch.actionEnable' : 'automation.batch.actionDisable'), success: batchResult.success, failed: batchResult.failed })
                  : t('automation.batch.resultAll', { action: t(batchResult.action === 'enable' ? 'automation.batch.actionEnable' : 'automation.batch.actionDisable'), count: batchResult.success })
                }
              </span>
            )}
          </div>
          <div className="batch-action-buttons">
            <button
              type="button"
              className="button"
              onClick={() => batchToggle('disable')}
              disabled={batchRunning || selectedIds.size === 0}
            >
              <Icon name="shield" />
              {batchRunning ? t('automation.batch.running') : t('automation.batch.disableSelected')}
            </button>
            <button
              type="button"
              className="button primary"
              onClick={() => batchToggle('enable')}
              disabled={batchRunning || selectedIds.size === 0}
            >
              <Icon name="zap" />
              {batchRunning ? t('automation.batch.running') : t('automation.batch.enableSelected')}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
