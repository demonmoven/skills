/**
 * StructuredReviewSection — 法师 typed 评审协议卡片（Phase 2）。
 *
 * 见 mage-review-spec v0.2 §5.5。在 MageReviewBand 展开态 comment 之前渲染。
 * StructuredReview 为 nil 时整区不渲染。
 *
 * 三块：
 *   - disagreements（异议）：dimension tag + severity 色 + claim + evidence + suggestion
 *   - risks_extra（额外风险）：kind + severity + detail + evidence
 *   - endorsements（亮点）：dimension + reason + evidence（≤3 条，compact）
 *
 * 每张卡右下角有 audit 标签「来源：法师 structured_review v1」——
 * 让用户知道分类是 mage 自己分的，不是平台猜的。
 */

import type {
  StructuredReview,
  MageDisagreement,
  MageExtraRisk,
  MageEndorsement,
  EvidenceRef,
  MageDimension,
  MageSeverity,
  EvidenceTargetKind,
} from '../api/types'
import i18n from '../i18n'
import { useTranslation } from 'react-i18next'

// ---------------------------------------------------------------------------
// 标签查表（纯展示，不含语义判断）
// ---------------------------------------------------------------------------

const DIMENSION_LABELS: Record<MageDimension, string> = {
  correctness: i18n.t('structuredReview.dimension.correctness'),
  completeness: i18n.t('structuredReview.dimension.completeness'),
  boundary: i18n.t('structuredReview.dimension.boundary'),
  risk: i18n.t('structuredReview.dimension.risk'),
  validation: i18n.t('structuredReview.dimension.validation'),
  coherence: i18n.t('structuredReview.dimension.coherence'),
  scope: i18n.t('structuredReview.dimension.scope'),
  style: i18n.t('structuredReview.dimension.style'),
  other: i18n.t('structuredReview.dimension.other'),
}

const SEVERITY_META: Record<MageSeverity, { label: string; color: string; bg: string }> = {
  high: { label: i18n.t('structuredReview.severity.high'), color: '#b91c1c', bg: 'rgba(185,28,28,0.08)' },
  med: { label: i18n.t('structuredReview.severity.med'), color: '#ca8a04', bg: 'rgba(234,179,8,0.06)' },
  low: { label: i18n.t('structuredReview.severity.low'), color: '#64748b', bg: 'rgba(100,116,139,0.06)' },
  info: { label: i18n.t('structuredReview.severity.info'), color: '#0ea5e9', bg: 'rgba(14,165,233,0.06)' },
}

const EVIDENCE_KIND_LABELS: Record<EvidenceTargetKind, string> = {
  warrior_message: i18n.t('structuredReview.evidenceKind.warrior_message'),
  artifact: i18n.t('structuredReview.evidenceKind.artifact'),
  check: i18n.t('structuredReview.evidenceKind.check'),
  diff_file: i18n.t('structuredReview.evidenceKind.diff_file'),
  workspace_effect: i18n.t('structuredReview.evidenceKind.workspace_effect'),
  policy_target: i18n.t('structuredReview.evidenceKind.policy_target'),
  quest: i18n.t('structuredReview.evidenceKind.quest'),
  design_doc: i18n.t('structuredReview.evidenceKind.design_doc'),
  other: i18n.t('structuredReview.evidenceKind.other'),
}

type Props = {
  sr: StructuredReview
  onJumpToEvidence?: (ref: EvidenceRef, sr?: StructuredReview) => void
}

