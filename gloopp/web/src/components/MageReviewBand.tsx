/**
 * MageReviewBand — 法师评审带（Phase 1）。
 *
 * 定位（见 mage-review-spec v0.2.2 §5）：
 *   - 永远在剑士结论卡下方（视觉层级低于剑士）
 *   - 法师是"第二只眼 / 抗偏差层"，不替代剑士、不替代用户终审
 *   - 不做任何语义判断，所有颜色/文案来自 mageReviewTone 查表
 *
 * 四种展示态（由 mageReviewState 决定）：
 *   reviewed        → 完整评审卡（折叠/展开）
 *   reviewing       → 横幅 "法师暂未提交评审"
 *   no_mage         → 横幅 "未配置法师 · 缺少抗偏差层"
 *   no_review_quick → 横幅 "快速模式 · 无独立评审"
 */

import { useState } from 'react'
import type { CSSProperties } from 'react'
import Icon from './Icon'
import MarkdownRenderer from './MarkdownRenderer'
import MageScoreChip from './MageScoreChip'
import StructuredReviewSection from './StructuredReviewSection'
import { mageReviewTone, mageReviewState, type LatestMageReviewMatch } from '../domain/mageReview'
import type { QuestMeta, EvidenceRef, StructuredReview } from '../api/types'
import { useTranslation } from 'react-i18next'

type Props = {
  match: LatestMageReviewMatch | null
  quest: Pick<QuestMeta, 'mage_id' | 'intensity' | 'phases'>
  onJumpToTrace?: () => void
  onJumpToEvidence?: (ref: EvidenceRef, sr?: StructuredReview) => void
  className?: string
  style?: CSSProperties
}

export default function MageReviewBand({ match, quest, onJumpToTrace, onJumpToEvidence, className, style }: Props) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const tone = mageReviewTone(match, quest)
  const state = mageReviewState(match, quest)

  // —— 降级态：阶段内轻量状态，不展开 ——
  if (state.display !== 'reviewed') {
    return (
      <section
        className={['mage-review-band', 'mage-review-band--inline', `tone--${tone.id}`, 'is-' + state.display, className].filter(Boolean).join(' ')}
        style={style}
        data-testid="mage-review-band"
        data-display={state.display}
      >
        <div className="mage-review-band__inline-status" style={{ color: tone.accent }}>
          <Icon name={state.display === 'no_mage' ? 'shield-alert' : state.display === 'reviewing' ? 'clock' : 'zap'} size={13} />
          <span>{tone.banner_text}</span>
        </div>
      </section>
    )
  }

  // —— reviewed 态：完整评审卡 ——
  const rec = match!.record
  const score = typeof rec.score === 'number' ? rec.score : 0
  const repairCount = typeof rec.repair_count === 'number' ? rec.repair_count : null
  const totalIssues = typeof rec.total_issue_count === 'number' ? rec.total_issue_count : null
  const hints = String(rec.rewrite_hints || '').trim()
  const comment = String(rec.comment || '').trim()
  const isLegacy = match!.source_kind === 'legacy_role_literal'
  const sourceIncomplete = !!rec.review_source?.source_incomplete

  const metaParts: string[] = []
  if (rec.ts) metaParts.push(new Date(rec.ts).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }))
  if (repairCount !== null && totalIssues !== null) metaParts.push(t('mageReview.repairStats', { repaired: repairCount, total: totalIssues }))
  else if (totalIssues !== null) metaParts.push(t('mageReview.issueStats', { total: totalIssues }))

  return (
    <section
      className={['panel', 'mage-review-band', `tone--${tone.id}`, className].filter(Boolean).join(' ')}
      style={{ ...style, borderColor: tone.accent, background: tone.background }}
      data-testid="mage-review-band"
      data-display="reviewed"
      data-verdict={rec.verdict}
    >
      {/* 横幅（仅 warn / danger 级别展示） */}
      {tone.banner_text && tone.banner_level !== 'none' && tone.banner_level !== 'info' && (
        <div className={`adventurer-banner level-${tone.banner_level}`} style={{ borderLeftColor: tone.accent }}>
          <Icon name="alert-triangle" size={14} />
          <span>{tone.banner_text}</span>
        </div>
      )}

      <div className="mage-review-band__head" onClick={() => setExpanded(v => !v)} role="button" tabIndex={0}>
        <div className="mage-review-band__title">
          <span style={{ display: 'inline-flex', color: tone.accent }} aria-hidden>
            <Icon name="eye" size={15} />
          </span>
          <span className="mage-review-band__label">{t('mageReview.label')}</span>
          <MageScoreChip
            score={score > 0 ? score : undefined}
            verdict={rec.verdict}
            sourceLegacy={isLegacy}
            size="md"
            match={match}
            mageId={quest.mage_id}
            intensity={quest.intensity}
          />
        </div>
        <div className="mage-review-band__meta">
          {metaParts.length > 0 && <span className="mono faint">{metaParts.join(' · ')}</span>}
          <Icon name={expanded ? 'chevron-down' : 'chevron-right'} size={14} className="mage-review-band__chevron" />
        </div>
      </div>

      {sourceIncomplete && (
        <div className="mage-review-band__source-incomplete" title={t('mageReview.sourceIncompleteTitle')}>
          <Icon name="alert-triangle" size={12} />
          <span>{t('mageReview.sourceIncomplete')}</span>
        </div>
      )}

      {expanded && (
        <div className="mage-review-band__body">
          {rec.structured_review && (
            <StructuredReviewSection
              sr={rec.structured_review}
              onJumpToEvidence={onJumpToEvidence}
            />
          )}
          {rec.structured_review?.free_comment_range && comment && (
            <p className="mage-review-band__comment-range mono">
              {t('mageReview.commentRange', { start: rec.structured_review.free_comment_range.start_char, end: rec.structured_review.free_comment_range.end_char })}
            </p>
          )}
          {comment ? (
            <MarkdownRenderer source={comment} className="compact mage-review-comment" stripFrontmatter />
          ) : (
            <p className="faint">{t('mageReview.noComment')}</p>
          )}

          {hints && (
            <details className="mage-review-band__hints">
              <summary>{t('mageReview.hintsSummary')}</summary>
              <pre className="mage-review-hints-text">{hints}</pre>
            </details>
          )}

          {onJumpToTrace && (
            <div className="mage-review-band__actions">
              <button type="button" className="link-button tiny" onClick={onJumpToTrace}>
                <Icon name="search" size={12} /> {t('mageReview.viewProcess')}
              </button>
            </div>
          )}
        </div>
      )}
    </section>
  )
}
