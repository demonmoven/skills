import { Children, isValidElement, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { MarkdownHooks } from 'react-markdown'
import type { Pluggable } from 'unified'
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize'
import rehypeSlug from 'rehype-slug'
import rehypeAutolinkHeadings from 'rehype-autolink-headings'
import rehypePrettyCode from 'rehype-pretty-code'
import remarkGfm from 'remark-gfm'
import remarkBreaks from 'remark-breaks'
import type { Element, Root, Text } from 'hast'
import type { Schema } from 'hast-util-sanitize'
import i18n from '../i18n'
import { useTranslation } from 'react-i18next'
import Icon from './Icon'

type Props = {
  source: string
  className?: string
  emptyLabel?: string
  stripFrontmatter?: boolean
}

export function stripYamlFrontmatter(md: string): string {
  const trimmed = md.replace(/^\uFEFF/, '')
  if (!trimmed.startsWith('---')) return md
  const match = trimmed.match(/^---\r?\n[\s\S]*?\r?\n---\r?\n?/)
  if (!match) return md
  return trimmed.slice(match[0].length).trimStart()
}

function textFromChildren(children: ReactNode): string {
  return Children.toArray(children).map((child) => {
    if (typeof child === 'string' || typeof child === 'number') return String(child)
    return ''
  }).join('')
}

function isDiffLanguage(language: string): boolean {
  return ['diff', 'patch', 'udiff'].includes(language.toLowerCase())
}

function diffStats(code: string): { additions: number; deletions: number } {
  return code.split('\n').reduce((acc, line) => {
    if (line.startsWith('+') && !line.startsWith('+++')) acc.additions += 1
    if (line.startsWith('-') && !line.startsWith('---')) acc.deletions += 1
    return acc
  }, { additions: 0, deletions: 0 })
}

function looksLikeFileRef(value: string): boolean {
  const trimmed = value.trim()
  if (!trimmed || /\s/.test(trimmed)) return false
  if (trimmed.length > 160) return false
  if (/^[a-z]+:\/\//i.test(trimmed)) return false
  if (trimmed.startsWith('-')) return false
  return /(^\.{1,2}\/|^[\w@.-]+\/|^[\w@.-]+\.[a-zA-Z0-9]{1,8}(:\d+)?$)/.test(trimmed)
}

function textFromHast(node: Element | Root): string {
  let out = ''
  const walk = (n: unknown) => {
    if (!n) return
    if (typeof n === 'object' && n !== null) {
      const obj = n as { type?: string; value?: unknown; children?: unknown[] }
      if (obj.type === 'text') out += String(obj.value ?? '')
      if (Array.isArray(obj.children)) obj.children.forEach(walk)
    }
  }
  walk(node)
  return out
}

function firstClassLanguage(className?: string | string[] | unknown): string {
  if (!className) return ''
  const cls = Array.isArray(className) ? className.join(' ') : String(className)
  return cls.match(/language-([^\s]+)/)?.[1] || ''
}

function CopyButton({ value, label, title }: { value: string; label: string; title: string }) {
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)

  async function copy() {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1200)
    } catch {
      setCopied(false)
    }
  }

  return (
    <button type="button" onClick={copy} title={title}>
      <Icon name={copied ? 'check' : 'copy'} size={13} />
      {copied ? t('markdown.copied') : label}
    </button>
  )
}

function FileRef({ value }: { value: string }) {
  const { t } = useTranslation()
  return (
    <span className="markdown-file-ref" title={value}>
      <Icon name="file-text" size={12} />
      <span>{value}</span>
      <CopyButton value={value} label={t('markdown.copy')} title={t('markdown.copyPath')} />
    </span>
  )
}

function HeadingLinkIcon() {
  // Visually subtle anchor icon; styles in styles.css under .markdown-heading-anchor
  return <span className="markdown-heading-anchor" aria-hidden>#</span>
}

