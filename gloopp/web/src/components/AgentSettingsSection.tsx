import DesignSelect from './DesignSelect'
import Icon from './Icon'
import { capabilityTierLabel } from './util'
import CapabilityTierBadge from './CapabilityTierBadge'
import i18n from '../i18n'
import { useTranslation } from 'react-i18next'

const BUILTIN_AGENT_TYPES: Record<string, AgentForm['type']> = {
  traex: 'acp',
  omp: 'acp',
  codex: 'acp',
  claude: 'acp',
  relay: 'cli',
  pi: 'cli',
  aiden: 'cli',
  aiden_x_claude: 'cli',
  aiden_x_codex: 'cli',
}

export type AgentForm = {
  id?: string
  name: string
  type: 'acp' | 'cli' | 'mock'
  command: string
  argsText: string
  envText: string
  defaultModel: string
  enabled: boolean
}

function agentName(p: any): string {
  return p?.name || p?.agent || ''
}

function isOfficialAgent(p: any): boolean {
  return !!p?.official || !!BUILTIN_AGENT_TYPES[agentName(p)]
}

function builtinAgentType(p: any): AgentForm['type'] | undefined {
  return BUILTIN_AGENT_TYPES[agentName(p)]
}

function typeLabel(value?: string): string {
  return (value || 'cli').toUpperCase()
}

export function agentToForm(p: any): AgentForm {
  return {
    id: p.id || (p.name ? 'exe_' + p.name : ''),
    name: p.name || p.agent || '',
    type: p.type || 'cli',
    command: p.command || '',
    argsText: Array.isArray(p.args) ? p.args.join('\n') : '',
    envText: p.env && typeof p.env === 'object'
      ? Object.entries(p.env).map(([k, v]) => `${k}=${String(v)}`).join('\n')
      : '',
    defaultModel: p.default_model || '',
    enabled: p.enabled !== false,
  }
}

function modelPolicyLabel(value?: string): string {
  const model = (value || '').trim()
  return model ? i18n.t('settings.agent.modelPolicy.specified', { model }) : i18n.t('settings.agent.modelPolicy.default')
}

type Props = {
  loading: boolean
  agents: any[]
  newAgentKeys: string[]
  agentDrafts: Record<string, AgentForm>
  expandedAgentKeys: Record<string, boolean>
  agentSaving: string | null
  agentDeleting: string | null
  onAddAgent: () => void
  onToggleExecutor: (id: string, enabled: boolean) => void
  onToggleExpanded: (key: string) => void
  onRemoveDraft: (key: string) => void
  onPatchAgent: (key: string, patch: Partial<AgentForm>) => void
  onSaveAgent: (key: string) => void
  onRequestDelete: (key: string, name: string) => void
}

