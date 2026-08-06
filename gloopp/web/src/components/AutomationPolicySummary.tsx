import type { TrustTier } from '../api/types'
import { capabilityTierLabel } from './util'
import Icon from './Icon'
import { useTranslation } from 'react-i18next'
import i18n from '../i18n'

export type AutomationPolicyInput = {
  triageMode?: string
  autoStart?: boolean
  autoApply?: boolean
  allowL2?: boolean
  trustTier?: TrustTier
  connectors?: string[]
  enabledConnectors?: string[]
  capabilityTier?: string
}

export function automationTriageMode(input: Pick<AutomationPolicyInput, 'triageMode' | 'autoStart'>): 'candidate' | 'direct' | string {
  if (input.triageMode) return input.triageMode
  return input.autoStart ? 'direct' : 'candidate'
}

export function automationPolicyWarnings(input: AutomationPolicyInput): string[] {
  const warnings: string[] = []
  const triageMode = automationTriageMode(input)
  const connectors = input.connectors || []
  const enabled = new Set(input.enabledConnectors || [])
  const disabledConnectors = connectors.filter((name) => !enabled.has(name))
  if (disabledConnectors.length > 0) {
    warnings.push(i18n.t('automation.warning.connectorNotEnabled', { names: disabledConnectors.join(', ') }))
  }
  if (triageMode === 'direct' && input.capabilityTier === 'tier_c') {
    warnings.push(i18n.t('automation.warning.directTierC'))
  }
  if (input.autoApply && input.allowL2) {
    warnings.push(i18n.t('automation.warning.autoApplyL2Conflict'))
  }
  return warnings
}

export type AutomationFlow = 'direct' | 'checked' | 'goal'

export type AutomationFlowConfig = {
  quest_type?: string
  intensity?: string
  with_design_phase?: boolean
}

export function flowFromAutomationConfig(config: AutomationFlowConfig): AutomationFlow {
  if (config.quest_type === 'design' || config.with_design_phase) return 'goal'
  if (config.intensity === 'quick') return 'direct'
  return 'checked'
}

export function patchFromAutomationFlow(flow: AutomationFlow): {
  quest_type: string
  intensity: string
  with_design_phase: boolean
} {
  switch (flow) {
    case 'direct':
      return { quest_type: 'execute', intensity: 'quick', with_design_phase: false }
    case 'checked':
      return { quest_type: 'execute', intensity: 'standard', with_design_phase: false }
    case 'goal':
      return { quest_type: 'design', intensity: 'standard', with_design_phase: false }
  }
}

export function automationFlowLabel(flow: AutomationFlow): string {
  const map: Record<AutomationFlow, string> = {
    direct: i18n.t('automation.flow.label.direct'),
    checked: i18n.t('automation.flow.label.checked'),
    goal: i18n.t('automation.flow.label.goal'),
  }
  return map[flow]
}

export function automationFlowDescription(flow: AutomationFlow): string {
  const map: Record<AutomationFlow, string> = {
    direct: i18n.t('automation.flow.desc.direct'),
    checked: i18n.t('automation.flow.desc.checked'),
    goal: i18n.t('automation.flow.desc.goal'),
  }
  return map[flow]
}

export function automationFlowIcon(flow: AutomationFlow): string {
  switch (flow) {
    case 'direct': return 'zap'
    case 'checked': return 'clipboard-check'
    case 'goal': return 'wand'
  }
}

export type AutomationCreateForm = {
  name: string
  description?: string
  query: string
  workingDir?: string
  trigger?: string
  cron?: string
  warriorId?: string
  mageId?: string
  autoStart: boolean
  autoApply: boolean
  allowL2: boolean
  planOnly: boolean
  connectors: string[]
}

