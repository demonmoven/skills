import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api/client'
import type {
  AdventurerFile,
  AllowedCommand,
  CommandRecommendation,
  ConfigPackagePreview,
  WorkspaceCleanupItem,
} from '../api/types'
import AgentSettingsSection, { agentToForm, type AgentForm } from '../components/AgentSettingsSection'
import { formatSize } from '../components/ContextSettingsSection'
import DesignSelect from '../components/DesignSelect'
import Icon from '../components/Icon'
import {
  buildSettingsPatch,
  envFromText,
  listFromText,
  parseAllowlistCount,
  parseMageCommandAllowlist,
  type GlobalCfg,
} from '../features/settings/settingsModel'
import { THEMES, applyTheme, getStoredTheme, type ThemeKey } from '../theme'
import useDragToDismiss from '../hooks/useDragToDismiss'

type Props = {
  adventurers?: AdventurerFile[]
  onError: (msg: string) => void
  onSuccess: (msg: string) => void
  onChanged: () => void
}

export default function Settings({ adventurers, onError, onSuccess, onChanged }: Props) {
  const { t } = useTranslation()
  const [cfg, setCfg] = useState<GlobalCfg>({})
  const [agents, setAgents] = useState<any[]>([])
  const [agentDrafts, setAgentDrafts] = useState<Record<string, AgentForm>>({})
  const [newAgentKeys, setNewAgentKeys] = useState<string[]>([])
  const [expandedAgentKeys, setExpandedAgentKeys] = useState<Record<string, boolean>>({})
  const [agentSaving, setAgentSaving] = useState<string | null>(null)
  const [agentDeleting, setAgentDeleting] = useState<string | null>(null)
  const [agentDeleteConfirm, setAgentDeleteConfirm] = useState<{ key: string; name: string } | null>(null)
  const { sheetRef: deleteAgentSheetRef, sheetStyle: deleteAgentSheetStyle, scrimStyle: deleteAgentScrimStyle } =
    useDragToDismiss({ onDismiss: () => setAgentDeleteConfirm(null), disabled: !agentDeleteConfirm })
  const [saving, setSaving] = useState(false)
  const [loading, setLoading] = useState(true)
  const [theme, setTheme] = useState<ThemeKey>(getStoredTheme())
  const [cleanupRunning, setCleanupRunning] = useState(false)
  const [cleanupDryRun, setCleanupDryRun] = useState(true)
  const [cleanupIncludeFailed, setCleanupIncludeFailed] = useState(false)
  const [cleanupItems, setCleanupItems] = useState<WorkspaceCleanupItem[]>([])
  const [cleanupAttempted, setCleanupAttempted] = useState(false)
  const [configImportOpen, setConfigImportOpen] = useState(false)
  const [configImportFile, setConfigImportFile] = useState<File | null>(null)
  const [configImportData, setConfigImportData] = useState<any>(null)
  const [configImportPreview, setConfigImportPreview] = useState<ConfigPackagePreview | null>(null)
  const [configImporting, setConfigImporting] = useState(false)
  const [configExporting, setConfigExporting] = useState(false)
  const configFileInputRef = useRef<HTMLInputElement>(null)
  const [commandAllowlistText, setCommandAllowlistText] = useState('[]')
  const [commandRecommendations, setCommandRecommendations] = useState<CommandRecommendation[]>([])
  const [commandRecommendDir, setCommandRecommendDir] = useState('')
  const [commandRecommendLoading, setCommandRecommendLoading] = useState(false)
  const enabledAgentCount = agents.filter((agent) => agent.enabled !== false).length
  const allowlistCount = parseAllowlistCount(commandAllowlistText)
  const defaultWorkingDir = cfg.default_working_dir?.trim() || t('settings.workspace.dirPlaceholder')
  const settingsHealthItems = [
    {
      label: t('settings.health.agentLabel'),
      value: loading ? t('settings.health.agentLoading') : t('settings.health.agentRatio', { enabled: enabledAgentCount, total: agents.length }),
      detail: agents.length > 0 ? t('settings.health.agentDetail') : t('settings.health.agentEmpty'),
      icon: 'users',
      tone: agents.length > 0 && enabledAgentCount === 0 ? 'warning' : 'default',
    },
    {
      label: t('settings.health.concurrency'),
      value: String(cfg.max_concurrent ?? 6),
      detail: t('settings.health.concurrencyDetail'),
      icon: 'layers',
      tone: 'default',
    },
    {
      label: t('settings.health.workspace'),
      value: defaultWorkingDir,
      detail: t('settings.health.workspaceDetail'),
      icon: 'file-code',
      tone: 'default',
    },
    {
      label: t('settings.health.command'),
      value: allowlistCount == null ? t('settings.health.commandError') : String(allowlistCount),
      detail: allowlistCount == null ? t('settings.health.commandJsonFix') : t('settings.health.commandDetail'),
      icon: allowlistCount == null ? 'triangle-alert' : 'shield',
      tone: allowlistCount == null ? 'danger' : 'default',
    },
  ]

  function handleThemeChange(key: ThemeKey) {
    setTheme(key)
    applyTheme(key)
  }

  const fetchCfg = useCallback(async () => {
    setLoading(true)
    try {
      const res = await api.get<{
        config: GlobalCfg
        agents: any[]
      }>('/api/settings')
      const nextCfg = res.config || ({} as GlobalCfg)
      setCfg(nextCfg)
      setCommandAllowlistText(JSON.stringify(nextCfg.mage_command_allowlist || [], null, 2))
      const nextAgents = res.agents || []
      setAgents(nextAgents)
      setNewAgentKeys([])
      setExpandedAgentKeys((prev) => {
        const next: Record<string, boolean> = {}
        for (const p of nextAgents) {
          const key = p.name || p.agent || p.id
          if (key && prev[key]) next[key] = true
        }
        return next
      })
      setAgentDrafts(Object.fromEntries(nextAgents.map((p: any) => [p.name || p.agent || p.id, agentToForm(p)])))
    } catch (e) {
      onError(e instanceof Error ? e.message : t('settings.error.loadFail'))
    } finally {
      setLoading(false)
    }
  }, [onError])

  useEffect(() => {
    fetchCfg()
  }, [fetchCfg])

  const runCleanup = useCallback(async (dryRun: boolean) => {
    setCleanupRunning(true)
    setCleanupDryRun(dryRun)
    try {
      const res = await api.post<{
        dry_run: boolean
        retention_days: number
        items: WorkspaceCleanupItem[]
      }>('/api/quests/cleanup', {
        retention_days: cfg.workspace_retention_days ?? 7,
        include_failed: cleanupIncludeFailed,
        dry_run: dryRun,
      })
      setCleanupItems(res.items || [])
      setCleanupAttempted(true)
      onChanged()
    } catch (e) {
      onError(e instanceof Error ? e.message : t('settings.error.cleanupFail'))
    } finally {
      setCleanupRunning(false)
    }
  }, [cfg.workspace_retention_days, cleanupIncludeFailed, onChanged, onError])

  const exportConfigPackage = useCallback(async () => {
    setConfigExporting(true)
    try {
      const download = await api.download('/api/settings/package/export')
      const a = document.createElement('a')
      a.href = download.url
      a.download = download.filename || 'gloop_config.json'
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      setTimeout(() => URL.revokeObjectURL(download.url), 1000)
    } catch (e) {
      onError(e instanceof Error ? e.message : t('settings.error.exportFail'))
    } finally {
      setConfigExporting(false)
    }
  }, [onError])

  const openConfigImport = useCallback(() => {
    setConfigImportOpen(true)
    setConfigImportFile(null)
    setConfigImportData(null)
    setConfigImportPreview(null)
    setTimeout(() => configFileInputRef.current?.click(), 50)
  }, [])

  const handleConfigFileChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0]
      if (!file) return
      setConfigImportFile(file)
      setConfigImportPreview(null)
      const reader = new FileReader()
      reader.onload = async () => {
        try {
          const parsed = JSON.parse(String(reader.result || '{}'))
          setConfigImportData(parsed)
          const res = await api.post<{ preview: ConfigPackagePreview }>(
            '/api/settings/package/preview',
            parsed,
          )
          setConfigImportPreview(res.preview)
        } catch (err) {
          setConfigImportData(null)
          setConfigImportPreview(null)
          onError(err instanceof Error ? err.message : t('settings.error.parseFail'))
        }
      }
      reader.onerror = () => onError(t('settings.error.readFail'))
      reader.readAsText(file)
      e.target.value = ''
    },
    [onError],
  )

  const cancelConfigImport = useCallback(() => {
    setConfigImportOpen(false)
    setConfigImportFile(null)
    setConfigImportData(null)
    setConfigImportPreview(null)
  }, [])

  const confirmConfigImport = useCallback(async () => {
    if (!configImportData) return
    setConfigImporting(true)
    try {
      await api.post('/api/settings/package/import', configImportData)
      cancelConfigImport()
      fetchCfg()
      onChanged()
    } catch (e) {
      onError(e instanceof Error ? e.message : t('settings.error.importFail'))
    } finally {
      setConfigImporting(false)
    }
  }, [configImportData, cancelConfigImport, fetchCfg, onChanged, onError])

  function patch<K extends keyof GlobalCfg>(k: K, v: GlobalCfg[K]) {
    setCfg((prev) => ({ ...prev, [k]: v }))
  }

  async function toggleExecutor(id: string, enabled: boolean) {
    try {
      await api.post(
        '/api/executors/' + id + '/' + (enabled ? 'enable' : 'disable'),
      )
      await fetchCfg()
      onChanged()
    } catch (e) {
      onError(e instanceof Error ? e.message : t('settings.error.toggleFail'))
    }
  }

  function patchAgent(key: string, patch: Partial<AgentForm>) {
    setAgentDrafts((prev) => ({
      ...prev,
      [key]: { ...(prev[key] || agentToForm({ name: key })), ...patch },
    }))
  }

  function addAgentDraft() {
    const key = 'new_agent_' + Date.now()
    setNewAgentKeys((prev) => [key, ...prev])
    setAgentDrafts((prev) => ({
      ...prev,
      [key]: agentToForm({ name: '', type: 'cli', enabled: true }),
    }))
    setExpandedAgentKeys((prev) => ({ ...prev, [key]: true }))
  }

  function toggleAgentExpanded(key: string) {
    setExpandedAgentKeys((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  function removeAgentDraft(key: string) {
    setNewAgentKeys((prev) => prev.filter((item) => item !== key))
    setExpandedAgentKeys((prev) => {
      const next = { ...prev }
      delete next[key]
      return next
    })
  }

  async function saveAgent(key: string) {
    const draft = agentDrafts[key]
    if (!draft) return
    if (!draft.name.trim()) {
      onError(t('settings.error.agentNameEmpty'))
      return
    }
    if (draft.type === 'cli' && draft.command.trim() === 'traex') {
      onError(t('settings.error.traexDisabled'))
      return
    }
    setAgentSaving(key)
    try {
      await api.post('/api/executors', {
        id: draft.id || 'exe_' + draft.name.trim(),
        name: draft.name.trim(),
        type: draft.type,
        command: draft.command.trim(),
        args: listFromText(draft.argsText),
        env: envFromText(draft.envText),
        default_model: draft.defaultModel.trim(),
        enabled: draft.enabled,
      })
      await fetchCfg()
      setNewAgentKeys((prev) => prev.filter((item) => item !== key))
      onChanged()
    } catch (e) {
      onError(e instanceof Error ? e.message : t('settings.error.saveAgentFail'))
    } finally {
      setAgentSaving(null)
    }
  }

  async function deleteAgent(key: string) {
    const draft = agentDrafts[key]
    const name = (draft?.name || key).trim()
    if (!name) return
    setAgentDeleting(key)
    try {
      await api.delete('/api/executors/' + encodeURIComponent(name))
      setAgentDeleteConfirm(null)
      await fetchCfg()
      onChanged()
    } catch (e) {
      onError(e instanceof Error ? e.message : t('settings.error.deleteAgentFail'))
    } finally {
      setAgentDeleting(null)
    }
  }

  async function save(e?: React.FormEvent) {
    e?.preventDefault()
    setSaving(true)
    try {
      let mageCommandAllowlist: AllowedCommand[]
      try {
        mageCommandAllowlist = parseMageCommandAllowlist(commandAllowlistText)
      } catch (err) {
        throw new Error(err instanceof Error ? err.message : t('settings.error.commandJsonFail'))
      }
      const body = buildSettingsPatch(cfg, mageCommandAllowlist)
      await api.patch('/api/settings', body)
      onChanged()
      onSuccess(t('settings.success.saved'))
    } catch (e) {
      onError(e instanceof Error ? e.message : t('settings.error.saveFail'))
    } finally {
      setSaving(false)
    }
  }

  async function loadCommandRecommendations() {
    setCommandRecommendLoading(true)
    try {
      const dir = commandRecommendDir.trim() || cfg.default_working_dir || ''
      const path = '/api/settings/mage-command-recommendations' + (dir ? '?project_dir=' + encodeURIComponent(dir) : '')
      const res = await api.get<{ project_dir: string; items: CommandRecommendation[] }>(path)
      setCommandRecommendDir(res.project_dir || dir)
      setCommandRecommendations(res.items || [])
    } catch (e) {
      onError(e instanceof Error ? e.message : t('settings.error.recommendFail'))
    } finally {
      setCommandRecommendLoading(false)
    }
  }

  function addRecommendedCommand(item: CommandRecommendation) {
    try {
      const current = JSON.parse(commandAllowlistText || '[]') as AllowedCommand[]
      if (!Array.isArray(current)) throw new Error(t('settings.error.commandJsonFail'))
      if (current.some((cmd) => cmd.id === item.id)) return
      const { reason: _reason, source: _source, present: _present, ...cmd } = item
      if (!cmd.side_effect_level) cmd.side_effect_level = 'L1'
      setCommandAllowlistText(JSON.stringify([...current, cmd], null, 2))
      setCommandRecommendations((prev) => prev.map((rec) => rec.id === item.id ? { ...rec, present: true } : rec))
    } catch (e) {
      onError(e instanceof Error ? e.message : t('settings.error.addRecommendFail'))
    }
  }

  return (
    <div className="settings-surface">
      <form id="settings-form" onSubmit={save} className="settings-form">
        <section className="settings-overview" aria-label={t('settings.overview.aria')}>
          <div className="settings-overview-copy">
            <span className="settings-overview-kicker">
              <Icon name={saving ? 'spinner' : loading ? 'spinner' : 'settings'} className={saving || loading ? 'spin' : ''} />
              {saving ? t('settings.overview.kickerSaving') : loading ? t('settings.overview.kickerLoading') : t('settings.overview.kickerReady')}
            </span>
            <strong>{t('settings.overview.title')}</strong>
            <span>{t('settings.overview.desc')}</span>
          </div>
          <div className="settings-health-grid">
            {settingsHealthItems.map((item) => (
              <div className={'settings-health-card tone-' + item.tone} key={item.label}>
                <Icon name={item.icon} />
                <span>{item.label}</span>
                <strong className="mono">{item.value}</strong>
                <em title={item.detail}>{item.detail}</em>
              </div>
            ))}
          </div>
          <nav className="settings-jump-row" aria-label={t('settings.overview.jumpAria')}>
            <a href="#settings-agents">Agent</a>
            <a href="#settings-limits">{t('settings.overview.jumpBudget')}</a>
            <a href="#settings-workspace">{t('settings.overview.jumpWorkspace')}</a>
            <a href="#settings-connectors">Connectors</a>
            <a href="#settings-package">{t('settings.overview.jumpPackage')}</a>
          </nav>
        </section>

        <section className="settings-section" id="settings-theme">
          <h2>{t('settings.theme.title')}</h2>
          <div className="theme-grid">
            {THEMES.map((t) => (
              <button
                key={t.key}
                type="button"
                className={'theme-card' + (theme === t.key ? ' active' : '')}
                onClick={() => handleThemeChange(t.key)}
              >
                <span
                  className="theme-swatch"
                  style={{ background: t.swatch }}
                />
                <span className="theme-check">✓</span>
                <span className="theme-name">{t.label}</span>
                <span className="theme-desc">{t.desc}</span>
              </button>
            ))}
          </div>
        </section>

        <div id="settings-agents" className="settings-anchor-target">
          <AgentSettingsSection
            loading={loading}
            agents={agents}
            newAgentKeys={newAgentKeys}
            agentDrafts={agentDrafts}
            expandedAgentKeys={expandedAgentKeys}
            agentSaving={agentSaving}
            agentDeleting={agentDeleting}
            onAddAgent={addAgentDraft}
            onToggleExecutor={toggleExecutor}
            onToggleExpanded={toggleAgentExpanded}
            onRemoveDraft={removeAgentDraft}
            onPatchAgent={patchAgent}
            onSaveAgent={saveAgent}
            onRequestDelete={(key, name) => setAgentDeleteConfirm({ key, name })}
          />
        </div>

        <section className="settings-section" id="settings-limits">
          <h2>{t('settings.budget.title')}</h2>
          <p className="settings-section-hint" dangerouslySetInnerHTML={{ __html: t('settings.budget.hint') }} />
          <div className="settings-box">
            <div className="form-grid">
              <label className="field inline-field">
                <span>{t('settings.budget.maxDurationPhase')}</span>
                <input
                  type="number"
                  min={0}
                  value={Math.round((cfg.max_duration_per_phase_ms ?? 0) / 60000) || 30}
                  onChange={(e) =>
                    patch(
                      'max_duration_per_phase_ms',
                      Number(e.target.value) * 60000,
                    )
                  }
                />
              </label>
              <label className="field inline-field">
                <span>{t('settings.budget.maxDurationQuest')}</span>
                <input
                  type="number"
                  min={0}
                  value={Math.round((cfg.max_duration_per_quest_ms ?? 0) / 60000) || 60}
                  onChange={(e) =>
                    patch(
                      'max_duration_per_quest_ms',
                      Number(e.target.value) * 60000,
                    )
                  }
                />
              </label>
              <label className="field inline-field">
                <span>{t('settings.budget.maxNoProgress')}</span>
                <input
                  type="number"
                  min={0}
                  value={cfg.max_no_progress_turns ?? 3}
                  onChange={(e) =>
                    patch('max_no_progress_turns', Number(e.target.value))
                  }
                />
              </label>
              <label className="field inline-field">
                <span>{t('settings.budget.idleWarning')}</span>
                <input
                  type="number"
                  min={0}
                  value={Math.round((cfg.runtime_idle_warning_ms ?? 60000) / 1000)}
                  onChange={(e) =>
                    patch('runtime_idle_warning_ms', Number(e.target.value) * 1000)
                  }
                />
              </label>
              <label className="field inline-field">
                <span>{t('settings.budget.maxConsecErrors')}</span>
                <input
                  type="number"
                  min={0}
                  value={cfg.max_consecutive_agent_errors ?? 2}
                  onChange={(e) =>
                    patch('max_consecutive_agent_errors', Number(e.target.value))
                  }
                />
              </label>
              <label className="field inline-field">
                <span>{t('settings.budget.maxTurnsPhase')} <em className="muted">（{t('settings.budget.safetyNet')}）</em></span>
                <input
                  type="number"
                  min={0}
                  value={cfg.max_turns_per_phase ?? 50}
                  onChange={(e) =>
                    patch('max_turns_per_phase', Number(e.target.value))
                  }
                />
              </label>
              <label className="field inline-field">
                <span>{t('settings.budget.maxRework')}</span>
                <input
                  type="number"
                  min={0}
                  value={cfg.max_rework_per_quest ?? 2}
                  onChange={(e) =>
                    patch('max_rework_per_quest', Number(e.target.value))
                  }
                />
              </label>
              <label className="field inline-field">
                <span>{t('settings.budget.maxConcurrent')}</span>
                <input
                  type="number"
                  min={0}
                  value={cfg.max_concurrent ?? 6}
                  onChange={(e) =>
                    patch('max_concurrent', Number(e.target.value))
                  }
                />
              </label>
              <label className="field inline-field">
                <span>{t('settings.budget.maxReviewHints')}</span>
                <input
                  type="number"
                  min={0}
                  value={cfg.max_review_hints ?? 2}
                  onChange={(e) =>
                    patch('max_review_hints', Number(e.target.value))
                  }
                />
              </label>
              <label className="field inline-field">
                <span>{t('settings.budget.maxExecHints')}</span>
                <input
                  type="number"
                  min={0}
                  value={cfg.max_execution_hints ?? 2}
                  onChange={(e) =>
                    patch('max_execution_hints', Number(e.target.value))
                  }
                />
              </label>
              <label className="field inline-field">
                <span>{t('settings.budget.userConfirmTimeout')}</span>
                <input
                  type="number"
                  min={0}
                  value={Math.round((cfg.user_confirm_timeout_ms ?? 0) / 3600000) || 72}
                  onChange={(e) =>
                    patch(
                      'user_confirm_timeout_ms',
                      Number(e.target.value) * 3600000,
                    )
                  }
                />
              </label>
            </div>
            <label className="field">
              <span>{t('settings.budget.commandList')}</span>
              <textarea
                rows={8}
                className="mono"
                value={commandAllowlistText}
                onChange={(e) => setCommandAllowlistText(e.target.value)}
                spellCheck={false}
              />
            </label>
            <div className="recommend-panel">
              <div className="maintenance-head">
                <div>
                  <strong>{t('settings.recommend.title')}</strong>
                  <em>{t('settings.recommend.desc')}</em>
                </div>
                <button
                  type="button"
                  className="button ghost"
                  onClick={loadCommandRecommendations}
                  disabled={commandRecommendLoading}
                >
                  <Icon name="test-tube" />
                  {commandRecommendLoading ? t('settings.recommend.scanning') : t('settings.recommend.scanBtn')}
                </button>
              </div>
              <label className="field">
                <span>{t('settings.recommend.scanDir')}</span>
                <input
                  value={commandRecommendDir}
                  onChange={(e) => setCommandRecommendDir(e.target.value)}
                  placeholder={cfg.default_working_dir || t('settings.recommend.scanPlaceholder')}
                />
              </label>
              {commandRecommendations.length > 0 && (
                <div className="recommend-list">
                  {commandRecommendations.map((item) => (
                    <div className="recommend-row" key={item.id}>
                      <div>
                        <strong>{item.id}</strong>
                        <span className="mono">{[item.command, ...(item.args || [])].join(' ')}</span>
                        <em>{item.reason || item.source || t('settings.recommend.projectHit')}</em>
                      </div>
                      <button
                        type="button"
                        className="button compact"
                        disabled={item.present}
                        onClick={() => addRecommendedCommand(item)}
                      >
                        {item.present ? t('settings.recommend.alreadyPresent') : t('settings.recommend.add')}
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </section>

        <section className="settings-section" id="settings-workspace">
          <h2>{t('settings.workspace.title')}</h2>
          <div className="settings-box">
            <label className="field">
              <span>{t('settings.workspace.defaultDir')}</span>
              <input
                value={cfg.default_working_dir || ''}
                onChange={(e) => patch('default_working_dir', e.target.value)}
                placeholder={t('settings.workspace.dirPlaceholder')}
              />
            </label>
            <div className="grid-2">
              <label className="field">
                <span>{t('settings.workspace.defaultWarrior')}</span>
                <DesignSelect
                  compact
                  value={cfg.default_warrior_id || ''}
                  options={[
                    { value: '', label: t('settings.workspace.notSpecified') },
                    ...(adventurers || [])
                      .filter((a) => a.class === 'warrior' && a.status !== 'retired')
                      .map((a) => ({
                        value: a.id,
                        label: `${a.name}${a.title ? ' · ' + a.title : ''}`,
                      })),
                  ]}
                  onChange={(v) => patch('default_warrior_id', v)}
                  ariaLabel={t('settings.workspace.defaultWarrior')}
                />
              </label>
              <label className="field">
                <span>{t('settings.workspace.defaultMage')}</span>
                <DesignSelect
                  compact
                  value={cfg.default_mage_id || ''}
                  options={[
                    { value: '', label: t('settings.workspace.notSpecified') },
                    ...(adventurers || [])
                      .filter((a) => a.class === 'mage' && a.status !== 'retired')
                      .map((a) => ({
                        value: a.id,
                        label: `${a.name}${a.title ? ' · ' + a.title : ''}`,
                      })),
                  ]}
                  onChange={(v) => patch('default_mage_id', v)}
                  ariaLabel={t('settings.workspace.defaultMage')}
                />
              </label>
            </div>
            <div className="toggles-col">
              <div className="custom-toggle-row">
                <button
                  type="button"
                  className={'custom-checkbox' + (cfg.hotl_auto_close ? ' checked' : '')}
                  onClick={() => patch('hotl_auto_close', !cfg.hotl_auto_close)}
                >
                  {cfg.hotl_auto_close && <Icon name="check" size={12} />}
                </button>
                <div>
                  <strong>{t('settings.workspace.hotlTitle')}</strong>
                  <em>{t('settings.workspace.hotlDesc')}</em>
                </div>
              </div>
              <div className="custom-toggle-row">
                <button
                  type="button"
                  className={'custom-checkbox' + (cfg.auto_cleanup_failed_quests ? ' checked' : '')}
                  onClick={() => patch('auto_cleanup_failed_quests', !cfg.auto_cleanup_failed_quests)}
                >
                  {cfg.auto_cleanup_failed_quests && <Icon name="check" size={12} />}
                </button>
                <div>
                  <strong>{t('settings.workspace.autoCleanupTitle')}</strong>
                  <em>{t('settings.workspace.autoCleanupDesc')}</em>
                </div>
              </div>
            </div>
            <div className="maintenance-panel">
              <div className="maintenance-head">
                <div>
                  <strong>{t('settings.workspace.cleanupTitle')}</strong>
                  <em>{t('settings.workspace.cleanupDesc')}</em>
                </div>
                <div className="section-actions">
                  <button
                    type="button"
                    className="button ghost"
                    onClick={() => runCleanup(true)}
                    disabled={cleanupRunning}
                  >
                    <Icon name="eye" />
                    {t('settings.workspace.preview')}
                  </button>
                  <button
                    type="button"
                    className="button danger"
                    onClick={() => runCleanup(false)}
                    disabled={cleanupRunning}
                  >
                    <Icon name="trash" />
                    {t('settings.workspace.cleanup')}
                  </button>
                </div>
              </div>
              <label className="inline-check">
                <input
                  type="checkbox"
                  checked={cleanupIncludeFailed}
                  onChange={(e) => setCleanupIncludeFailed(e.target.checked)}
                />
                {t('settings.workspace.includeFailed')}
              </label>
              {cleanupItems.length > 0 && (
                <div className="cleanup-result-list">
                  <span className="dim-meta">
                    {cleanupDryRun ? t('settings.workspace.previewHit') : t('settings.workspace.processed')} {t('settings.workspace.workspaceCount', { count: cleanupItems.length })}
                  </span>
                  {cleanupItems.slice(0, 8).map((item) => (
                    <div className="cleanup-result-row" key={item.quest_id}>
                      <code>{item.short_id || item.quest_id}</code>
                      <span>{item.status || '-'}</span>
                      <em>{item.workspace_path || '-'}</em>
                    </div>
                  ))}
                  {cleanupItems.length > 8 && <span className="dim-meta">{t('settings.workspace.moreItems', { count: cleanupItems.length - 8 })}</span>}
                </div>
              )}
              {cleanupAttempted && cleanupItems.length === 0 && (
                <div className="cleanup-result-list">
                  <span className="dim-meta">{t('settings.workspace.nothingToClean')}</span>
                </div>
              )}
            </div>
          </div>
        </section>

        <section className="settings-section" id="settings-connectors">
          <h2>{t('settings.connectors.title')}</h2>
          <p className="settings-section-hint">
            {t('settings.connectors.hint')}
          </p>
          <div className="settings-box">
            <div className="connector-block">
              <div className="connector-head">
                <Icon name="git-merge" />
                <div className="connector-head-text">
                  <strong>{t('settings.connectors.git')}</strong>
                  <span dangerouslySetInnerHTML={{ __html: t('settings.connectors.gitDesc') }} />
                </div>
                <button
                  type="button"
                  className={'toggle-switch ' + (cfg.connectors?.git?.enabled ? 'on' : '')}
                  onClick={() =>
                    setCfg((prev) => ({
                      ...prev,
                      connectors: {
                        ...prev.connectors,
                        git: { ...(prev.connectors?.git || {}), enabled: !prev.connectors?.git?.enabled },
                      },
                    }))
                  }
                  aria-label={t('settings.connectors.gitToggleAria')}
                />
              </div>
              {cfg.connectors?.git?.enabled && (
                <div className="grid-2">
                  <label className="field">
                    <span>{t('settings.connectors.remote')}</span>
                    <input
                      value={cfg.connectors?.git?.remote || ''}
                      onChange={(e) =>
                        setCfg((prev) => ({
                          ...prev,
                          connectors: {
                            ...prev.connectors,
                            git: { ...(prev.connectors?.git || {}), remote: e.target.value },
                          },
                        }))
                      }
                      placeholder="origin"
                      className="mono"
                    />
                  </label>
                  <label className="field">
                    <span>{t('settings.connectors.baseBranch')}</span>
                    <input
                      value={cfg.connectors?.git?.base_branch || ''}
                      onChange={(e) =>
                        setCfg((prev) => ({
                          ...prev,
                          connectors: {
                            ...prev.connectors,
                            git: { ...(prev.connectors?.git || {}), base_branch: e.target.value },
                          },
                        }))
                      }
                      placeholder="main"
                      className="mono"
                    />
                  </label>
                  <label className="field">
                    <span>{t('settings.connectors.authorName')}</span>
                    <input
                      value={cfg.connectors?.git?.author_name || ''}
                      onChange={(e) =>
                        setCfg((prev) => ({
                          ...prev,
                          connectors: {
                            ...prev.connectors,
                            git: { ...(prev.connectors?.git || {}), author_name: e.target.value },
                          },
                        }))
                      }
                      placeholder={t('settings.connectors.authorPlaceholder')}
                    />
                  </label>
                  <label className="field">
                    <span>{t('settings.connectors.authorEmail')}</span>
                    <input
                      value={cfg.connectors?.git?.author_email || ''}
                      onChange={(e) =>
                        setCfg((prev) => ({
                          ...prev,
                          connectors: {
                            ...prev.connectors,
                            git: { ...(prev.connectors?.git || {}), author_email: e.target.value },
                          },
                        }))
                      }
                      placeholder={t('settings.connectors.authorPlaceholder')}
                    />
                  </label>
                  <label className="check-row launch-row" style={{ gridColumn: '1 / -1' }}>
                    <input
                      type="checkbox"
                      checked={cfg.connectors?.git?.push || false}
                      onChange={(e) =>
                        setCfg((prev) => ({
                          ...prev,
                          connectors: {
                            ...prev.connectors,
                            git: { ...(prev.connectors?.git || {}), push: e.target.checked },
                          },
                        }))
                      }
                    />
                    <span>{t('settings.connectors.pushRemote')}</span>
                    <em>{t('settings.connectors.pushDesc')}</em>
                  </label>
                </div>
              )}
            </div>
          </div>
        </section>

        <section className="settings-section" id="settings-package">
          <div className="section-head">
            <h2>{t('settings.package.title')}</h2>
            <div className="section-actions">
              <button
                type="button"
                className="button ghost"
                onClick={openConfigImport}
                disabled={configImporting}
              >
                <Icon name="upload" />
                {t('settings.package.import')}
              </button>
              <button
                type="button"
                className="button ghost"
                onClick={exportConfigPackage}
                disabled={configExporting}
              >
                <Icon name="save" />
                {t('settings.package.export')}
              </button>
            </div>
          </div>
          <input
            ref={configFileInputRef}
            type="file"
            accept=".json,application/json"
            style={{ display: 'none' }}
            onChange={handleConfigFileChange}
          />
          <div className="settings-box">
            <div className="data-package-summary">
              <strong>{t('settings.package.migrateTitle')}</strong>
              <span>
                {t('settings.package.migrateInclude')}
                {t('settings.package.migrateExclude')}
              </span>
              <em>{t('settings.package.migrateNote')}</em>
            </div>
            {configImportOpen && (
              <div className="context-import-panel config-package-panel">
                <div className="import-head">
                  <h3>{t('settings.package.importTitle')}</h3>
                  <button
                    type="button"
                    className="icon-button"
                    onClick={cancelConfigImport}
                    title={t('settings.package.cancel')}
                  >
                    <Icon name="x" size={14} />
                  </button>
                </div>
                <div className="import-body">
                  <button
                    type="button"
                    className="button ghost"
                    onClick={() => configFileInputRef.current?.click()}
                    disabled={configImporting}
                  >
                    <Icon name="upload" />
                    {t('settings.package.selectFile')}
                  </button>
                  {configImportFile && (
                    <span className="dim-meta">
                      {configImportFile.name} · {formatSize(configImportFile.size)}
                    </span>
                  )}
                  {configImportPreview ? (
                    <div className="config-preview-list">
                      {configImportPreview.sections
                        .filter((s) => s.creates || s.updates || s.skips || s.warnings?.length)
                        .map((s) => (
                          <div className="config-preview-row" key={s.name}>
                            <strong>{s.name}</strong>
                            <span>
                              {t('settings.package.previewLine', { creates: s.creates, updates: s.updates, skips: s.skips })}
                            </span>
                            {!!s.warnings?.length && (
                              <em>{s.warnings.join('；')}</em>
                            )}
                          </div>
                        ))}
                      {!!configImportPreview.warnings?.length && (
                        <div className="config-preview-warning">
                          {configImportPreview.warnings.join('；')}
                        </div>
                      )}
                    </div>
                  ) : (
                    <div className="empty-page small">{t('settings.package.selectHint')}</div>
                  )}
                </div>
                <div className="import-actions">
                  <button
                    type="button"
                    className="button ghost"
                    onClick={cancelConfigImport}
                    disabled={configImporting}
                  >
                    {t('settings.package.cancel')}
                  </button>
                  <button
                    type="button"
                    className="button primary"
                    onClick={confirmConfigImport}
                    disabled={!configImportData || !configImportPreview || configImporting}
                  >
                    <Icon name="upload" />
                    {configImporting ? t('settings.package.importing') : t('settings.package.confirmImport')}
                  </button>
                </div>
              </div>
            )}
          </div>
        </section>

        <div className="settings-actions">
          <Icon name={allowlistCount == null ? 'triangle-alert' : 'info'} />
          <span>
            {loading
              ? t('settings.actions.loading')
              : allowlistCount == null
                ? t('settings.actions.jsonBroken')
                : t('settings.actions.savedNote')}
          </span>
          <button className="button primary settings-mobile-save" type="submit" disabled={saving || loading}>
            <Icon name="save" />
            {saving ? t('settings.actions.saving') : t('settings.actions.saveBtn')}
          </button>
        </div>
      </form>
      {agentDeleteConfirm && (
        <div className="sheet-scrim" style={deleteAgentScrimStyle} role="dialog" aria-modal="true" aria-label={t('settings.deleteAgent.aria')}>
          <div className="sheet-card review-dialog" ref={deleteAgentSheetRef} style={deleteAgentSheetStyle} onClick={(e) => e.stopPropagation()}>
            <div className="sheet-head">
              <span className="sheet-icon warning">
                <Icon name="trash" />
              </span>
              <div>
                <h2>{t('settings.deleteAgent.title')}</h2>
                <p>{t('settings.deleteAgent.body')}</p>
              </div>
              <button className="icon-button" onClick={() => setAgentDeleteConfirm(null)} aria-label={t('settings.deleteAgent.close')}>
                <Icon name="x" size={14} />
              </button>
            </div>
            <div className="review-dialog-body">
              <div className="settings-box">
                <strong>{agentDeleteConfirm.name}</strong>
                <span className="dim-meta mono">agents/{agentDeleteConfirm.name}.json</span>
              </div>
            </div>
            <div className="sheet-foot">
              <button className="button ghost" onClick={() => setAgentDeleteConfirm(null)} disabled={agentDeleting === agentDeleteConfirm.key}>
                {t('settings.deleteAgent.cancel')}
              </button>
              <button className="button danger" onClick={() => deleteAgent(agentDeleteConfirm.key)} disabled={agentDeleting === agentDeleteConfirm.key}>
                <Icon name="trash" />
                {agentDeleting === agentDeleteConfirm.key ? t('settings.deleteAgent.deleting') : t('settings.deleteAgent.confirm')}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
