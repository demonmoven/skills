import { ApiError, withToken } from '../api/client'
import i18n from '../i18n'
import type {
  ApplyBackupMeta,
  DesignDoc,
  KnowledgeExportDiff,
  KnowledgeExportMeta,
  QuestArtifact,
  QuestCheck,
  QuestMeta,
  SafetyWarning,
  ThreadPost,
} from '../api/types'

export type DiffFile = { name: string; additions: number; deletions: number; lines: string[] }

export function artifactURL(qid: string, artifact: QuestArtifact): string {
  return withToken('/api/quests/' + qid + '/artifacts/' + encodeURIComponent(artifact.id))
}

export function formatArtifactSize(bytes?: number): string {
  const n = Number(bytes || 0)
  if (n < 1024) return n + ' B'
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB'
  return (n / (1024 * 1024)).toFixed(1) + ' MB'
}

export function outputIconName(artifact: QuestArtifact, isExternal: boolean): string {
  if (isExternal) return 'external-link'
  switch (artifact.kind) {
    case 'image': return 'image'
    case 'archive': return 'package'
    case 'document': return 'file-text'
    case 'log': return 'file-text'
    case 'link': return 'external-link'
    default: return 'file'
  }
}

export function phaseSummaryFromToolResult(result: string): string {
  if (!result) return ''
  try {
    const parsed = JSON.parse(result)
    const data = parsed && typeof parsed === 'object' ? (parsed as { data?: unknown; summary?: unknown }) : null
    const dataObj = data?.data && typeof data.data === 'object' ? data.data as { summary?: unknown } : null
    const summary = dataObj?.summary ?? data?.summary
    return typeof summary === 'string' ? summary.trim() : ''
  } catch {
    return ''
  }
}

export function contextBlockString(block: unknown, key: 'name' | 'source' | 'trust' | 'content'): string {
  if (!block || typeof block !== 'object') return ''
  const raw = block as Record<string, unknown>
  const pascal = key.slice(0, 1).toUpperCase() + key.slice(1)
  const value = raw[key] ?? raw[pascal]
  return typeof value === 'string' ? value : ''
}

export function isTextualArtifact(artifact: QuestArtifact): boolean {
  const mime = String(artifact.mime || '').toLowerCase()
  const name = String(artifact.name || '').toLowerCase()
  return mime.startsWith('text/') ||
    mime.includes('json') ||
    mime.includes('markdown') ||
    name.endsWith('.md') ||
    name.endsWith('.txt') ||
    name.endsWith('.log') ||
    name.endsWith('.json')
}

export function normalizeTextForDedupe(value: string): string {
  return String(value || '')
    .replace(/\r\n/g, '\n')
    .replace(/[ \t\u3000]+/g, ' ')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
}

export function fastHash(value: string): number {
  let hash = 2166136261
  for (let i = 0; i < value.length; i++) {
    hash ^= value.charCodeAt(i)
    hash = Math.imul(hash, 16777619)
  }
  return hash >>> 0
}

export type SummaryArtifactMatch = { artifact: QuestArtifact; reason: 'kind_summary' | 'name_like' | 'first_textual' }
export function pickSummaryArtifact(outputs: QuestArtifact[]): SummaryArtifactMatch | null {
  if (!Array.isArray(outputs) || outputs.length === 0) return null
  const direct = outputs.find((a) => String(a.kind || '') === 'summary')
  if (direct) return { artifact: direct, reason: 'kind_summary' }
  const byName = outputs.find((a) => {
    const name = String(a.name || '').toLowerCase()
    return name.includes('summary') || name.includes('review') || name.includes('conclusion')
  })
  if (byName) return { artifact: byName, reason: 'name_like' }
  // 回退：取第一个文本产物（design quest 的 md/spec 等都算「冒险者结论」产物）
  const textual = outputs.find((a) => isTextualArtifact(a))
  return textual ? { artifact: textual, reason: 'first_textual' } : null
}

export type ApplyWarningState = {
  message: string
  warnings: SafetyWarning[]
  fromReview?: boolean
}

export type BackupDetailState = {
  loading: boolean
  item: ApplyBackupMeta | null
}

export const BLOCKED_ACTION = {
  continue: 'continue',
  userReview: 'user-review',
  cancel: 'cancel',
} as const
export type BlockedActionState = {
  action: typeof BLOCKED_ACTION[keyof typeof BLOCKED_ACTION]
  label: string
  tone: 'primary' | 'danger' | 'default'
  description: string
}

