import type { MakerReport } from '../api/types'
import type { QuestDisplaySection } from '../pages/questDetailHelpers'
import { useTranslation } from 'react-i18next'
import AgentMessageRenderer from './AgentMessageRenderer'
import Icon from './Icon'

type Props = {
  report: MakerReport | null
  fallback: QuestDisplaySection | null
  loadingSummaryArtifact?: boolean
  onJumpToArtifact?: (artifactId: string) => void
  onJumpToTrace?: () => void
}

export default function MakerReportCard({ report, fallback, loadingSummaryArtifact, onJumpToArtifact, onJumpToTrace }: Props) {
  const { t } = useTranslation()
  if (!report && !fallback && !loadingSummaryArtifact) return null

  const sourceLabel = report
    ? 'MakerReport · ' + report.schema_version
    : fallback
      ? fallback.source
      : t('makerReport.fallback')

  return (
    <section id="quest-maker-delivery" className="panel loop-report-card maker-report-card mt-3" data-legacy-anchor="quest-adventurer-conclusion">
      <div className="panel-title">
        <h2>Maker Delivery</h2>
        <span className="mono">{sourceLabel}</span>
        <div className="panel-title-actions">
          {fallback?.kind === 'summary_artifact' && onJumpToArtifact && (
            <button type="button" className="link-button tiny" onClick={() => onJumpToArtifact(fallback.artifactId)}>
              <Icon name="file-text" size={12} /> {t('makerReport.viewArtifact')}
            </button>
          )}
          {onJumpToTrace && (
            <button type="button" className="link-button tiny" onClick={onJumpToTrace} title={t('makerReport.jumpToTrace')}>
              <Icon name="search" size={12} /> {t('makerReport.viewProcess')}
            </button>
          )}
        </div>
      </div>

      {report ? (
        <div className="loop-report-body">
          <AgentMessageRenderer content={report.summary} maxLength={2400} />
          <div className="review-meta-row">
            <span className="report-chip">verdict: {report.verdict}</span>
            <span className="report-chip">phase: {report.phase}</span>
            {report.artifacts?.length ? <span className="report-chip">artifacts: {report.artifacts.length}</span> : null}
            {report.changeset && <span className="report-chip">changeset</span>}
            {report.transition && <span className="report-chip">transition</span>}
          </div>
          {report.open_questions?.length ? (
            <div className="loop-report-list">
              <strong>Open Questions</strong>
              <ul>
                {report.open_questions.map((item, index) => <li key={index}>{item}</li>)}
              </ul>
            </div>
          ) : null}
        </div>
      ) : fallback ? (
        <AgentMessageRenderer
          content={fallback.content}
          className="loop-report-body"
          maxLength={2400}
        />
      ) : (
        <div className="exec-trace-empty">
          <Icon name="spinner" className="spin" />
          <span>{t('makerReport.loading')}</span>
        </div>
      )}
    </section>
  )
}