export default function AgentSettingsSection({
  loading,
  agents,
  newAgentKeys,
  agentDrafts,
  expandedAgentKeys,
  agentSaving,
  agentDeleting,
  onAddAgent,
  onToggleExecutor,
  onToggleExpanded,
  onRemoveDraft,
  onPatchAgent,
  onSaveAgent,
  onRequestDelete,
}: Props) {
  const { t } = useTranslation()
  const rows = [
    ...newAgentKeys.map((key) => ({
      key,
      item: { name: '', type: 'cli', enabled: true, command: '', default_model: '', __new: true },
    })),
    ...agents.map((p) => ({ key: p.name || p.agent || p.id, item: p })),
  ]

  return (
    <section className="settings-section">
      <div className="section-title-row">
        <h2>{t('settings.agent.section.title')}</h2>
        <button type="button" className="button" onClick={onAddAgent}>
          <Icon name="plus" />
          {t('settings.agent.section.add')}
        </button>
      </div>
      <div className="settings-box divide">
        {loading && <div className="empty-page small">{t('settings.agent.empty.loading')}</div>}
        {!loading && agents.length === 0 && newAgentKeys.length === 0 && (
          <div className="empty-page small">{t('settings.agent.empty.none')}</div>
        )}
        {rows.map(({ key, item: p }) => {
          const draft = agentDrafts[key] || agentToForm(p)
          const expanded = !!expandedAgentKeys[key]
          const locked = isOfficialAgent(p)
          const lockedType = builtinAgentType(p)
          const typeText = typeLabel(lockedType || draft.type || p.type)
          return (
            <article className={'settings-row agent-settings-row' + (expanded ? ' expanded' : '')} key={key}>
              <div className="agent-summary">
                <strong>{p.__new ? t('settings.agent.row.newAgent') : draft.name || p.name}</strong>
                <span className="agent-type-tag">{typeText}</span>
                {locked && <span className="agent-type-tag official">{t('settings.agent.row.official')}</span>}
                {p.capability_tier && <CapabilityTierBadge tier={p.capability_tier} compact />}
                <span className={'agent-status-dot ' + (draft.enabled ? 'on' : 'off')} />
                <em className="dim-meta">
                  {draft.command || p.command || t('settings.agent.row.noCommand')} · {modelPolicyLabel(draft.defaultModel || p.default_model)}
                </em>
              </div>
              <div className="agent-row-actions">
                {!p.__new && (
                  <button
                    type="button"
                    className={'toggle-switch ' + (p.enabled ? 'on' : '')}
                    onClick={() => onToggleExecutor(p.name, !p.enabled)}
                    aria-label={t('settings.agent.aria.toggle')}
                  />
                )}
                <button
                  type="button"
                  className="button ghost compact"
                  onClick={() => onToggleExpanded(key)}
                >
                  <Icon name={expanded ? 'chevron-down' : 'pencil'} />
                  {expanded ? t('settings.agent.action.collapse') : t('settings.agent.action.edit')}
                </button>
                {p.__new && (
                  <button
                    type="button"
                    className="icon-button"
                    onClick={() => onRemoveDraft(key)}
                    aria-label={t('settings.agent.aria.removeDraft')}
                  >
                    <Icon name="x" size={14} />
                  </button>
                )}
              </div>
              {expanded && (
                <div className="agent-editor full">
                  {locked && (
                    <div className="agent-lock-note full">
                      <Icon name="shield" size={14} />
                      <span>{t('settings.agent.lockNote', { type: typeText })}</span>
                    </div>
                  )}
                  <div className="agent-editor-group agent-editor-basic">
                    <label className="field">
                      <span>{t('settings.agent.field.name')}</span>
                      <input
                        value={draft.name}
                        disabled={locked}
                        onChange={(e) => onPatchAgent(key, { name: e.target.value })}
                        placeholder={t('settings.agent.field.namePlaceholder')}
                      />
                      {locked && <small className="field-hint">{t('settings.agent.field.nameHint')}</small>}
                    </label>
                    <label className="field">
                      <span>{t('settings.agent.field.type')}</span>
                      {locked ? (
                        <>
                          <input value={typeText} disabled />
                          <small className="field-hint">{t('settings.agent.field.typeHint')}</small>
                        </>
                      ) : (
                        <DesignSelect
                          compact
                          value={draft.type}
                          options={[
                            { value: 'acp', label: 'ACP' },
                            { value: 'cli', label: 'CLI' },
                            { value: 'mock', label: 'Mock' },
                          ]}
                          onChange={(v) => onPatchAgent(key, { type: v as AgentForm['type'] })}
                          ariaLabel={t('aria.type')}
                        />
                      )}
                    </label>
                    <label className="field">
                      <span>{t('settings.agent.field.command')}</span>
                      <input
                        value={draft.command}
                        onChange={(e) => onPatchAgent(key, { command: e.target.value })}
                        placeholder={t('settings.agent.field.commandPlaceholder')}
                      />
                    </label>
                    <label className="field">
                      <span>{t('settings.agent.field.defaultModel')}</span>
                      <input
                        value={draft.defaultModel}
                        onChange={(e) => onPatchAgent(key, { defaultModel: e.target.value })}
                        placeholder={t('settings.agent.field.modelPlaceholder')}
                      />
                      <small className="field-hint">{modelPolicyLabel(draft.defaultModel)}</small>
                    </label>
                    <label className="field">
                      <span>{t('settings.agent.field.capability')}</span>
                      <div className="capability-field">
                        <CapabilityTierBadge tier={p.capability_tier} />
                      </div>
                      <small className="field-hint">{t('settings.agent.field.capabilityHint')}</small>
                    </label>
                  </div>
                  <div className="agent-editor-group agent-editor-textareas">
                    <label className="field">
                      <span>{t('settings.agent.field.args')}</span>
                      <textarea
                        rows={3}
                        value={draft.argsText}
                        onChange={(e) => onPatchAgent(key, { argsText: e.target.value })}
                        placeholder={t('settings.agent.field.argsPlaceholder')}
                      />
                    </label>
                    <label className="field">
                      <span>{t('settings.agent.field.env')}</span>
                      <textarea
                        rows={3}
                        value={draft.envText}
                        onChange={(e) => onPatchAgent(key, { envText: e.target.value })}
                        placeholder={t('settings.agent.field.envPlaceholder')}
                      />
                    </label>
                  </div>
                  <div className="agent-editor-actions full">
                    <label className="inline-check">
                      <input
                        type="checkbox"
                        checked={draft.enabled}
                        onChange={(e) => onPatchAgent(key, { enabled: e.target.checked })}
                      />
                      {t('settings.agent.action.enable')}
                    </label>
                    <button
                      type="button"
                      className="button primary"
                      disabled={agentSaving === key}
                      onClick={() => onSaveAgent(key)}
                    >
                      <Icon name="save" />
                      {agentSaving === key ? t('settings.agent.action.saving') : t('settings.agent.action.save')}
                    </button>
                    {!p.__new && (
                      <button
                        type="button"
                        className="button danger ghost"
                        disabled={locked || agentDeleting === key}
                        title={locked ? t('settings.agent.action.deleteLockedTitle') : undefined}
                        onClick={() => !locked && onRequestDelete(key, draft.name || key)}
                      >
                        <Icon name="trash" />
                        {agentDeleting === key ? t('settings.agent.action.deleting') : t('settings.agent.action.delete')}
                      </button>
                    )}
                  </div>
                </div>
              )}
            </article>
          )
        })}
      </div>
    </section>
  )
}
