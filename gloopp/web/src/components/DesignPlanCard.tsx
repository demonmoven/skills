import type { QuestArtifact } from '../api/types'
import i18n from '../i18n'
import { useTranslation } from 'react-i18next'
import Icon from './Icon'
import MarkdownRenderer from './MarkdownRenderer'

type Props = {
  artifact: QuestArtifact | null
  href?: string
  content?: string
  loading?: boolean
}

export default function DesignPlanCard({ artifact, href, content = '', loading = false }: Props) {
  const { t } = useTranslation()
  if (!artifact) return null
  const summary = firstMeaningfulLine(content) || artifact.name || t('designPlan.fallback')
  const external = artifact.storage_path?.startsWith('http://') || artifact.storage_path?.startsWith('https://')
  const target = href || (external ? artifact.storage_path : undefined)

  return (
    <section className="panel design-plan-card">
      <div className="panel-title">
        <h2><Icon name="clipboard-check" size={14} /> {t('designPlan.title')}</h2>
        <span>{artifact.kind}</span>
      </div>
      <div className="design-plan-body">
        <p title={summary}>{loading ? t('designPlan.loading') : summary}</p>
        {target && (
          <a className="text-button" href={target} target="_blank" rel="noreferrer">
            {t('designPlan.open')}
          </a>
        )}
      </div>
      {content.trim() && (
        <details className="design-plan-detail">
          <summary>{t('designPlan.viewFull')}</summary>
          <MarkdownRenderer source={content} className="review-copy compact" />
        </details>
      )}
    </section>
  )
}

function firstMeaningfulLine(content: string): string {
  const line = content
    .split('\n')
    .map((item) => item.replace(/^#+\s*/, '').trim())
    .find(Boolean)
  return line || ''
}
