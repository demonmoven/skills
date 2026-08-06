import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import i18n from '../i18n'
import { api } from '../api/client'
import type { AdventurerFile, ExecutorInfo, QuestIntensity, QuestMeta, QuestType } from '../api/types'
import { agentKey, agentLabel, isRunnableAdventurer, isSelectableAgentExecutor, modelPolicyLabel } from './util'
import AdvancedSection from './AdvancedSection'
import CapabilityTierBadge, { capabilityTierAdvice, capabilityTierRank } from './CapabilityTierBadge'
import DesignSelect from './DesignSelect'
import Icon from './Icon'
import useDragToDismiss from '../hooks/useDragToDismiss'

type AdventurerClass = AdventurerFile['class']

type Props = {
  adventurers: AdventurerFile[]
  executors: ExecutorInfo[]
  onClose: () => void
  onError: (msg: string) => void
  onConfigureAgents: () => void
  onCreated: (quest: QuestMeta) => void
  onAdventurerCreated?: (adv: AdventurerFile) => void
}

type QuickDefaults = {
  name: string
  description: string
  tools: string
}

type QuickForm = {
  name: string
  agent: string
}

// v0.5: 三档 workflow mode — Direct / Checked / Goal
// - Direct: 低风险自动闭环；外部写自动升档至 Checked；不可逆/权限/业务责任才 Human confirm
// - Checked: Maker 执行 → Checker 验证；fail 自动返工
// - Goal: 先出方案文档，审查后可按方案执行（Plan only 子模式：只出方案不执行）
type QuestFlow = 'direct' | 'checked' | 'goal'