// ---- rehype-sanitize custom schema ----
// Extend default schema so:
//  - Shiki/rehype-pretty-code output (<span class/data-line/...>, <code data-language>) survives
//  - Heading ids from rehype-slug survive
//  - rehype-autolink-headings extra <a class="markdown-heading-anchor"> survives
//  - KaTeX / Mermaid class attributes + data-* survive (keeps option to add later)
//  - data-* and ARIA are allowed globally
const sanitizeSchema = (() => {
  const base = defaultSchema as Schema
  const tagNames = Array.from(new Set([...(base.tagNames || []), 'kbd', 'sup', 'sub', 'article', 'aside', 'figure', 'figcaption', 'details', 'summary', 'mark']))
  // rehype-sanitize's PropertyDefinition supports strings and [name, predicate] tuples (including RegExp).
  // We cast to its accepted type after building.
  type AttrMap = Record<string, unknown[]>
  const attributes: AttrMap = { ...(base.attributes as AttrMap || {}) }
  const allowList = ['id', 'className', 'class', 'data-language', 'data-theme', 'data-line', 'data-chars', 'data-highlighted-line', 'data-line-numbers', 'role', 'aria-label', 'aria-hidden', 'aria-labelledby', 'tabindex']
  // per-tag extensions
  for (const tag of ['span', 'code', 'pre', 'div', 'p', 'ul', 'ol', 'li', 'a', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'table', 'thead', 'tbody', 'tr', 'th', 'td', 'blockquote', 'del', 'ins', 'strong', 'em']) {
    const existing: unknown[] = attributes[tag] ? [...(attributes[tag] as unknown[])] : []
    for (const attr of allowList) {
      if (!existing.includes(attr)) existing.push(attr)
    }
    attributes[tag] = existing
  }
  // allow data-* globally via the [name, regex] tuple form rehype-sanitize accepts.
  if (!attributes['*']) attributes['*'] = []
  attributes['*'] = Array.from(new Set([
    ...(attributes['*'] || []) as unknown[],
    ['data', /^data-/] as unknown,
    'className', 'class', 'role', 'aria-label', 'aria-hidden', 'aria-labelledby', 'tabindex',
  ]))
  // a: keep href/target/rel
  attributes['a'] = Array.from(new Set([
    ...(attributes['a'] || []) as unknown[],
    'href', 'target', 'rel', 'name',
  ]))
  // img: alt/title/src
  attributes['img'] = Array.from(new Set([
    ...(attributes['img'] || []) as unknown[],
    'src', 'alt', 'title', 'width', 'height', 'loading',
  ]))
  // allow style on span/code (Shiki inline themes, though we use CSS vars; belt & suspenders)
  for (const tag of ['span', 'code', 'pre']) {
    const list: unknown[] = attributes[tag] || []
    if (!list.includes('style')) list.push('style')
    attributes[tag] = list
  }
  // protocols: keep default protocols + allow file refs (mailto/tel already fine)
  const protocols = base.protocols || { href: ['http', 'https', 'mailto', ':'] }
  return {
    tagNames,
    attributes: attributes as Schema['attributes'],
    protocols,
    strip: base.strip ?? [],
    required: base.required ?? {},
    ancestors: base.ancestors ?? {},
    clobberPrefix: base.clobberPrefix ?? 'user-content-',
  }
})()

// ---- rehype-pretty-code options ----
// Use a dual CSS-var theme so tokens don't depend on inline styles (sanitize-safe + dark-mode friendly).
// We define vars in styles.css.
const prettyCodeOptions = {
  theme: {
    dark: 'github-dark-dimmed',
    light: 'github-light',
  } as const,
  keepBackground: false,
  defaultLang: 'plaintext',
  // Enable onVisitLine for line numbers; however to keep things simple, defer line numbers opt-in via meta.
} satisfies Parameters<typeof rehypePrettyCode>[0]

