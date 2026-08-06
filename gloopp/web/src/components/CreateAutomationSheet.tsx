import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api/client'
import type { AdventurerFile, AutomationConfig, ExecutorInfo, QuestIntensity } from '../api/types'
import { agentKey, isRunnableAdventurer } from './util'
import AdvancedSection from './AdvancedSection'
import AutomationPolicySummary, {
  AutomationFlow,
  automationFlowDescription,
  automationFlowIcon,
  automationFlowLabel,
  buildAutomationCreatePayload,
  flowFromAutomationConfig,
  patchFromAutomationFlow,
} from './AutomationPolicySummary'
import CronField from './CronField'
import DesignSelect from './DesignSelect'
import Icon from './Icon'
import useDragToDismiss from '../hooks/useDragToDismiss'

function initialAutomationFlow(initial?: Partial<AutomationConfig>): AutomationFlow {
  return flowFromAutomationConfig({
    quest_type: initial?.quest_type,
    intensity: initial?.intensity,
    with_design_phase: initial?.with_design_phase,
  })
}

export default function CreateAutomationSheet({
  adventurers,
  executors,
  onClose,
  onError,
  onCreated,
  initial,
}: {
  adventurers: AdventurerFile[]
  executors: ExecutorInfo[]
  onClose: () => void
  onError: (msg: string) => void
  onCreated: (a: AutomationConfig) => void
  initial?: Partial<AutomationConfig>
}) {
  const { t } = useTranslation()
  const [name, setName] = useState(initial?.name || '')
  const [description, setDescription] = useState(initial?.description || '')
  const [query, setQuery] = useState(initial?.query || '')
  const [workingDir, setWorkingDir] = useState(initial?.working_dir || '')
  const [flow, setFlow] = useState<AutomationFlow>(initialAutomationFlow(initial))
  const [trigger, setTrigger] = useState<'manual' | 'schedule'>(
    initial?.trigger === 'manual' ? 'manual' : 'schedule',
  )
  const [cron, setCron] = useState(initial?.cron || 'daily')
  const [warriorId, setWarriorId] = useState(initial?.warrior_id || '')
  const [mageId, setMageId] = useState(initial?.mage_id || '')
  const [autoStart, setAutoStart] = useState(initial?.auto_start ?? true)
  const [autoApply, setAutoApply] = useState(initial?.auto_apply ?? false)
  const [allowL2, setAllowL2] = useState(initial?.allow_l2 ?? false)
  const [planOnly, setPlanOnly] = useState(
    initial?.quest_type === 'design' ? !(initial?.auto_spawn_execute ?? true) : false,
  )
  const [enabledConnectors, setEnabledConnectors] = useState<string[]>([])
  const [selectedConnectors, setSelectedConnectors] = useState<string[]>(() => initial?.connectors || [])
  const [sending, setSending] = useState(false)
  const [fieldErr, setFieldErr] = useState('')

  const warriors = adventurers.filter((a) => a.class === 'warrior' && isRunnableAdventurer(a, executors))
  const mages = adventurers.filter((a) => a.class === 'mage' && isRunnableAdventurer(a, executors))
  const { sheetRef, handleRef, sheetStyle, scrimStyle } = useDragToDismiss({ onDismiss: onClose })
  const warriorOptions = [
    { value: '', label: t('automation.create.warriorAuto') },
    ...warriors.map((a) => ({ value: a.id, label: a.name })),
  ]
  const mageOptions = [
    { value: '', label: t('automation.create.mageAuto') },
    ...mages.map((a) => ({ value: a.id, label: a.name })),
  ]
  const flowPatch = patchFromAutomationFlow(flow)
  const flowIntensity = flowPatch.intensity as QuestIntensity
  const canPlanOnlyToggle = flow === 'goal' && flowIntensity !== 'quick'
  const selectedWarrior = adventurers.find((item) => item.id === warriorId)
  const selectedExecutor = selectedWarrior ? executors.find((executor) => agentKey(executor) === selectedWarrior.agent) : null

  const submit = useCallback(async () => {
    if (!name.trim() || !query.trim()) {
      setFieldErr(t('automation.create.errorRequired'))
      return
    }
    setFieldErr('')
    setSending(true)
    try {
      const payload = buildAutomationCreatePayload({
        name,
        description,
        query,
        workingDir,
        trigger,
        cron,
        warriorId,
        mageId,
        autoStart,
        autoApply,
        allowL2,
        planOnly,
        connectors: selectedConnectors,
      }, flow)
      const res = await api.post<{ item: AutomationConfig }>('/api/automations', payload)
      onCreated(res.item)
      onClose()
    } catch (e) {
      onError(e instanceof Error ? e.message : t('automation.create.errorCreate'))
    } finally {
      setSending(false)
    }
  }, [
    name,
    description,
    query,
    workingDir,
    flow,
    flowIntensity,
    trigger,
    cron,
    warriorId,
    mageId,
    autoStart,
    autoApply,
    allowL2,
    planOnly,
    canPlanOnlyToggle,
    selectedConnectors,
    onCreated,
    onClose,
    onError,
  ])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      }
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
        // 只在非 textarea 里允许快捷键提交，避免误触
        const tag = (e.target as HTMLElement | null)?.tagName
        if (tag !== 'TEXTAREA') {
          e.preventDefault()
          submit()
        }
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose, submit])

  useEffect(() => {
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prev
    }
  }, [])

  useEffect(() => {
    let cancelled = false
    api
      .get<{ config?: { connectors?: { git?: { enabled?: boolean } } } }>('/api/settings')
      .then((res) => {
        if (cancelled) return
        const list: string[] = []
        if (res.config?.connectors?.git?.enabled) list.push('git')
        setEnabledConnectors(list)
      })
      .catch(() => {
        /* 静默失败：connector 选项不阻塞创建 */
      })
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <div className="sheet-scrim" style={scrimStyle} onClick={onClose} role="dialog" aria-modal="true" aria-label={t('automation.create.aria')}>
      <div className="sheet-card design-sheet" ref={sheetRef} style={sheetStyle} onClick={(e) => e.stopPropagation()}>
        <i className="sheet-handle" ref={handleRef} aria-hidden="true" />
        <div className="sheet-head design-sheet-head">
          <div className="sheet-title-row">
            <span className="sheet-icon">
              <Icon name="bolt" />
            </span>
            <div>
              <div className="sheet-title-line">
                <h2>{t('automation.create.title')}</h2>
                <span className="sheet-badge">{t('automation.create.badge')}</span>
              </div>
              <p className="sheet-subtitle">{t('automation.create.subtitle')}</p>
            </div>
          </div>
          <button className="icon-button" onClick={onClose} aria-label={t('aria.close')}>
            <Icon name="x" />
          </button>
        </div>
        <div className="sheet-body design-sheet-body">
          {fieldErr && (
            <div className="form-banner error" role="alert">
              <Icon name="error" />
              {fieldErr}
            </div>
          )}
          <div className="grid-2 design-grid-tight">
            <label className="field">
              <span>{t('automation.create.name')}</span>
              <input value={name} onChange={(e) => setName(e.target.value)} placeholder={t('automation.create.namePlaceholder')} autoFocus />
            </label>
            <label className="field">
              <span>{t('automation.create.description')}</span>
              <input
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder={t('automation.create.descPlaceholder')}
              />
            </label>
          </div>
          <label className="field">
            <span className="field-label-row">
              <span>{t('automation.create.query')}</span>
              <em>{t('automation.create.queryHint')}</em>
            </span>
            <textarea
              rows={5}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={t('automation.create.queryPlaceholder')}
            />
          </label>
          <div className="field">
            <span>{t('automation.create.trigger')}</span>
            <div className="option-cards two-col">
              <button
                type="button"
                className={'option-card ' + (trigger === 'manual' ? 'selected' : '')}
                onClick={() => setTrigger('manual')}
              >
                <Icon name="hand" />
                <div>
                  <strong>{t('automation.create.triggerManual')}</strong>
                  <em>{t('automation.create.triggerManualDesc')}</em>
                </div>
              </button>
              <button
                type="button"
                className={'option-card ' + (trigger === 'schedule' ? 'selected' : '')}
                onClick={() => setTrigger('schedule')}
              >
                <Icon name="clock" />
                <div>
                  <strong>{t('automation.create.triggerSchedule')}</strong>
                  <em>{t('automation.create.triggerScheduleDesc')}</em>
                </div>
              </button>
            </div>
            {trigger === 'schedule' && <CronField value={cron} onChange={setCron} />}
          </div>

          <div className="form-grid">
            <div className="field flow-field">
              <span>{t('automation.create.flow')}</span>
              <div className="option-cards flow-options">
                <button
                  type="button"
                  className={'option-card ' + (flow === 'direct' ? 'selected' : '')}
                  onClick={() => setFlow('direct')}
                >
                  <Icon name={automationFlowIcon('direct')} />
                  <div>
                    <strong>{automationFlowLabel('direct')}</strong>
                    <em>{automationFlowDescription('direct')}</em>
                  </div>
                </button>
                <button
                  type="button"
                  className={'option-card ' + (flow === 'checked' ? 'selected' : '')}
                  onClick={() => setFlow('checked')}
                >
                  <Icon name={automationFlowIcon('checked')} />
                  <div>
                    <strong>{automationFlowLabel('checked')}</strong>
                    <em>{automationFlowDescription('checked')}</em>
                  </div>
                </button>
                <button
                  type="button"
                  className={'option-card ' + (flow === 'goal' ? 'selected' : '')}
                  onClick={() => setFlow('goal')}
                >
                  <Icon name={automationFlowIcon('goal')} />
                  <div>
                    <strong>{automationFlowLabel('goal')}</strong>
                    <em>{automationFlowDescription('goal')}</em>
                  </div>
                </button>
              </div>
            </div>

          </div>

          <AutomationPolicySummary
            compact
            input={{
              autoStart,
              triageMode: autoStart ? 'direct' : 'candidate',
              autoApply,
              allowL2,
              connectors: selectedConnectors,
              enabledConnectors,
              capabilityTier: selectedExecutor?.capability_tier,
            }}
          />


          <AdvancedSection hint={t('automation.create.advancedHint')}>
            <div className="grid-2 design-grid-tight">
              <label className="field flow-field">
                <span>{t('automation.create.workingDir')}</span>
                <input
                  value={workingDir}
                  onChange={(e) => setWorkingDir(e.target.value)}
                  placeholder={t('automation.create.workingDirPlaceholder')}
                  className="mono"
                />
              </label>
            </div>

            <label className="field">
              <span>{t('automation.policy.triage')}</span>
              <div className="option-cards two-col">
                <button
                  type="button"
                  className={'option-card ' + (!autoStart ? 'selected' : '')}
                  onClick={() => setAutoStart(false)}
                >
                  <Icon name="inbox" />
                  <div>
                    <strong>{t('automation.create.triageCandidate')}</strong>
                    <em>{t('automation.create.triageCandidateDesc')}</em>
                  </div>
                </button>
                <button
                  type="button"
                  className={'option-card ' + (autoStart ? 'selected' : '')}
                  onClick={() => setAutoStart(true)}
                >
                  <Icon name="send" />
                  <div>
                    <strong>{t('automation.create.triageDirect')}</strong>
                    <em>{t('automation.create.triageDirectDesc')}</em>
                  </div>
                </button>
              </div>
            </label>

            <label className="check-row launch-row">
              <input
                type="checkbox"
                checked={autoApply}
                onChange={(e) => {
                  setAutoApply(e.target.checked)
                  if (e.target.checked) setAllowL2(false)
                }}
              />
              <span>{t('automation.create.autoApply')}</span>
              <em>{t('automation.create.autoApplyDesc')}</em>
            </label>

            {canPlanOnlyToggle && (
              <label className="check-row launch-row">
                <input
                  type="checkbox"
                  checked={planOnly}
                  onChange={(e) => setPlanOnly(e.target.checked)}
                />
                <span>{t('automation.create.planOnly')}</span>
                <em>{t('automation.create.planOnlyDesc')}</em>
              </label>
            )}
            <label className="check-row launch-row">
              <input
                type="checkbox"
                checked={allowL2}
                onChange={(e) => {
                  setAllowL2(e.target.checked)
                  if (e.target.checked) setAutoApply(false)
                }}
              />
              <span>{t('automation.create.allowL2')}</span>
              <em>{t('automation.create.allowL2Desc')}</em>
            </label>
            <div className="grid-2">
              <label className="field">
                <span>{t('automation.create.warriorLabel')}</span>
                <DesignSelect
                  value={warriorId}
                  options={warriorOptions}
                  onChange={setWarriorId}
                  ariaLabel={t('aria.warrior')}
                />
              </label>
              <label className="field">
                <span>{t('automation.create.mageLabel')}</span>
                <DesignSelect
                  value={mageId}
                  options={mageOptions}
                  onChange={setMageId}
                  ariaLabel={t('aria.mage')}
                />
              </label>
            </div>

            {enabledConnectors.length > 0 && (
              <div className="check-row launch-row connectors-row">
                <span className="connectors-label">{t('automation.create.connectorsLabel')}</span>
                <div className="connectors-options">
                  {enabledConnectors.map((name) => (
                    <label key={name} className="connector-chip">
                      <input
                        type="checkbox"
                        checked={selectedConnectors.includes(name)}
                        onChange={(e) => {
                          setSelectedConnectors((prev) =>
                            e.target.checked ? [...prev, name] : prev.filter((c) => c !== name),
                          )
                        }}
                      />
                      <Icon name="git-merge" size={13} />
                      {name === 'git' ? t('automation.create.connectorGit') : name}
                    </label>
                  ))}
                </div>
                <em>{t('automation.create.connectorHint')}</em>
              </div>
            )}
          </AdvancedSection>

        </div>
        <div className="sheet-foot design-sheet-foot">
          <button className="button ghost" onClick={onClose} disabled={sending}>
            {t('automation.create.cancel')}
          </button>
          <div className="spacer" />
          <button
            className="button primary"
            onClick={submit}
            disabled={sending}
          >
            {sending ? (
              <>
                <Icon name="spinner" />
                {t('automation.create.creating')}
              </>
            ) : (
              <>
                <Icon name="plus" />
                {t('automation.create.submit')}
                <kbd>⌘↵</kbd>
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  )
}
