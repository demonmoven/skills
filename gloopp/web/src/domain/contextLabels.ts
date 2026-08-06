import i18n from '../i18n'

export const BLOCK_KIND_LABELS: Record<string, string> = {
  phase_control_panel: i18n.t('contextLabel.phase_control_panel'),
  review_control_panel: i18n.t('contextLabel.review_control_panel'),
  latest_review_blockers: i18n.t('contextLabel.latest_review_blockers'),
  conflict_policy: i18n.t('contextLabel.conflict_policy'),
  original_task_boundary: i18n.t('contextLabel.original_task_boundary'),
  completion_checklist: i18n.t('contextLabel.completion_checklist'),
  quest_user_intent: i18n.t('contextLabel.quest_user_intent'),
  intensity_note: i18n.t('contextLabel.intensity_note'),
  design_quest_note: i18n.t('contextLabel.design_quest_note'),
  rework_hints: i18n.t('contextLabel.rework_hints'),
  previous_round_summary: i18n.t('contextLabel.previous_round_summary'),
  workspace_diff_snapshot: i18n.t('contextLabel.workspace_diff_snapshot'),
  rework_progress: i18n.t('contextLabel.rework_progress'),
  acceptance_criteria: i18n.t('contextLabel.acceptance_criteria'),
  execution_instruction: i18n.t('contextLabel.execution_instruction'),
  warrior_artifact: i18n.t('contextLabel.warrior_artifact'),
  platform_tool_evidence: i18n.t('contextLabel.platform_tool_evidence'),
  review_history: i18n.t('contextLabel.review_history'),
  review_evidence_pack: i18n.t('contextLabel.review_evidence_pack'),
  review_requirements: i18n.t('contextLabel.review_requirements'),
  review_protocol: i18n.t('contextLabel.review_protocol'),
  context_catalog: i18n.t('contextLabel.context_catalog'),
  context_files: i18n.t('contextLabel.context_files'),
  platform_auto_evidence: i18n.t('contextLabel.platform_auto_evidence'),
  platform_mechanisms: i18n.t('contextLabel.platform_mechanisms'),
  warrior_native_tool_calls: i18n.t('contextLabel.warrior_native_tool_calls'),
  warrior_deliverables: i18n.t('contextLabel.warrior_deliverables'),
  warrior_summary: i18n.t('contextLabel.warrior_summary'),
  design_plan: i18n.t('contextLabel.design_plan'),
  design_review_requirements: i18n.t('contextLabel.design_review_requirements'),
  design_evidence_pack: i18n.t('contextLabel.design_evidence_pack'),
  design_alignment_review: i18n.t('contextLabel.design_alignment_review'),
  implementation_review_summary: i18n.t('contextLabel.implementation_review_summary'),
}

export function blockKindLabel(name: string): string {
  return BLOCK_KIND_LABELS[name] || name
}

export const BLOCK_ROLE_LABELS: Record<string, string> = {
  control: i18n.t('contextRole.control'),
  primary_input: i18n.t('contextRole.primary_input'),
  evidence: i18n.t('contextRole.evidence'),
  history: i18n.t('contextRole.history'),
  protocol: i18n.t('contextRole.protocol'),
  supporting: i18n.t('contextRole.supporting'),
}

export function blockRoleLabel(role?: string): string {
  return role ? (BLOCK_ROLE_LABELS[role] || role) : i18n.t('contextRole.unclassified')
}

export const BLOCK_STALENESS_LABELS: Record<string, string> = {
  live: i18n.t('contextStaleness.live'),
  phase_snapshot: i18n.t('contextStaleness.phase_snapshot'),
  previous_review: i18n.t('contextStaleness.previous_review'),
  previous_round: i18n.t('contextStaleness.previous_round'),
  original: i18n.t('contextStaleness.original'),
  historical: i18n.t('contextStaleness.historical'),
}

export function blockStalenessLabel(staleness?: string): string {
  return staleness ? (BLOCK_STALENESS_LABELS[staleness] || staleness) : ''
}

export const KEY_CONTEXT_BLOCKS = [
  { name: 'phase_control_panel', label: blockKindLabel('phase_control_panel'), hint: i18n.t('contextHint.phase_control_panel') },
  { name: 'review_control_panel', label: blockKindLabel('review_control_panel'), hint: i18n.t('contextHint.review_control_panel') },
  { name: 'conflict_policy', label: blockKindLabel('conflict_policy'), hint: i18n.t('contextHint.conflict_policy') },
  { name: 'quest_user_intent', label: blockKindLabel('quest_user_intent'), hint: i18n.t('contextHint.quest_user_intent'), required: true },
  { name: 'acceptance_criteria', label: blockKindLabel('acceptance_criteria'), hint: i18n.t('contextHint.acceptance_criteria') },
  { name: 'execution_instruction', label: blockKindLabel('execution_instruction'), hint: i18n.t('contextHint.execution_instruction') },
  { name: 'review_protocol', label: blockKindLabel('review_protocol'), hint: i18n.t('contextHint.review_protocol') },
]