export function beginPatchFormat(raw: string): DiffFile[] | null {
  if (!raw.startsWith('*** Begin Patch') && !raw.includes('*** Begin Patch\n')) return null
  const normalized = raw.replace(/^\*\*\* Begin Patch\n?/, '')
  const segments = normalized.split(/\n(?=\*{3} Update File: )/)
  const files: DiffFile[] = []
  for (const seg of segments) {
    const head = seg.match(/^\*{3} Update File:\s*(.+?)\s*\n/)
    if (!head) continue
    const name = head[1].trim()
    const bodyStart = head[0].length
    const body = seg.slice(bodyStart).replace(/\n?\*{3} End Patch.*/s, '').replace(/\n?\*{3}$/s, '')
    const lines = body.split('\n')
    let additions = 0
    let deletions = 0
    const kept: string[] = []
    for (const line of lines) {
      if (line.startsWith('---') || line.startsWith('+++') || line.startsWith('index ')) continue
      kept.push(line)
      if (line.startsWith('+') && !line.startsWith('+++')) additions++
      else if (line.startsWith('-') && !line.startsWith('---')) deletions++
    }
    files.push({ name, additions, deletions, lines: kept })
  }
  return files.length > 0 ? files : null
}

export function unifiedDiffFormat(raw: string): DiffFile[] {
  const files: DiffFile[] = []
  const lines = raw.split('\n')
  let current: DiffFile | null = null
  for (const line of lines) {
    const fileMatch = line.match(/^diff --git a\/(.+?) b\//)
    if (fileMatch) {
      current = { name: fileMatch[1], additions: 0, deletions: 0, lines: [] }
      files.push(current)
      continue
    }
    if (!current) {
      if (lines.length > 0 && files.length === 0) {
        current = { name: '(diff)', additions: 0, deletions: 0, lines: [] }
        files.push(current)
      } else continue
    }
    if (line.startsWith('---') || line.startsWith('+++') || line.startsWith('index ')) continue
    current.lines.push(line)
    if (line.startsWith('+') && !line.startsWith('+++')) current.additions++
    if (line.startsWith('-') && !line.startsWith('---')) current.deletions++
  }
  return files
}

export function parseDiffFiles(raw: string): DiffFile[] {
  if (!raw || !raw.trim()) return []
  const begin = beginPatchFormat(raw)
  if (begin && begin.length > 0) return begin
  const unified = unifiedDiffFormat(raw)
  if (unified.length > 0) return unified
  return [{ name: '(diff)', additions: 0, deletions: 0, lines: raw.split('\n') }]
}

export function warningsFromError(e: unknown): SafetyWarning[] {
  if (!(e instanceof ApiError) || !e.data || typeof e.data !== 'object') return []
  const warnings = (e.data as { warnings?: unknown }).warnings
  return Array.isArray(warnings) ? (warnings as SafetyWarning[]) : []
}

export function warningLabel(warning: SafetyWarning): string {
  const path = warning.path ? ` · ${warning.path}` : ''
  return `[${warning.severity}] ${warning.category}${path}: ${warning.message}`
}

export function checkTitle(check: QuestCheck): string {
  const raw =
    check.name ??
    check.title ??
    check.id ??
    check.type ??
    check.category ??
    i18n.t('questDetail.checkFallback')
  return String(raw)
}

export function checkMeta(check: QuestCheck): string {
  const parts = [
    check.status,
    check.severity,
    check.score != null ? `score ${check.score}` : null,
  ].filter((item): item is string | number => item != null && String(item) !== '')
  return parts.map(String).join(' · ')
}

export function checkMessage(check: QuestCheck): string {
  const raw = check.message ?? check.summary ?? check.detail ?? check.reason
  if (raw != null && String(raw).trim()) return String(raw)
  return JSON.stringify(check, null, 2)
}

export async function writeClipboardText(value: string): Promise<boolean> {
  if (navigator.clipboard?.writeText && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(value)
      return true
    } catch {
      // Fall through to the textarea path for local HTTP or permission-denied cases.
    }
  }

  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.top = '0'
  textarea.style.left = '0'
  textarea.style.width = '1px'
  textarea.style.height = '1px'
  textarea.style.opacity = '0'

  document.body.appendChild(textarea)
  textarea.focus()
  textarea.select()
  try {
    return document.execCommand('copy')
  } finally {
    document.body.removeChild(textarea)
  }
}