export default function StructuredReviewSection({ sr, onJumpToEvidence }: Props) {
  const { t } = useTranslation()
  const hasDisagreements = sr.disagreements && sr.disagreements.length > 0
  const hasRisks = sr.risks_extra && sr.risks_extra.length > 0
  const hasEndorsements = sr.endorsements && sr.endorsements.length > 0

  if (!hasDisagreements && !hasRisks && !hasEndorsements) return null

  return (
    <div className="structured-review-section" data-testid="structured-review-section">
      {/* 统计行 */}
      <div className="structured-review-stats">
        {hasDisagreements && (
          <span className="sr-stat sr-stat--disagreement">
            ✋ {t('structuredReview.disagreement')} ({sr.disagreements.length})
          </span>
        )}
        {hasEndorsements && (
          <span className="sr-stat sr-stat--endorsement">
            ✨ {t('structuredReview.endorsement')} ({sr.endorsements.length})
          </span>
        )}
        {hasRisks && (
          <span className="sr-stat sr-stat--risk">
            💣 {t('structuredReview.risk')} ({sr.risks_extra.length})
          </span>
        )}
        {sr.coverage_ratio && sr.coverage_ratio[1] > 0 && (
          <span className="sr-stat sr-stat--coverage mono">
            {t('structuredReview.coverage')} {sr.coverage_ratio[0]}/{sr.coverage_ratio[1]}
          </span>
        )}
      </div>

      {/* 异议卡片 */}
      {hasDisagreements && (
        <div className="sr-card-list sr-card-list--disagreement">
          {sr.disagreements.map((d) => (
            <DisagreementCard key={d.id} d={d} sr={sr} onJumpToEvidence={onJumpToEvidence} />
          ))}
        </div>
      )}

      {/* 亮点（compact strip，≤3） */}
      {hasEndorsements && (
        <div className="sr-endorsement-strip">
          {sr.endorsements.map((e) => (
            <EndorsementItem key={e.id} e={e} />
          ))}
        </div>
      )}

      {/* 额外风险卡片 */}
      {hasRisks && (
        <div className="sr-card-list sr-card-list--risk">
          {sr.risks_extra.map((r) => (
            <RiskCard key={r.id} r={r} sr={sr} onJumpToEvidence={onJumpToEvidence} />
          ))}
        </div>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// 异议卡片
// ---------------------------------------------------------------------------

function DisagreementCard({ d, sr, onJumpToEvidence }: { d: MageDisagreement; sr: StructuredReview; onJumpToEvidence?: (ref: EvidenceRef, sr?: StructuredReview) => void }) {
  const sev = SEVERITY_META[d.severity] || SEVERITY_META.med
  return (
    <div
      className="sr-card sr-card--disagreement"
      style={{ borderLeftColor: sev.color, background: sev.bg }}
      data-severity={d.severity}
    >
      <div className="sr-card__header">
        <span className="sr-tag sr-tag--dimension">{DIMENSION_LABELS[d.dimension] || d.dimension}</span>
        <span className="sr-tag sr-tag--severity" style={{ color: sev.color, borderColor: sev.color }}>
          {sev.label}
        </span>
        <span className="sr-tag sr-tag--target" title={`${i18n.t('structuredReview.targetPrefix')}${EVIDENCE_KIND_LABELS[d.target_kind] || d.target_kind}`}>
          {EVIDENCE_KIND_LABELS[d.target_kind] || d.target_kind}
        </span>
      </div>
      <p className="sr-card__claim">{d.claim}</p>
      {d.evidence && d.evidence.length > 0 && (
        <div className="sr-card__evidence">
          {d.evidence.map((ref, i) => (
            <EvidenceRefItem key={i} ref_={ref} sr={sr} onJump={onJumpToEvidence} />
          ))}
        </div>
      )}
      {d.suggestion && (
        <p className="sr-card__suggestion">
          <span className="sr-card__suggestion-label">{i18n.t('structuredReview.suggestion')}</span>
          {d.suggestion}
        </p>
      )}
      <AuditTag />
    </div>
  )
}

// ---------------------------------------------------------------------------
// 风险卡片
// ---------------------------------------------------------------------------

function RiskCard({ r, sr, onJumpToEvidence }: { r: MageExtraRisk; sr: StructuredReview; onJumpToEvidence?: (ref: EvidenceRef, sr?: StructuredReview) => void }) {
  const sev = SEVERITY_META[r.severity] || SEVERITY_META.med
  return (
    <div
      className="sr-card sr-card--risk"
      style={{ borderLeftColor: sev.color, background: sev.bg }}
      data-severity={r.severity}
    >
      <div className="sr-card__header">
        <span className="sr-tag sr-tag--risk-kind">{r.kind || 'risk'}</span>
        <span className="sr-tag sr-tag--severity" style={{ color: sev.color, borderColor: sev.color }}>
          {sev.label}
        </span>
      </div>
      <p className="sr-card__claim">{r.detail}</p>
      {r.evidence && r.evidence.length > 0 && (
        <div className="sr-card__evidence">
          {r.evidence.map((ref, i) => (
            <EvidenceRefItem key={i} ref_={ref} sr={sr} onJump={onJumpToEvidence} />
          ))}
        </div>
      )}
      <AuditTag />
    </div>
  )
}

// ---------------------------------------------------------------------------
// 亮点 strip（compact）
// ---------------------------------------------------------------------------

function EndorsementItem({ e }: { e: MageEndorsement }) {
  return (
    <div className="sr-endorsement-item">
      <span className="sr-tag sr-tag--dimension sr-tag--endorsement">
        {DIMENSION_LABELS[e.dimension] || e.dimension}
      </span>
      <span className="sr-endorsement-reason">{e.reason}</span>
    </div>
  )
}

// ---------------------------------------------------------------------------
// 证据引用
// ---------------------------------------------------------------------------

function EvidenceRefItem({ ref_, sr, onJump }: { ref_: EvidenceRef; sr?: StructuredReview; onJump?: (ref: EvidenceRef, sr?: StructuredReview) => void }) {
  const kindLabel = EVIDENCE_KIND_LABELS[ref_.kind] || ref_.kind
  const clickable = !!onJump
  return (
    <div
      className={['sr-evidence', clickable ? 'sr-evidence--jumpable' : null].filter(Boolean).join(' ')}
      data-evidence-kind={ref_.kind}
      role={clickable ? 'button' : undefined}
      tabIndex={clickable ? 0 : undefined}
      title={clickable ? i18n.t('structuredReview.jumpToEvidence') : undefined}
      onClick={clickable ? () => onJump!(ref_, sr) : undefined}
      onKeyDown={clickable ? (e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onJump!(ref_, sr) } } : undefined}
    >
      <span className="sr-evidence__kind mono">{kindLabel}</span>
      {ref_.id && <span className="sr-evidence__id mono" title={ref_.id}>{ref_.id}</span>}
      {(ref_.field || ref_.anchor) && (
        <span className="sr-evidence__loc mono">
          {[ref_.field, ref_.anchor].filter(Boolean).join(' · ')}
        </span>
      )}
      {clickable && <span className="sr-evidence__jump-icon" aria-hidden>↗</span>}
      {ref_.snippet && (
        <details className="sr-evidence__snippet" onClick={(e) => e.stopPropagation()}>
          <summary>{i18n.t('structuredReview.snippet')}</summary>
          <pre>{ref_.snippet}</pre>
        </details>
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// audit 标签
// ---------------------------------------------------------------------------

function AuditTag() {
  return (
    <span className="sr-audit-tag mono" title={i18n.t('structuredReview.auditTitle')}>
      {i18n.t('structuredReview.auditSource')}
    </span>
  )
}
