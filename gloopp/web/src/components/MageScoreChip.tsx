/**
 * MageScoreChip — 法师评分 × verdict 小胶囊（Phase 1）。
 *
 * 复用于：侧边栏法师信息行、MageReviewBand 折叠态标题、Quest 列表摘要卡（未来）。
 * 渲染逻辑完全由 mageReviewTone（domain 纯函数）查表得到，不含任何语义判断。
 *
 * 输入：
 *   - score: 0 表示未评分，显示 "— / 10"
 *   - verdict: 若缺省，胶囊只显示 score
 *   - sourceLegacy: true → 显示 "legacy" 小字 tag（ReviewedBy="mage" 写法）
 *   - size: 'sm'（侧边栏）/ 'md'（MageReviewBand 标题）
 */

import type { Verdict } from '../api/types'
import { mageReviewTone, verdictLabel, type LatestMageReviewMatch } from '../domain/mageReview'
import i18n from '../i18n'

type Props = {
  score?: number | null
  verdict?: Verdict | string | null
  sourceLegacy?: boolean
  size?: 'sm' | 'md'
  className?: string
  /** 若提供完整 match，优先从它查表。 */
  match?: LatestMageReviewMatch | null
  mageId?: string | null
  intensity?: string | null
}

export default function MageScoreChip({
  score, verdict, sourceLegacy, size = 'sm', className, match, mageId, intensity,
}: Props) {
  // 允许调用方传 match，也允许传零散字段（方便侧边栏独立用法只拿到 score/verdict）。
  const effectiveMatch: LatestMageReviewMatch | null = match ?? (verdict
    ? {
        record: {
          ts: 0,
          reviewed_by: sourceLegacy ? 'mage' : (mageId || ''),
          verdict: verdict as Verdict,
          comment: '',
          score: typeof score === 'number' ? score : undefined,
        },
        source_kind: sourceLegacy ? 'legacy_role_literal' : 'exact_adventurer',
      }
    : null)

  const tone = mageReviewTone(effectiveMatch, { mage_id: mageId ?? null, intensity: intensity ?? null })

  const scoreText = effectiveMatch?.record
    ? (effectiveMatch.record.score && effectiveMatch.record.score > 0
        ? `${effectiveMatch.record.score} / 10`
        : i18n.t('mageScoreChip.unscored'))
    : (typeof score === 'number' && score > 0 ? `${score} / 10` : '— / 10')

  const verdictText = effectiveMatch?.record?.verdict
    ? verdictLabel(effectiveMatch.record.verdict as Verdict)
    : (verdict ? verdictLabel(verdict as Verdict) : null)

  return (
    <span
      className={[
        'mage-score-chip',
        `mage-score-chip--${size}`,
        `tone--${tone.id}`,
        className || '',
      ].filter(Boolean).join(' ')}
      style={{ borderColor: tone.accent, background: tone.background }}
      data-testid="mage-score-chip"
      data-tone={tone.id}
    >
      <span className="mage-score-chip__score mono" style={{ color: tone.accent }}>
        ⭐ {scoreText}
      </span>
      {verdictText && (
        <span className="mage-score-chip__verdict" title={verdictText}>
          <i className="mage-score-chip__dot" style={{ background: tone.accent }} />
          {verdictText}
        </span>
      )}
      {sourceLegacy && (
        <span
          className="mage-score-chip__tag"
          title={i18n.t('mageScoreChip.legacyTitle')}
        >
          legacy
        </span>
      )}
    </span>
  )
}
