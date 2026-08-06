import { useState, useMemo } from 'react'
import type { ContextPackData, ContextBlock } from '../api/types'
import { blockKindLabel, blockRoleLabel, blockStalenessLabel, KEY_CONTEXT_BLOCKS } from '../domain/contextLabels'
import i18n from '../i18n'
import { useTranslation } from 'react-i18next'
import Icon from './Icon'
import './ContextPackView.css'

interface Props {
  data: ContextPackData
  compact?: boolean
}

const TRUST_COLORS: Record<string, { bg: string; text: string; labelKey: string }> = {
  platform: { bg: 'rgba(59, 130, 246, 0.12)', text: 'var(--blue, #3b82f6)', labelKey: 'contextPack.trust.platform' },
  user: { bg: 'rgba(34, 197, 94, 0.12)', text: 'var(--green, #22c55e)', labelKey: 'contextPack.trust.user' },
  agent_output: { bg: 'rgba(249, 115, 22, 0.12)', text: 'var(--orange, #f97316)', labelKey: 'contextPack.trust.aiOutput' },
  mixed: { bg: 'rgba(139, 92, 246, 0.12)', text: 'var(--purple, #8b5cf6)', labelKey: 'contextPack.trust.mixed' },
}

const ROLE_COLORS: Record<string, { bg: string; text: string }> = {
  control: { bg: 'rgba(239, 68, 68, 0.10)', text: 'var(--red, #ef4444)' },
  primary_input: { bg: 'rgba(34, 197, 94, 0.10)', text: 'var(--green, #22c55e)' },
  evidence: { bg: 'rgba(59, 130, 246, 0.10)', text: 'var(--blue, #3b82f6)' },
  history: { bg: 'rgba(107, 114, 128, 0.12)', text: 'var(--ink-500, #6b7280)' },
  protocol: { bg: 'rgba(139, 92, 246, 0.10)', text: 'var(--purple, #8b5cf6)' },
  supporting: { bg: 'rgba(249, 115, 22, 0.10)', text: 'var(--orange, #f97316)' },
}

function trustStyle(trust?: string) {
  const entry = TRUST_COLORS[trust || '']
  if (entry) return { bg: entry.bg, text: entry.text, label: i18n.t(entry.labelKey) }
  return { bg: 'rgba(107, 114, 128, 0.12)', text: 'var(--ink-500, #6b7280)', label: trust || i18n.t('contextPack.trust.unknown') }
}

function formatChars(chars: number): string {
  if (chars >= 1000) return `${(chars / 1000).toFixed(1)}k`
  return `${chars}`
}

function roleStyle(role?: string) {
  return ROLE_COLORS[role || ''] || { bg: 'rgba(107, 114, 128, 0.10)', text: 'var(--ink-500, #6b7280)' }
}