export function buildAutomationCreatePayload(form: AutomationCreateForm, flow: AutomationFlow) {
  const flowPatch = patchFromAutomationFlow(flow)
  const canPlanOnlyToggle = flow === 'goal' && flowPatch.intensity !== 'quick'
  const payload: Record<string, unknown> = {
    name: form.name.trim(),
    description: (form.description || '').trim(),
    query: form.query.trim(),
    quest_type: flowPatch.quest_type,
    intensity: flowPatch.intensity,
    working_dir: (form.workingDir || '').trim(),
    trigger: form.trigger,
    cron: form.cron,
    warrior_id: form.warriorId,
    mage_id: form.mageId,
    auto_start: form.autoStart,
    triage_mode: form.autoStart ? 'direct' : 'candidate',
    auto_apply: form.autoApply,
    allow_l2: form.allowL2,
    with_design_phase: flowPatch.with_design_phase,
    connectors: form.connectors,
    enabled: true,
  }
  if (canPlanOnlyToggle) {
    payload.auto_spawn_execute = !form.planOnly
  }
  return payload
}

export type AutomationUpdateForm = {
  name?: string
  description?: string
  query?: string
  quest_type?: string
  with_design_phase?: boolean
  intensity?: string
  working_dir?: string
  trigger?: string
  cron?: string
  auto_start?: boolean
  auto_apply?: boolean
  allow_l2?: boolean
  auto_spawn_execute?: boolean
  warrior_id?: string
  mage_id?: string
  connectors?: string[]
  priority?: number
}

export function buildAutomationUpdatePayload(
  form: AutomationUpdateForm,
  currentFlow: AutomationFlow,
  previousFlow: AutomationFlow,
  activeAutoSpawnExecute: boolean | undefined,
  canPlanOnlyToggle: boolean,
  guardedAutomation: boolean,
  activeAutoStart: boolean | undefined,
) {
  const payload: Record<string, unknown> = {
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
    auto_apply: guardedAutomation ? false : form.auto_apply,
    allow_l2: guardedAutomation ? false : form.allow_l2,
    warrior_id: form.warrior_id,
    mage_id: form.mage_id,
    connectors: form.connectors,
    priority: form.priority,
    triage_mode: (form.auto_start ?? activeAutoStart) ? 'direct' : 'candidate',
  }
  if (currentFlow === 'goal' && canPlanOnlyToggle) {
    payload.auto_spawn_execute = form.auto_spawn_execute ?? activeAutoSpawnExecute ?? false
  } else if (previousFlow === 'goal' && currentFlow !== 'goal') {
    payload.auto_spawn_execute = false
  }
  return payload
}

export default function AutomationPolicySummary({ input, compact = false }: { input: AutomationPolicyInput; compact?: boolean }) {
  const { t } = useTranslation()
  const triageMode = automationTriageMode(input)
  const warnings = automationPolicyWarnings(input)
  const connectors = input.connectors || []
  return (
    <section className={'automation-policy-summary' + (compact ? ' compact' : '')} aria-label={t('aria.automationPolicySummary')}>
      <div className="automation-policy-head">
        <Icon name="shield" size={14} />
        <strong>{t('term.policy')}</strong>
        <span className={'policy-mode mode-' + triageMode}>
          {triageMode === 'direct' ? t('automation.value.direct') : t('automation.value.candidate')}
        </span>
      </div>
      <div className="automation-policy-grid">
        <PolicyCell label={t('automation.policy.triage')} value={triageMode === 'direct' ? t('automation.value.directRun') : t('automation.value.candidateFirst')} />
        <PolicyCell label={t('automation.policy.trust')} value={input.trustTier ? String(input.trustTier) : t('automation.value.unset')} />
        <PolicyCell label={t('automation.policy.autoApply')} value={input.autoApply ? t('automation.value.on') : t('automation.value.off')} />
        <PolicyCell label={t('automation.policy.l2')} value={input.allowL2 ? t('automation.value.allowed') : t('automation.value.blocked')} />
        <PolicyCell label={t('automation.policy.agentTier')} value={input.capabilityTier ? capabilityTierLabel(input.capabilityTier) : t('automation.value.auto')} />
        <PolicyCell label={t('automation.policy.connectors')} value={connectors.length ? connectors.join(', ') : t('automation.value.none')} />
      </div>
      {warnings.length > 0 && (
        <div className="automation-policy-warnings">
          {warnings.map((warning) => (
            <span key={warning}><Icon name="triangle-alert" size={12} />{warning}</span>
          ))}
        </div>
      )}
    </section>
  )
}

function PolicyCell({ label, value }: { label: string; value: string }) {
  return (
    <span className="automation-policy-cell">
      <em>{label}</em>
      <strong>{value}</strong>
    </span>
  )
}