export default function MarkdownRenderer({
  source,
  className,
  emptyLabel,
  stripFrontmatter = false,
}: Props) {
  const { t } = useTranslation()
  const body = stripFrontmatter ? stripYamlFrontmatter(source || '') : source || ''
  const normalized = body.trim() ? body : (emptyLabel ?? t('markdown.empty'))

  const plugins = useMemo(() => ({
    remark: [
      remarkGfm,
      // GFM's break behavior requires trailing two spaces; swordsman output usually uses natural line breaks.
      remarkBreaks,
    ] as Pluggable[],
    rehype: [
      // Order matters: pretty-code first (so its spans survive slug/autolink), then slug, then autolink, then sanitize last.
      [rehypePrettyCode, prettyCodeOptions],
      rehypeSlug,
      [rehypeAutolinkHeadings, {
        behavior: 'prepend' as const,
        content: () => [HeadingLinkIcon()],
        properties: {
          className: ['markdown-heading-link'],
          ariaLabel: 'Link to this section',
        },
      }],
      [rehypeSanitize, sanitizeSchema],
    ] as Pluggable[],
  }), [])

  return (
    <article className={['markdown-renderer', className].filter(Boolean).join(' ')}>
      <MarkdownHooks
        remarkPlugins={plugins.remark}
        rehypePlugins={plugins.rehype}
        fallback={<span className="markdown-renderer-fallback">{normalized}</span>}
        components={{
          a({ href, children, ...props }) {
            const external = typeof href === 'string' && /^https?:\/\//.test(href)
            return (
              <a
                href={href}
                target={external ? '_blank' : undefined}
                rel={external ? 'noreferrer noopener' : undefined}
                {...props}
              >
                {children}
                {external ? <Icon name="external-link" size={12} aria-hidden className="markdown-external-icon" /> : null}
              </a>
            )
          },
          table({ children, ...props }) {
            return (
              <div className="markdown-table-wrap">
                <table {...props}>{children}</table>
              </div>
            )
          },
          li({ children, ...props }) {
            // Make GFM task list checkboxes interactive (native input, not static).
            const first = Children.toArray(children)[0]
            let checkbox: ReactNode = null
            let rest = children
            if (isValidElement(first) && typeof first.type === 'string' && first.type === 'input') {
              const f = first as React.ReactElement<{ type?: string; checked?: boolean }>
              if (f.props.type === 'checkbox') {
                checkbox = (
                  <input
                    type="checkbox"
                    defaultChecked={Boolean(f.props.checked)}
                    aria-label="task checkbox"
                    className="markdown-task-checkbox"
                    readOnly
                  />
                )
                const arr = Children.toArray(children)
                rest = arr.slice(1)
              }
            }
            return (
              <li {...props}>
                {checkbox}
                {rest}
              </li>
            )
          },
          pre({ children, node, className: preClassName }) {
            // Read raw text + language from the hast node so we keep Shiki token spans intact.
            const codeElement: Element | null = node && 'children' in node && Array.isArray(node.children)
              ? (node.children.find((c) => (c as { type?: string }).type === 'element' && (c as Element).tagName === 'code') as Element | undefined) || null
              : null
            const language = firstClassLanguage(codeElement?.properties?.className)
            const rawCode = codeElement ? textFromHast(codeElement).replace(/\n$/, '') : textFromChildren(children)
            const isDiff = isDiffLanguage(language)
            const stats = isDiff ? diffStats(rawCode) : null

            return (
              <div className={[
                'markdown-codeblock',
                isDiff ? 'markdown-diffblock' : '',
                'markdown-codeblock--highlighted',
              ].filter(Boolean).join(' ')}>
                <div className="markdown-codebar">
                  <span>
                    {language || 'text'}
                    {stats && (
                      <em>
                        <b>+{stats.additions}</b>
                        <b>-{stats.deletions}</b>
                      </em>
                    )}
                  </span>
                  <CopyButton value={rawCode} label={t('markdown.copy')} title={t('markdown.copyCode')} />
                </div>
                {codeElement ? (
                  // Shiki / rehype-pretty-code already built <pre><code>...</code></pre>.
                  // Render the hast-derived React children directly to preserve token spans.
                  <pre className={String(preClassName || '')}>{children}</pre>
                ) : (
                  <pre className={String(preClassName || '')}>{children}</pre>
                )}
              </div>
            )
          },
          code({ children, className, ...props }) {
            const value = textFromChildren(children)
            // Only treat inline code (no language class) as a possible file ref.
            if (!className && looksLikeFileRef(value)) {
              return <FileRef value={value.trim()} />
            }
            return (
              <code className={className} {...props}>
                {children}
              </code>
            )
          },
        }}
      >
        {normalized}
      </MarkdownHooks>
    </article>
  )
}