export default function ContextPackView({ data, compact = false }: Props) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [showRaw, setShowRaw] = useState(false)

  const blocks = useMemo(() => (data.blocks || []).map(normalizeContextBlock), [data.blocks])
  const summary = data.summary
  const roleCounts = useMemo(() => {
    const counts = new Map<string, number>()
    for (const block of blocks) {
      const role = block.role || 'uncategorized'
      counts.set(role, (counts.get(role) || 0) + 1)
    }
    return Array.from(counts.entries())
  }, [blocks])

  const missingKeyBlocks = useMemo(() => {
    const names = new Set(blocks.map(b => b.name))
    return KEY_CONTEXT_BLOCKS.filter(kb => {
      // 只提示跟当前 context kind 相关的关键块
      if (summary?.kind?.includes('review') && kb.name === 'execution_instruction') return false
      if (!summary?.kind?.includes('review') && kb.name === 'review_protocol') return false
      return kb.name !== 'execution_instruction' || summary?.kind?.includes('execution') || summary?.kind?.includes('quest_execution')
    }).filter(kb => !names.has(kb.name))
  }, [blocks, summary])

  function toggleBlock(name: string) {
    setExpanded(prev => {
      const next = new Set(prev)
      if (next.has(name)) next.delete(name)
      else next.add(name)
      return next
    })
  }

  function expandAll() {
    setExpanded(new Set(blocks.map(b => b.name)))
  }

  function collapseAll() {
    setExpanded(new Set())
  }

  if (blocks.length === 0 && !summary && !data.rendered) {
    return (
      <div className="ctx-pack empty">
        <span className="muted">{t('contextPack.empty')}</span>
      </div>
    )
  }

  if (showRaw) {
    return (
      <div className="ctx-pack raw-view">
        <div className="ctx-pack-header">
          <span className="ctx-pack-title">
            <Icon name="code" size={14} />
            {t('contextPack.rawPrompt')}
          </span>
          <button className="text-button" onClick={() => setShowRaw(false)}>
            {t('contextPack.backToStructured')}
          </button>
        </div>
        <pre className="ctx-pack-raw">{data.rendered || t('contextPack.noRawText')}</pre>
      </div>
    )
  }

  return (
    <div className={`ctx-pack ${compact ? 'compact' : ''}`}>
      <div className="ctx-pack-header">
        <div className="ctx-pack-title-row">
          <span className="ctx-pack-title">
            <Icon name="layers" size={14} />
            {t('contextPack.title')}
          </span>
          {summary?.kind && (
            <span className="ctx-pack-kind">{summary.kind}</span>
          )}
        </div>
        <div className="ctx-pack-actions">
          {!compact && (
            <>
              <button className="text-button" onClick={expandAll}>{t('contextPack.expandAll')}</button>
              <button className="text-button" onClick={collapseAll}>{t('contextPack.collapseAll')}</button>
            </>
          )}
          {data.rendered && (
            <button className="text-button" onClick={() => setShowRaw(true)}>
              {t('contextPack.viewRaw')}
            </button>
          )}
        </div>
      </div>

      {summary && !compact && (
        <div className="ctx-pack-stats">
          <span className="stat">
            <em>{blocks.length}</em> {t('contextPack.stat.blocks')}
          </span>
          <span className="stat">
            <em>{formatChars(summary.rendered_chars || 0)}</em> {t('contextPack.stat.chars')}
          </span>
          {summary.truncated && (
            <span className="stat warn">
              <Icon name="warning" size={12} />
              {t('contextPack.stat.truncated')}
            </span>
          )}
        </div>
      )}

        {roleCounts.length > 0 && !compact && (
          <div className="ctx-pack-roles">
            {roleCounts.map(([role, count]) => {
              const style = roleStyle(role)
              return (
                <span key={role} className="ctx-pack-role-chip" style={{ background: style.bg, color: style.text }}>
                  {blockRoleLabel(role === 'uncategorized' ? undefined : role)} {count}
                </span>
              )
            })}
          </div>
        )}

      {missingKeyBlocks.length > 0 && !compact && (
        <div className="ctx-pack-missing">
          <div className="ctx-pack-missing-title">
            <Icon name="info" size={14} />
            {t('contextPack.missing.title')}
          </div>
          <div className="ctx-pack-missing-list">
            {missingKeyBlocks.map(kb => (
              <div key={kb.name} className="ctx-pack-missing-item" title={kb.hint}>
                <span className="ctx-pack-missing-name">{kb.label}</span>
                {kb.required && <span className="ctx-pack-missing-tag">{t('contextPack.missing.suggest')}</span>}
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="ctx-pack-blocks">
        {blocks.map((block) => (
          <ContextBlockRow
            key={block.name}
            block={block}
            expanded={expanded.has(block.name)}
            onToggle={() => toggleBlock(block.name)}
          />
        ))}
        {blocks.length === 0 && summary?.blocks && (
          // 只有摘要没有完整 blocks 时，显示摘要视图
          summary.blocks.map((b) => (
            <div key={b.name} className="ctx-pack-block summary-only">
              <div className="ctx-pack-block-head">
                <span
                  className="ctx-pack-block-trust"
                  style={{ background: trustStyle(b.trust).bg, color: trustStyle(b.trust).text }}
                >
                  {trustStyle(b.trust).label}
                </span>
                <span className="ctx-pack-block-name">{blockKindLabel(b.name)}</span>
                <span className="ctx-pack-block-meta">
                  {formatChars(b.chars)} {t('contextPack.stat.chars')}
                  {b.truncated && t('contextPack.truncatedSuffix')}
                </span>
              </div>
              <div className="ctx-pack-block-preview muted">
                {b.preview || t('contextPack.noPreview')}
                <span className="ctx-pack-block-hint">{t('contextPack.noFullContent')}</span>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}

function normalizeContextBlock(block: ContextBlock): ContextBlock {
  const raw = block as ContextBlock & Record<string, unknown>
  return {
    ...block,
    name: String(block.name || raw.Name || ''),
    source: block.source || String(raw.Source || ''),
    trust: block.trust || String(raw.Trust || ''),
    phase: block.phase || String(raw.Phase || ''),
      role: block.role || String(raw.Role || ''),
      staleness: block.staleness || String(raw.Staleness || ''),
      priority: block.priority ?? (typeof raw.Priority === 'number' ? raw.Priority : undefined),
    user_controlled: block.user_controlled ?? Boolean(raw.UserControlled),
    content: String(block.content || raw.Content || ''),
  }
}

function ContextBlockRow({
  block,
  expanded,
  onToggle,
}: {
  block: ContextBlock
  expanded: boolean
  onToggle: () => void
}) {
  const { t } = useTranslation()
  const style = trustStyle(block.trust)
    const role = roleStyle(block.role)
  const chars = [...(block.content || '')].length

  return (
    <div className={`ctx-pack-block ${expanded ? 'expanded' : ''}`}>
      <button className="ctx-pack-block-head" onClick={onToggle}>
        <span className="ctx-pack-block-expand">
          <Icon name="chevron-right" size={12} className={expanded ? 'rotated' : ''} />
        </span>
        <span
          className="ctx-pack-block-trust"
          style={{ background: style.bg, color: style.text }}
        >
          {style.label}
        </span>
        <span className="ctx-pack-block-name">{blockKindLabel(block.name)}</span>
          {block.role && (
            <span
              className="ctx-pack-block-role"
              style={{ background: role.bg, color: role.text }}
            >
              {blockRoleLabel(block.role)}
            </span>
          )}
        <span className="ctx-pack-block-meta">
          {formatChars(chars)} {t('contextPack.stat.chars')}
            {block.staleness && ` · ${blockStalenessLabel(block.staleness)}`}
            {block.priority != null && ` · P${block.priority}`}
          {block.truncated && t('contextPack.truncatedSuffix')}
          {block.user_controlled && t('contextPack.editableSuffix')}
        </span>
      </button>
      {expanded && (
        <div className="ctx-pack-block-body">
          {block.source && (
            <div className="ctx-pack-block-source muted">
              {t('contextPack.source.prefix')}{block.source}
              {block.phase && t('contextPack.source.phase', { phase: block.phase })}
                {block.role && t('contextPack.source.role', { role: blockRoleLabel(block.role) })}
                {block.staleness && t('contextPack.source.staleness', { staleness: blockStalenessLabel(block.staleness) })}
            </div>
          )}
          <pre className="ctx-pack-block-content">{block.content}</pre>
        </div>
      )}
    </div>
  )
}