export type KnowledgeArtifactPreview = {
  artifact: QuestArtifact
  meta: KnowledgeExportMeta | null
  diff: KnowledgeExportDiff | null
}

export type QuestDisplaySection =
  | { kind: 'warrior_message'; source: string; content: string; ts: number; turn: number; phase: number }
  | { kind: 'warrior_context_artifact'; source: string; content: string; ts: number; turn: number; phase: number }
  | { kind: 'warrior_tool_summary'; source: string; content: string; ts: number; turn: number; phase: number }
  | { kind: 'summary_artifact'; source: string; content: string; artifactId: string; artifactName: string }
  | { kind: 'warrior_summary'; source: string; content: string }
  | { kind: 'design_summary'; source: string; content: string }

export type QuestDisplayInputs = {
  quest?: Pick<QuestMeta, 'design_summary' | 'warrior_summary'> | null
  messages: Array<{ content: string; phase: number; ts: number; turn: number }>
  contextPacks: Array<{ ts: number; turn: number; phase: number; data: { blocks?: unknown[] } }>
  toolResults: Array<{ content: string; phase: number; ts: number; turn: number; toolName: string }>
  summaryArtifact?: SummaryArtifactMatch | null
  summaryArtifactText?: string | null
}

export function selectQuestProducerArtifact(input: QuestDisplayInputs): QuestDisplaySection | null {
  // 产物 md 优先：产物是 warrior 的实际交付物，优于 turn summary（口头描述）。
  // turn summary 常含 docs/xxx.md 引用，渲染成 inline code 体验差；产物 md 全文才是「冒险者结论」该展示的。
  if (input.summaryArtifact && input.summaryArtifactText && input.summaryArtifactText.trim()) {
    return {
      kind: 'summary_artifact',
      source: i18n.t('questDetail.source.textArtifact', { name: input.summaryArtifact.artifact.name }),
      content: input.summaryArtifactText,
      artifactId: input.summaryArtifact.artifact.id,
      artifactName: input.summaryArtifact.artifact.name,
    }
  }

  // WarriorSummary：phase done --summary 的人类可读摘要（v0.3.8+）
  const warriorSummary = String(input.quest?.warrior_summary || '').trim()
  if (warriorSummary) {
    return { kind: 'warrior_summary', source: i18n.t('questDetail.source.executorConclusion'), content: warriorSummary }
  }

  const orderedMessages = [...input.messages].sort((a, b) => b.ts - a.ts)
  const warrior = orderedMessages.find((m) => {
    if ((m.phase ?? 0) !== 0) return false
    const text = String(m.content || '').trim()
    if (!text) return false
    if (text.startsWith('{"type":"system"') || text.startsWith('[{"type":"system"')) return false
    if (text.startsWith('{') && text.includes('"hook_id"')) return false
    return true
  })
  if (warrior) {
    return {
      kind: 'warrior_message',
      source: i18n.t('questDetail.source.warriorTurn', { turn: warrior.turn }),
      content: warrior.content,
      ts: warrior.ts,
      turn: warrior.turn,
      phase: warrior.phase ?? 0,
    }
  }

  const warriorContext = selectWarriorContextArtifact(input.contextPacks)
  if (warriorContext) {
    return warriorContext
  }

  const warriorTool = [...input.toolResults]
    .sort((a, b) => b.ts - a.ts)
    .find((item) => (item.phase ?? 0) === 0 && phaseSummaryFromToolResult(item.content))
  if (warriorTool) {
    return {
      kind: 'warrior_tool_summary',
      source: i18n.t('questDetail.source.warriorDeliver', { tool: warriorTool.toolName || 'phase done' }),
      content: phaseSummaryFromToolResult(warriorTool.content),
      ts: warriorTool.ts,
      turn: warriorTool.turn,
      phase: warriorTool.phase ?? 0,
    }
  }

  const designSummary = String(input.quest?.design_summary || '').trim()
  if (designSummary) {
    return { kind: 'design_summary', source: i18n.t('questDetail.source.designSummary'), content: designSummary }
  }
  return null
}

