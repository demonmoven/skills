import type { EvidenceRef, ReviewReport, StructuredReview } from '../api/types'
import { useTranslation } from 'react-i18next'
import AgentMessageRenderer from './AgentMessageRenderer'
import Icon from './Icon'
import StructuredReviewSection from './StructuredReviewSection'

type Props = {
  report: ReviewReport
  onJumpToEvidence?: (ref: EvidenceRef, sr?: StructuredReview) => void
  onJumpToTrace?: () => void
}

export default function ReviewReportCard({ report, onJumpToEvidence, onJumpToTrace }: Props) {
  const { t } = useTranslation()
  return (
    <section id="quest-review-report" className="panel loop-report-card review-report-card mt-3">
      <div className="panel-title">
        <h2>Checker Review</h2>
        <span className="mono">{report.schema_version}</span>
        {onJumpToTrace && (
          <div className="panel-title-actions">
            <button type="button" className="link-button tiny" onClick={onJumpToTrace}>
              <Icon name="search" size={12} /> {t('reviewReport.viewProcess')}
            </button>
          </div>
        )}
      </div>
      <div className="review-meta-row">
        <span className="report-chip">verdict: {report.verdict}</span>
        <span className="report-chip">confidence: {report.confidence}</span>
        <span className="report-chip">checked: {report.checked_against?.length || 0}</span>
        {report.evidence_refs?.length ? <span className="report-chip">evidence: {report.evidence_refs.length}</span> : null}
        {report.required_changes?.length ? <span className="report-chip required">required: {report.required_changes.length}</span> : null}
        {report.residual_risks?.length ? <span className="report-chip risk">risk: {report.residual_risks.length}</span> : null}
        {report.transition_note && <span className="report-chip">transition</span>}
      </div>

      {report.structured_review && (
        <StructuredReviewSection sr={report.structured_review} onJumpToEvidence={onJumpToEvidence} />
      )}

      {report.comment && (
        <AgentMessageRenderer
          content={report.comment}
          className="loop-report-body"
          maxLength={1600}
        />
      )}

      {(report.required_changes?.length || report.residual_risks?.length || report.checked_against?.length) ? (
        <div className="review-report-grid">
          {report.required_changes?.length ? (
            <div className="loop-report-list required">
              <strong>Required Changes</strong>
              <ul>
                {report.required_changes.map((item, index) => <li key={index}>{item}</li>)}
              </ul>
            </div>
          ) : null}
          {report.residual_risks?.length ? (
            <div className="loop-report-list risk">
              <strong>Residual Risk</strong>
              <ul>
                {report.residual_risks.map((item, index) => <li key={index}>{item}</li>)}
              </ul>
            </div>
          ) : null}
          {report.checked_against?.length ? (
            <div className="loop-report-list checked">
              <strong>Checked Against</strong>
              <ul>
                {report.checked_against.slice(0, 6).map((item, index) => <li key={index}>{item}</li>)}
              </ul>
            </div>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}
