import MarkdownRenderer from './MarkdownRenderer'

type Props = {
  content: string
  className?: string
  compact?: boolean
  maxLength?: number
}

function looksLikeJson(value: string): boolean {
  const trimmed = value.trim()
  if (!trimmed) return false
  return (trimmed.startsWith('{') && trimmed.endsWith('}')) || (trimmed.startsWith('[') && trimmed.endsWith(']'))
}

function looksLikeMarkdown(value: string): boolean {
  return /(^|\n)(#{1,6}\s|[-*+]\s|\d+\.\s|>\s|```|\|.+\||!\[|\[[^\]]+\]\([^)]+\))|(\*\*[^*]+\*\*)|(`[^`]+`)/.test(value)
}

function fencePlainText(value: string, language = 'text'): string {
  return '```' + language + '\n' + value.replace(/```/g, '`\\`\\`') + '\n```'
}

function normalizeAgentContent(content: string): string {
  const trimmed = content.trim()
  if (!trimmed) return ''
  if (looksLikeJson(trimmed)) return fencePlainText(trimmed, 'json')
  if (trimmed.includes('\n') && !looksLikeMarkdown(trimmed)) return fencePlainText(trimmed)
  return content
}

export default function AgentMessageRenderer({
  content,
  className,
  compact = false,
  maxLength,
}: Props) {
  const body = maxLength && content.length > maxLength
    ? content.slice(0, maxLength) + '\n...'
    : content

  return (
    <MarkdownRenderer
      source={normalizeAgentContent(body)}
      className={[
        'agent-message-renderer',
        compact ? 'compact' : '',
        className,
      ].filter(Boolean).join(' ')}
      emptyLabel=""
    />
  )
}
