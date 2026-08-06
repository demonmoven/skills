import type { AllowedCommand } from '../../api/types'
import i18n from '../../i18n'

export type GlobalCfg = {
  host?: string
  port?: number
  max_turns_per_phase?: number
  max_rework_per_quest?: number
  max_concurrent?: number
  default_working_dir?: string
  default_warrior_id?: string
  default_mage_id?: string
  hotl_auto_close?: boolean
  workspace_retention_days?: number
  auto_cleanup_failed_quests?: boolean
  max_duration_per_phase_ms?: number
  max_duration_per_quest_ms?: number
  user_confirm_timeout_ms?: number
  max_review_hints?: number
  max_execution_hints?: number
  max_consecutive_agent_errors?: number
  max_no_progress_turns?: number
  runtime_idle_warning_ms?: number
  mage_command_allowlist?: AllowedCommand[]
  connectors?: {
    git?: {
      enabled?: boolean
      remote?: string
      base_branch?: string
      author_name?: string
      author_email?: string
      push?: boolean
    }
  }
  [k: string]: unknown
}

export function listFromText(text: string): string[] {
  return text
    .split(/\r?\n|,/)
    .map((item) => item.trim())
    .filter(Boolean)
}

export function envFromText(text: string): Record<string, string> {
  const env: Record<string, string> = {}
  for (const raw of text.split(/\r?\n/)) {
    const line = raw.trim()
    if (!line || line.startsWith('#')) continue
    const idx = line.indexOf('=')
    if (idx <= 0) continue
    env[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
  }
  return env
}

export function parseAllowlistCount(text: string): number | null {
  try {
    const parsed = JSON.parse(text || '[]')
    return Array.isArray(parsed) ? parsed.length : null
  } catch {
    return null
  }
}

export function parseMageCommandAllowlist(text: string): AllowedCommand[] {
  const parsed = JSON.parse(text || '[]')
  if (!Array.isArray(parsed)) {
    throw new Error(i18n.t('settingsModel.mageAllowlistMustBeArray'))
  }
  return parsed
}

export function buildSettingsPatch(cfg: GlobalCfg, mageCommandAllowlist: AllowedCommand[]): Record<string, unknown> {
  const body: Record<string, unknown> = {}
  function push(k: string, v: unknown) {
    if (v !== undefined && v !== null) body[k] = v
  }
  push('max_turns_per_phase', cfg.max_turns_per_phase)
  push('max_rework_per_quest', cfg.max_rework_per_quest)
  push('max_concurrent', cfg.max_concurrent)
  push('default_working_dir', cfg.default_working_dir)
  push('default_warrior_id', cfg.default_warrior_id)
  push('default_mage_id', cfg.default_mage_id)
  push('hotl_auto_close', cfg.hotl_auto_close)
  push('workspace_retention_days', cfg.workspace_retention_days)
  push('auto_cleanup_failed_quests', cfg.auto_cleanup_failed_quests)
  push('max_duration_per_phase_ms', cfg.max_duration_per_phase_ms)
  push('max_duration_per_quest_ms', cfg.max_duration_per_quest_ms)
  push('user_confirm_timeout_ms', cfg.user_confirm_timeout_ms)
  push('max_review_hints', cfg.max_review_hints)
  push('max_execution_hints', cfg.max_execution_hints)
  push('max_consecutive_agent_errors', cfg.max_consecutive_agent_errors)
  push('max_no_progress_turns', cfg.max_no_progress_turns)
  push('runtime_idle_warning_ms', cfg.runtime_idle_warning_ms)
  push('mage_command_allowlist', mageCommandAllowlist)
  push('connectors', cfg.connectors)
  return body
}
