import Icon from './Icon'
import i18n from '../i18n'
import { useTranslation } from 'react-i18next'

type QuestSummaryParts = {
  title: string
  body: string
  lineCount: number
  isLong: boolean
}

type QuestSummaryProps = {
  query: string
  variant: 'card' | 'row' | 'detail'
  title?: string
  expanded?: boolean
  copied?: boolean
  onCopy?: () => void
  onToggleExpanded?: () => void
}

export function splitQuestSummary(query: string, titleLimit = 72): QuestSummaryParts {
  const cleanQuery = query.trim()
  const lines = cleanQuery
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean)

  if (lines.length === 0) {
    return { title: i18n.t('questSummary.untitled'), body: '', lineCount: 0, isLong: false }
  }

  const firstLine = lines[0]
  const hasLongFirstLine = firstLine.length > titleLimit
  const title = hasLongFirstLine ? firstLine.slice(0, titleLimit).trimEnd() + '...' : firstLine
  const body = hasLongFirstLine
    ? [firstLine.slice(titleLimit).trim(), ...lines.slice(1)].filter(Boolean).join('\n')
    : lines.slice(1).join('\n')

  return {
    title,
    body,
    lineCount: lines.length,
    isLong: cleanQuery.length > 220 || lines.length > 2 || body.length > 140,
  }
}

export default function QuestSummary({
  query,
  variant,
  title,
  expanded = false,
  copied = false,
  onCopy,
  onToggleExpanded,
}: QuestSummaryProps) {
  const { t } = useTranslation()
  const summary = splitQuestSummary(query)
  const hasBody = Boolean(summary.body)
  const displayTitle = title || t('questSummary.defaultTitle')

  if (variant === 'row') {
    return (
      <span className={'quest-summary quest-summary-row' + (hasBody ? ' has-body' : '')}>
        <span className="quest-summary-title">{summary.title}</span>
        {hasBody && <span className="quest-summary-body">{summary.body}</span>}
      </span>
    )
  }

  if (variant === 'detail') {
    return (
      <section className="panel quest-summary quest-summary-detail">
        <div className="panel-title quest-summary-bar">
          <h2>{displayTitle}</h2>
          <div className="quest-summary-actions">
            {summary.lineCount > 1 && <span className="mono">{t('questSummary.lineCount', { count: summary.lineCount })}</span>}
            {onCopy && (
              <button className="icon-button" type="button" onClick={onCopy} aria-label={t('questSummary.copy')} title={t('questSummary.copy')}>
                <Icon name={copied ? 'check' : 'copy'} size={15} />
              </button>
            )}
            {summary.isLong && onToggleExpanded && (
              <button className="button tiny" type="button" onClick={onToggleExpanded}>
                {expanded ? t('questSummary.collapse') : t('questSummary.expand')}
              </button>
            )}
          </div>
        </div>
        <div className={'quest-summary-content' + (expanded ? ' expanded' : '') + (summary.isLong ? ' clamped' : '')}>
          {query}
        </div>
      </section>
    )
  }

  return (
    <div className={'quest-summary quest-summary-card' + (hasBody ? ' has-body' : '')}>
      <strong className="quest-summary-title">{summary.title}</strong>
      {hasBody && <p className="quest-summary-body">{summary.body}</p>}
    </div>
  )
}