export default function CreateQuestSheet({
  adventurers,
  executors,
  onClose,
  onError,
  onConfigureAgents,
  onCreated,
  onAdventurerCreated,
}: Props) {
  const { t } = useTranslation()
  const QUICK_DEFAULTS: Record<AdventurerClass, QuickDefaults> = {
    warrior: {
      name: t('quest.create.quickSetup.warriorDefaultName'),
      description: t('quest.create.quickSetup.warriorDefaultDesc'),
      tools: 'bash, read, edit, write, glob, grep, web_fetch',
    },
    mage: {
      name: t('quest.create.quickSetup.mageDefaultName'),
      description: t('quest.create.quickSetup.mageDefaultDesc'),
      tools: 'read, glob, grep',
    },
  }
  const warriors = adventurers.filter((a) => a.class === 'warrior' && isRunnableAdventurer(a, executors))
  const mages = adventurers.filter((a) => a.class === 'mage' && isRunnableAdventurer(a, executors))
  const { sheetRef, handleRef, sheetStyle, scrimStyle } = useDragToDismiss({ onDismiss: onClose })
  const [query, setQuery] = useState('')
  const [flow, setFlow] = useState<QuestFlow>('checked')
  const [workDir, setWorkDir] = useState('')
  const [warriorID, setWarriorID] = useState('')
  const [mageID, setMageID] = useState('')
  const [executeAgentID, setExecuteAgentID] = useState('')
  const [reviewAgentID, setReviewAgentID] = useState('')
  // v0.5: 创建后立即启动 = 默认行为（高级选项可关）
  const [autoStart, setAutoStart] = useState(true)
  // v0.5: Goal 子模式 — Plan only（只出方案，方案过审后不自动执行）
  const [planOnly, setPlanOnly] = useState(false)
  const [enabledConnectors, setEnabledConnectors] = useState<string[]>([])
  const [selectedConnectors, setSelectedConnectors] = useState<string[]>([])
  const [busy, setBusy] = useState(false)
  const [queryErr, setQueryErr] = useState('')
  const [artifacts, setArtifacts] = useState<File[]>([])
  const [dragOver, setDragOver] = useState(false)
  const [creatingClass, setCreatingClass] = useState<AdventurerClass | null>(null)
  const [quickForms, setQuickForms] = useState<Record<AdventurerClass, QuickForm>>({
    warrior: { name: QUICK_DEFAULTS.warrior.name, agent: '' },
    mage: { name: QUICK_DEFAULTS.mage.name, agent: '' },
  })

  const enabledExecutors = useMemo(
    () => executors.filter(isSelectableAgentExecutor),
    [executors],
  )
  const hasActiveWarrior = warriors.length > 0
  const hasActiveMage = mages.length > 0

  // v0.5 agent-first: 去重后的可选 agent executor 列表
  const selectableAgentExecutors = useMemo(() => {
    const seen = new Set<string>()
    return enabledExecutors
      .map((executor) => ({ executor, agent: agentKey(executor) }))
      .filter(({ agent }) => {
        if (!agent || seen.has(agent)) return false
        seen.add(agent)
        return true
      })
      .sort((a, b) => {
        // official 优先，再按 name 字母序
        if (a.executor.official !== b.executor.official) {
          return (b.executor.official ? 1 : 0) - (a.executor.official ? 1 : 0)
        }
        return agentLabel(a.agent).localeCompare(agentLabel(b.agent))
      })
  }, [enabledExecutors])
  const hasSelectableAgent = selectableAgentExecutors.length > 0
  const agentOptions = [
    { value: '', label: t('quest.create.autoSelect') },
    ...selectableAgentExecutors.map(({ agent, executor }) => ({
      value: agent,
      label: agentLabel(agent) + (executor.official ? ' ★' : ''),
    })),
  ]

  // agent-first: 有 selectable agent 就不强制要求 adventurer
  const needsAdventurerSetup = (!hasActiveWarrior || !hasActiveMage) && !hasSelectableAgent
  const warriorOptions = [{ value: '', label: t('quest.create.autoSelect') }, ...warriors.map((a) => ({ value: a.id, label: a.name }))]
  const mageOptions = [{ value: '', label: t('quest.create.autoSelect') }, ...mages.map((a) => ({ value: a.id, label: a.name }))]
  const artifactTotalSize = artifacts.reduce((sum, file) => sum + file.size, 0)

  // v0.5: flow → type/intensity 映射
  // - Direct → execute + quick（低风险快速闭环）
  // - Checked → execute + standard（Maker + Checker 标准流程）
  // - Goal → design + standard（先方案后执行）
  const type: QuestType = flow === 'goal' ? 'design' : 'execute'
  const intensity: QuestIntensity = flow === 'direct' ? 'quick' : 'standard'
  const withDesignPhase = flow === 'goal'

  // v0.5: mode 参数映射到后端期望的 mode 字符串
  // 后端用 mode=run/check/design 区分
  const backendMode = flow === 'direct' ? 'run' : flow === 'checked' ? 'check' : 'design'

  // Direct 模式只有单阶段，不发送评审 agent
  const effectiveReviewAgentID = flow === 'direct' ? '' : reviewAgentID

  const addArtifacts = useCallback((files: FileList | File[]) => {
    const next = Array.from(files).filter(Boolean)
    if (next.length === 0) return
    setArtifacts((prev) => {
      const merged = [...prev]
      for (const file of next) {
        if (merged.length >= 8) break
        const duplicate = merged.some((item) => item.name === file.name && item.size === file.size && item.lastModified === file.lastModified)
        if (!duplicate) merged.push(file)
      }
      return merged
    })
  }, [])

  const removeArtifact = useCallback((index: number) => {
    setArtifacts((prev) => prev.filter((_, i) => i !== index))
  }, [])

  const updateQuickForm = (className: AdventurerClass, patch: Partial<QuickForm>) => {
    setQuickForms((prev) => ({
      ...prev,
      [className]: {
        ...prev[className],
        ...patch,
      },
    }))
  }

  const createQuickAdventurer = useCallback(async (className: AdventurerClass) => {
    const form = quickForms[className]
    if (!form.name.trim()) {
      onError(t('quest.create.error.nameEmpty'))
      return
    }
    if (!form.agent.trim()) {
      onError(t('quest.create.error.selectAgent'))
      return
    }
    setCreatingClass(className)
    try {
      const def = QUICK_DEFAULTS[className]
      const res = await api.post<{ item: AdventurerFile }>('/api/adventurers', {
        name: form.name.trim(),
        class: className,
        agent: form.agent.trim(),
        description: def.description,
        tools: def.tools.split(',').map((t) => t.trim()).filter(Boolean),
        level: 1,
        status: 'active',
      })
      const adv = res.item
      onAdventurerCreated?.(adv)
      if (adv.class === 'warrior') setWarriorID(adv.id)
      if (adv.class === 'mage') setMageID(adv.id)
    } catch (e) {
      onError(e instanceof Error ? e.message : t('quest.create.error.createAdventurerFail'))
    } finally {
      setCreatingClass(null)
    }
  }, [onAdventurerCreated, onError, quickForms])

  const trimmedWorkDir = workDir.trim()

  const submit = useCallback(async () => {
    if (!query.trim()) {
      setQueryErr(t('quest.create.error.queryEmpty'))
      return
    }
    if (needsAdventurerSetup) {
      onError(t('quest.create.error.needAdventurer'))
      return
    }
    if (hasSelectableAgent && !executeAgentID.trim()) {
      onError(t('quest.create.error.needExecuteAgent'))
      return
    }
    setQueryErr('')
    setBusy(true)
    try {
      let res: { quest: QuestMeta }
      if (artifacts.length > 0) {
        const body = new FormData()
        body.set('query', query)
        body.set('mode', backendMode)
        body.set('type', type)
        body.set('intensity', intensity)
        body.set('work_dir', trimmedWorkDir)
        body.set('warrior_id', warriorID)
        body.set('mage_id', mageID)
        if (executeAgentID) body.set('execute_agent_id', executeAgentID)
        if (effectiveReviewAgentID) body.set('review_agent_id', effectiveReviewAgentID)
        body.set('auto_start', String(autoStart))
        body.set('with_design_phase', String(withDesignPhase))
        body.set('plan_only', String(flow === 'goal' && planOnly))
        // v0.5: 不再发 allow_quick_auto_complete — Direct 低风险自动闭环是默认行为
        // v0.5: 不再发 auto_spawn_execute — Goal 子模式用 plan_only 控制
        selectedConnectors.forEach((c) => body.append('connectors', c))
        artifacts.forEach((file) => body.append('artifacts', file, file.name))
        res = await api.postForm<{ quest: QuestMeta }>('/api/quests', body)
      } else {
        res = await api.post<{ quest: QuestMeta }>('/api/quests', {
          query,
          mode: backendMode,
          type,
          intensity,
          work_dir: trimmedWorkDir,
          warrior_id: warriorID,
          mage_id: mageID,
          execute_agent_id: executeAgentID || undefined,
          review_agent_id: effectiveReviewAgentID || undefined,
          auto_start: autoStart,
          with_design_phase: withDesignPhase,
          plan_only: flow === 'goal' && planOnly,
          connectors: selectedConnectors,
        })
      }
      onCreated(res.quest)
      onClose()
    } catch (e) {
      onError(e instanceof Error ? e.message : t('quest.create.error.createQuestFail'))
    } finally {
      setBusy(false)
    }
  }, [
    query,
    backendMode,
    intensity,
    workDir,
    warriorID,
    mageID,
    executeAgentID,
    reviewAgentID,
    effectiveReviewAgentID,
    hasSelectableAgent,
    autoStart,
    withDesignPhase,
    flow,
    planOnly,
    selectedConnectors,
    needsAdventurerSetup,
    artifacts,
    onCreated,
    onClose,
    onError,
    type,
    trimmedWorkDir,
  ])

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      }
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
        e.preventDefault()
        submit()
      }
    }
    window.addEventListener('keydown', handleKey)
    return () => window.removeEventListener('keydown', handleKey)
  }, [onClose, submit])

  useEffect(() => {
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.body.style.overflow = prev
    }
  }, [])

  // 加载可用 connectors + 默认冒险者 + 默认 agent
  useEffect(() => {
    let cancelled = false
    api.get<{ config?: { connectors?: { git?: { enabled?: boolean } }; default_warrior_id?: string; default_mage_id?: string; default_execute_agent_id?: string; default_review_agent_id?: string } }>('/api/settings')
      .then((res) => {
        if (cancelled) return
        const list: string[] = []
        if (res.config?.connectors?.git?.enabled) list.push('git')
        setEnabledConnectors(list)
        if (res.config?.default_warrior_id) setWarriorID(res.config.default_warrior_id)
        if (res.config?.default_mage_id) setMageID(res.config.default_mage_id)
        // agent-first: 从 settings 取默认 agent
        const cfgExecAgent = res.config?.default_execute_agent_id || ''
        const cfgRevAgent = res.config?.default_review_agent_id || ''
        const firstAgent = selectableAgentExecutors[0]?.agent || ''
        setExecuteAgentID(cfgExecAgent || firstAgent)
        setReviewAgentID(cfgRevAgent || (cfgExecAgent ? '' : firstAgent))
      })
      .catch(() => { /* 不阻塞创建 */ })
    return () => { cancelled = true }
  }, [selectableAgentExecutors])

  // executors 加载后，如果 agent 还没选，自动填第一个可选的
  useEffect(() => {
    if (executeAgentID === '' && hasSelectableAgent) {
      setExecuteAgentID(selectableAgentExecutors[0].agent)
    }
    if (reviewAgentID === '' && hasSelectableAgent && flow !== 'direct') {
      // Direct 模式只有单阶段，不需要评审 agent
      setReviewAgentID(selectableAgentExecutors[0].agent)
    }
  }, [hasSelectableAgent, selectableAgentExecutors, executeAgentID, reviewAgentID, flow])

  return (
    <div className="sheet-scrim" style={scrimStyle} onClick={onClose} role="dialog" aria-modal="true" aria-label={t('aria.questCreate')}>
      <div className="sheet-card design-sheet quest-create-sheet" ref={sheetRef} style={sheetStyle} onClick={(e) => e.stopPropagation()}>
        <i className="sheet-handle" ref={handleRef} aria-hidden="true" />
        <header className="sheet-head design-sheet-head">
          <div className="sheet-title-row">
            <span className="sheet-icon">
              <Icon name="send" />
            </span>
            <div>
              <div className="sheet-title-line">
                <h2>{t('quest.create.title')}</h2>
                <span className="sheet-badge">{t('quest.create.badge')}</span>
              </div>
              <p className="sheet-subtitle">{t('quest.create.subtitle')}</p>
            </div>
          </div>
          <button className="icon-button" onClick={onClose} aria-label={t('aria.close')}>
            <Icon name="x" />
          </button>
        </header>

        <div className="sheet-body design-sheet-body">
          <label className="field">
            <span className="field-label-row">
              <span>{t('quest.create.field.query')}</span>
              <em>{t('quest.create.field.queryHint')}</em>
            </span>
            <textarea
              value={query}
              onChange={(e) => {
                setQuery(e.target.value)
                if (queryErr) setQueryErr('')
              }}
              onKeyDown={(e) => {
                if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                  e.preventDefault()
                  submit()
                }
              }}
              onPaste={(e) => {
                const files = Array.from(e.clipboardData.files)
                if (files.length > 0) {
                  e.preventDefault()
                  addArtifacts(files)
                }
              }}
              placeholder={t('quest.create.field.queryPlaceholder')}
              rows={6}
              autoFocus
            />
            {queryErr && <small className="field-error">{queryErr}</small>}
          </label>

          {/* v0.5: 主界面只留三档 — Direct / Checked / Goal */}
          <div className="form-grid">
            <div className="field flow-field">
              <span>{t('quest.create.field.flow')}</span>
              <div className="option-cards flow-options">
                <button
                  type="button"
                  className={'option-card ' + (flow === 'direct' ? 'selected' : '')}
                  onClick={() => setFlow('direct')}
                >
                  <Icon name="bolt" />
                  <div>
                    <strong>Direct</strong>
                    <em>{t('quest.create.flow.direct.desc')}</em>
                  </div>
                </button>
                <button
                  type="button"
                  className={'option-card ' + (flow === 'checked' ? 'selected' : '')}
                  onClick={() => setFlow('checked')}
                >
                  <Icon name="sword" />
                  <div>
                    <strong>Checked</strong>
                    <em>{t('quest.create.flow.checked.desc')}</em>
                  </div>
                </button>
                <button
                  type="button"
                  className={'option-card ' + (flow === 'goal' ? 'selected' : '')}
                  onClick={() => setFlow('goal')}
                >
                  <Icon name="clipboard-check" />
                  <div>
                    <strong>Goal</strong>
                    <em>{t('quest.create.flow.goal.desc')}</em>
                  </div>
                </button>
              </div>
            </div>
          </div>

          {/* Goal 子模式说明 */}
          {flow === 'goal' && (
            <div className="form-banner info">
              <Icon name="info" />
              <span>{t('quest.create.banner.goal')}</span>
            </div>
          )}

          {/* Direct 模式说明 */}
          {flow === 'direct' && (
            <div className="form-banner info">
              <Icon name="info" />
              <span>{t('quest.create.banner.direct')}</span>
            </div>
          )}

          {/* v0.5 agent-first: Agent 选择器为主控件 */}
          <div className="form-grid agents-grid">
            <label className="field choice-field">
              <span className="field-label-row">
                <span>{t('quest.create.field.executeAgent')}</span>
                <em>{t('quest.create.field.executeAgentHint')}</em>
              </span>
              <DesignSelect
                value={executeAgentID}
                options={agentOptions}
                onChange={setExecuteAgentID}
                ariaLabel={t('quest.create.field.executeAgent')}
              />
            </label>
            {flow !== 'direct' && (
              <label className="field choice-field">
                <span className="field-label-row">
                  <span>{t('quest.create.field.reviewAgent')}</span>
                  <em>{t('quest.create.field.reviewAgentHint')}</em>
                </span>
                <DesignSelect
                  value={reviewAgentID}
                  options={agentOptions}
                  onChange={setReviewAgentID}
                  ariaLabel={t('quest.create.field.reviewAgent')}
                />
              </label>
            )}
          </div>
          {!hasSelectableAgent && (
            <div className="form-banner error">
              <Icon name="alert" />
              <span>{t('quest.create.quickSetup.noAgentError')}</span>
              <button type="button" className="button tiny" onClick={onConfigureAgents}>
                <Icon name="settings" />
                {t('quest.create.quickSetup.goEnableAgent')}
              </button>
            </div>
          )}

          <AdvancedSection hint={t('quest.create.advanced.workspace')}>
            <section className="quest-workspace-panel">
              <div className="quest-workspace-head">
                <Icon name="layers" />
                <div>
                  <strong>{t('quest.create.advanced.workspaceTitle')}</strong>
                  <span>{t('quest.create.advanced.workspaceHint')}</span>
                </div>
              </div>
              <div className="quest-workspace-grid">
                <label className="field quest-workdir-field">
                  <span>{t('quest.create.field.workDir')}</span>
                  <input
                    className="mono"
                    value={workDir}
                    onChange={(e) => setWorkDir(e.target.value)}
                    placeholder={t('quest.create.field.workDirPlaceholder')}
                  />
                </label>
              </div>
            </section>

            {/* Persona 冒险者（可选 overlay） */}
            <section className="quest-workspace-panel">
              <div className="quest-workspace-head">
                <Icon name="user" />
                <div>
                  <strong>{t('quest.create.advanced.personaTitle')}</strong>
                  <span>{t('quest.create.advanced.personaHint')}</span>
                </div>
              </div>
              <div className="form-grid agents-grid">
                <label className="field choice-field">
                  <span>{t('quest.create.field.warrior')}</span>
                  <DesignSelect
                    value={warriorID}
                    options={warriorOptions}
                    onChange={setWarriorID}
                    ariaLabel={t('aria.warrior')}
                  />
                </label>
                {flow !== 'direct' && (
                  <label className="field choice-field">
                    <span>{t('quest.create.field.mage')}</span>
                    <DesignSelect
                      value={mageID}
                      options={mageOptions}
                      onChange={setMageID}
                      ariaLabel={t('aria.mage')}
                    />
                  </label>
                )}
              </div>
            </section>

            {/* v0.5: 创建后立即启动 = 默认行为，保留为高级选项可关闭 */}
            <label className="check-row launch-row">
              <input type="checkbox" checked={autoStart} onChange={(e) => setAutoStart(e.target.checked)} />
              <span>{t('quest.create.field.autoStart')}</span>
              <em>{t('quest.create.autoStartHint')}</em>
            </label>

            {/* v0.5: Goal 子模式 — Plan only（只出方案） */}
            {flow === 'goal' && (
              <label className="check-row launch-row">
                <input
                  type="checkbox"
                  checked={planOnly}
                  onChange={(e) => setPlanOnly(e.target.checked)}
                />
                <span>{t('quest.create.field.planOnly')}</span>
                <em>{t('quest.create.planOnlyHint')}</em>
              </label>
            )}

            {enabledConnectors.length > 0 && (
              <div className="check-row launch-row connectors-row">
                <span className="connectors-label">{t('quest.create.field.connectors')}</span>
                <div className="connectors-options">
                  {enabledConnectors.map((name) => (
                    <label key={name} className="connector-chip">
                      <input
                        type="checkbox"
                        checked={selectedConnectors.includes(name)}
                        onChange={(e) => {
                          setSelectedConnectors((prev) =>
                            e.target.checked ? [...prev, name] : prev.filter((c) => c !== name)
                          )
                        }}
                      />
                      <Icon name="git-merge" size={13} />
                      {name === 'git' ? t('quest.create.connectorGit') : name}
                    </label>
                  ))}
                </div>
                <em>{t('quest.create.connectorsHint')}</em>
              </div>
            )}
          </AdvancedSection>

          <section
            className={'quest-artifact-dropzone' + (dragOver ? ' drag-active' : '')}
            onDragOver={(e) => {
              e.preventDefault()
              setDragOver(true)
            }}
            onDragLeave={() => setDragOver(false)}
            onDrop={(e) => {
              e.preventDefault()
              setDragOver(false)
              addArtifacts(e.dataTransfer.files)
            }}
          >
            <div className="quest-artifact-dropzone-head">
              <Icon name="upload" />
              <div>
                <strong>{t('quest.create.field.artifact')}</strong>
                <span>{t('quest.create.field.artifactHint')}</span>
              </div>
              <label className="button compact">
                <input
                  type="file"
                  multiple
                  onChange={(e) => {
                    if (e.target.files) addArtifacts(e.target.files)
                  }}
                  style={{ display: 'none' }}
                />
                <Icon name="plus" size={12} />
                {t('quest.create.field.addFile')}
              </label>
            </div>
            {artifacts.length > 0 && (
              <div className="quest-artifact-list">
                {artifacts.map((file, i) => (
                  <div key={file.name + i} className="quest-artifact-item">
                    <Icon name="file-text" size={14} />
                    <span className="quest-artifact-name" title={file.name}>{file.name}</span>
                    <span className="mono quest-artifact-size">{formatArtifactSize(file.size)}</span>
                    <button
                      type="button"
                      className="icon-button tiny"
                      onClick={() => removeArtifact(i)}
                      aria-label={t('aria.removeArtifact')}
                    >
                      <Icon name="x" size={12} />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </section>

          {needsAdventurerSetup && (
            <div className="quest-setup-guide">
              <div className="form-banner error">
                <Icon name="alert" />
                <span>{t('quest.create.setupMissing')}</span>
                <button type="button" className="button tiny" onClick={onConfigureAgents}>
                  <Icon name="settings" />
                  {t('quest.create.quickSetup.goEnableAgent')}
                </button>
              </div>
            </div>
          )}
          {hasSelectableAgent && (!hasActiveWarrior || !hasActiveMage) && (
            <div className="quest-setup-guide">
              <div className="form-banner info">
                <Icon name="info" />
                <span>{t('quest.create.advanced.personaHint')}</span>
              </div>
              {!hasActiveWarrior && (
                <QuickAdventurerSetup
                  className="warrior"
                  form={quickForms.warrior}
                  busy={creatingClass === 'warrior'}
                  executors={enabledExecutors}
                  onConfigureAgents={onConfigureAgents}
                  onChange={(patch) => updateQuickForm('warrior', patch)}
                  onCreate={() => createQuickAdventurer('warrior')}
                />
              )}
              {!hasActiveMage && flow !== 'direct' && (
                <QuickAdventurerSetup
                  className="mage"
                  form={quickForms.mage}
                  busy={creatingClass === 'mage'}
                  executors={enabledExecutors}
                  onConfigureAgents={onConfigureAgents}
                  onChange={(patch) => updateQuickForm('mage', patch)}
                  onCreate={() => createQuickAdventurer('mage')}
                />
              )}
            </div>
          )}

        </div>

        <footer className="sheet-foot sheet-foot-primary design-sheet-foot">
          <div className="sheet-shortcut-hint">
            <kbd>⌘↵</kbd>
            <span>{t('quest.create.shortcutHint')}</span>
          </div>
          <button
            className="button primary submit-button"
            disabled={busy || needsAdventurerSetup}
            onClick={submit}
          >
            {busy ? (
              <>
                <Icon name="spinner" className="spin" />
                {t('quest.create.submitting')}
              </>
            ) : (
              <>
                <Icon name="send" size={14} />
                {t('quest.create.title')}
                <kbd>⌘↵</kbd>
              </>
            )}
          </button>
        </footer>
      </div>
    </div>
  )
}


function formatArtifactSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function QuickAdventurerSetup({
  className,
  form,
  busy,
  executors,
  onConfigureAgents,
  onChange,
  onCreate,
}: {
  className: AdventurerClass
  form: QuickForm
  busy: boolean
  executors: ExecutorInfo[]
  onConfigureAgents: () => void
  onChange: (patch: Partial<QuickForm>) => void
  onCreate: () => void
}) {
  const { t } = useTranslation()
  const isWarrior = className === 'warrior'
  const executorOptions = useMemo(() => {
    const seen = new Set<string>()
    return executors
      .map((executor) => ({ executor, agent: agentKey(executor) }))
      .filter(({ agent }) => {
        if (!agent || seen.has(agent)) return false
        seen.add(agent)
        return true
      })
      .sort((a, b) => capabilityTierRank(a.executor.capability_tier) - capabilityTierRank(b.executor.capability_tier) || agentLabel(a.agent).localeCompare(agentLabel(b.agent)))
  }, [executors])
  const selectedExecutor = executorOptions.find(({ agent }) => agent === form.agent)?.executor || null
  return (
    <section className="quick-adventurer-card">
      <div className="quick-adventurer-title">
        <Icon name={isWarrior ? 'sword' : 'wand'} />
        <div>
          <strong>{isWarrior ? t('quest.create.quickSetup.warriorTitle') : t('quest.create.quickSetup.mageTitle')}</strong>
          <span>{isWarrior ? t('quest.create.quickSetup.warriorRole') : t('quest.create.quickSetup.mageRole')}</span>
        </div>
      </div>
      <div className="grid-2">
        <label className="field">
          <span>{t('quest.create.field.adventurerName')}</span>
          <input value={form.name} onChange={(e) => onChange({ name: e.target.value })} />
        </label>
        <label className="field">
          <span>{t('term.agent')}</span>
          <DesignSelect
            value={form.agent}
            options={[
              { value: '', label: t('quest.create.quickSetup.selectAgentPlaceholder') },
              ...executorOptions.map(({ agent }) => ({ value: agent, label: agentLabel(agent) })),
            ]}
            onChange={(agent) => onChange({ agent })}
            ariaLabel={t('aria.agent')}
          />
        </label>
        <label className="field">
          <span>{t('quest.create.field.modelPolicy')}</span>
          <input value={modelPolicyLabel(selectedExecutor)} readOnly />
        </label>
        <label className="field">
          <span>{t('quest.create.field.capabilityTier')}</span>
          <div className="capability-field">
            <CapabilityTierBadge tier={selectedExecutor?.capability_tier} />
          </div>
          <small className="field-hint">{capabilityTierAdvice(selectedExecutor?.capability_tier)}</small>
        </label>
      </div>
      {executors.length === 0 && (
        <div className="form-banner error">
          <Icon name="alert" />
          <span>{t('quest.create.quickSetup.noAgentError')}</span>
          <button type="button" className="button tiny" onClick={onConfigureAgents}>
            <Icon name="settings" />
            {t('quest.create.quickSetup.goEnableAgent')}
          </button>
        </div>
      )}
      <button
        className="button secondary quick-adventurer-action"
        type="button"
        disabled={busy || !form.name.trim() || !form.agent.trim()}
        onClick={onCreate}
      >
        {busy ? (
          <>
            <Icon name="spinner" className="spin" />
            {t('quest.create.quickSetup.creating')}
          </>
        ) : (
          <>
            <Icon name="plus" />
            {isWarrior ? t('quest.create.quickSetup.warriorTitle') : t('quest.create.quickSetup.mageTitle')}
          </>
        )}
      </button>
    </section>
  )
}