function selectWarriorContextArtifact(contextPacks: QuestDisplayInputs['contextPacks']): QuestDisplaySection | null {
  const packs = [...contextPacks].sort((a, b) => b.ts - a.ts)
  for (const pack of packs) {
    const blocks = pack.data.blocks || []
    for (const block of blocks) {
      const name = contextBlockString(block, 'name')
      const source = contextBlockString(block, 'source')
      const trust = contextBlockString(block, 'trust')
      const content = contextBlockString(block, 'content').trim()
      if (!content) continue
      if (name === 'warrior_artifact' || (source === 'warrior' && trust === 'agent_output')) {
        return {
          kind: 'warrior_context_artifact',
          source: i18n.t('questDetail.source.warriorContext'),
          content,
          ts: pack.ts,
          turn: pack.turn,
          phase: 0,
        }
      }
    }
  }
  return null
}

export function shouldShowFinalVerdict(quest: Pick<QuestMeta, 'finalized_by'> | null | undefined): boolean {
  return Boolean(quest?.finalized_by)
}

// v0.5 Slice 3: workflow mode pill + upgrade button 的纯函数决策。
// 只允许 direct → checked → goal 单向升档；blocked / ended / waiting_input / goal / 未知 mode 不显示按钮。
export type WorkflowUpgradeInfo = {
  pillLabel: string | null
  pillClass: string | null
  canUpgrade: boolean
  nextMode: 'checked' | 'goal' | ''
  nextLabel: string
}

export function getWorkflowUpgradeInfo(
  workflowMode: string | undefined,
  isBlocked: boolean,
  isEnded: boolean,
  isWaitingInput: boolean,
  isRuntimeActive = false,
): WorkflowUpgradeInfo {
  const mode = workflowMode || ''
  const pillLabel = mode === 'direct' ? 'Direct' : mode === 'checked' ? 'Checked' : mode === 'goal' ? 'Goal' : null
  const pillClass = mode === 'goal' ? 'workflow-pill goal'
    : mode === 'checked' ? 'workflow-pill checked'
    : mode === 'direct' ? 'workflow-pill direct'
    : null
  // 前端 runtime-active gate + 后端 UpgradeWorkflowMode 双重保护（PA-BE-1 已落地）
  const canUpgrade = !isBlocked && !isEnded && !isWaitingInput && !isRuntimeActive && (mode === 'direct' || mode === 'checked')
  const nextMode = mode === 'direct' ? 'checked' : mode === 'checked' ? 'goal' : ''
  const nextLabel = nextMode === 'checked' ? i18n.t('questDetail.upgrade.checked') : nextMode === 'goal' ? i18n.t('questDetail.upgrade.goal') : ''
  return { pillLabel, pillClass, canUpgrade, nextMode, nextLabel }
}

export interface GoalRuntimeView {
  showPanel: boolean
  docReady: boolean
  decisionRecorded: boolean
  executeStatus: string
  steps: string[]
  showSpawnButton: boolean
  showOpenChildButton: boolean
  showSpawnError: boolean
  showAutoSpawnPlaceholder: boolean
  childExecuteQuestID: string | null
  spawnError: string | null
}

export function deriveGoalRuntimeView(
  designDoc: DesignDoc | null | undefined,
  quest: QuestMeta | null | undefined,
  threadPosts: ThreadPost[],
  canSpawn: boolean,
): GoalRuntimeView {
  const showPanel = !!designDoc || quest?.type === 'design'
  const docReady = !!designDoc
  const decisionRecorded = threadPosts.some((p) => p.kind === 'decision_note') || !!quest?.pinned_outcome_summary
  const childID = quest?.child_execute_quest_id ? String(quest.child_execute_quest_id) : null
  const spawnErr = quest?.spawn_error || null
  let executeStatus = i18n.t('questDetail.goal.waitingDesign')
  if (childID) executeStatus = i18n.t('questDetail.goal.created', { id: childID.slice(-6) })
  else if (spawnErr) executeStatus = i18n.t('questDetail.goal.failed')
  else if (canSpawn) executeStatus = i18n.t('questDetail.goal.ready')
  const steps = designDoc?.implementation_steps && designDoc.implementation_steps.length > 0
    ? designDoc.implementation_steps
    : []
  const showSpawnButton = canSpawn
  const showOpenChildButton = !!childID
  const showSpawnError = !!spawnErr
  const showAutoSpawnPlaceholder = !canSpawn && !childID && !spawnErr && !!quest?.auto_spawn_execute
  return {
    showPanel,
    docReady,
    decisionRecorded,
    executeStatus,
    steps,
    showSpawnButton,
    showOpenChildButton,
    showSpawnError,
    showAutoSpawnPlaceholder,
    childExecuteQuestID: childID,
    spawnError: spawnErr,
  }
}
