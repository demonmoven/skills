/**
 * Mage Review 纯函数层（Phase 1 · 零后端改动）。
 *
 * 本文件只做三件事：
 *   1) selectLatestMageReview — 从 reviews[] 里挑出“法师提交的最新那条”
 *      （兼容 ReviewedBy === mage_id 精确匹配 和 ReviewedBy === "mage" legacy 写法；
 *       mage_id 为空时不信任 legacy 写法）
 *   2) mageReviewTone — 根据 match + quest 查表返回 UI 色板 / 横幅文案
 *      （查表顺序唯一，无歧义，见 mage-review-spec v0.2.2 §5.1）
 *   3) mageReviewState — 返回展示态和 ExecutionTrace 跳转目标
 *      （从 quest.phases 匹配 mage 的 role/class/adventurer_id，不 hardcode phase=1）
 *
 * 严格遵守 Gloop PRD v2「系统做机制，agent 做决策」：
 *   - 不做任何语义判断（disagreement / risk / dimension / severity 全不碰）
 *   - 所有输入字段都来自白名单：quest.mage_id / quest.intensity /
 *     reviews[].reviewed_by / phases[].role|class|adventurer_id …
 *   - Phase 1.5 引入 typed ReviewSource 后，这里只会在优先路径上用它，
 *     不会做字段推断。
 *
 * 纯函数，可独立测试（见 web/src/__tests__/mage-review.test.ts）。
 */

import type { QuestPhaseTask, QuestReview, Verdict } from '../api/types'
import i18n from '../i18n'

// ---------------------------------------------------------------------------
// 类型
// ---------------------------------------------------------------------------

export type MageSourceKind = 'exact_adventurer' | 'legacy_role_literal'

export interface LatestMageReviewMatch {
  record: QuestReview
  source_kind: MageSourceKind
}

export type ToneId =
  | 'reject'                      // verdict === reject
  | 'request_changes'             // verdict === request_changes
  | 'pass_high'                   // pass + score >= 8
  | 'pass_mid'                    // pass + 5 <= score < 8
  | 'pass_low'                    // pass + 0 < score < 5
  | 'pass_unscored'               // pass + score === 0 / undefined
  | 'reviewing'                   // 有 mage_id 但尚未提交评审
  | 'no_mage'                     // 未配置法师
  | 'no_review_quick'             // quick 模式，跳过法师评审

export interface Tone {
  id: ToneId
  /** 适合放在剑士卡边框 / MageScoreChip 上的颜色（CSS 变量或 hex）。 */
  accent: string
  /** 背景轻染色（rgba）。 */
  background: string
  /** 剑士卡顶部横幅文案。空字符串 = 不展示横幅。 */
  banner_text: string
  /** banner tone 的语义等级（用于样式 class 后缀）。 */
  banner_level: 'none' | 'info' | 'warn' | 'danger'
}

export type DisplayState =
  | 'reviewed'                    // 法师已评审（LatestMageReview 非空）
  | 'reviewing'                   // 法师参与中（mage_id 存在 + 无 review）
  | 'no_mage'                     // 未配置法师
  | 'no_review_quick'             // quick 模式

export interface ReviewDisplayState {
  display: DisplayState
  /** ExecutionTrace 跳转目标。phase_idx 永远合法（缺省 fallback 到 1）。 */
  phase_idx: number
  /** phase 对应的 session id（存在就用它跳，不存在按 phase_idx 回退）。 */
  session_id?: string
  /** 展示给人看的 mage 名字占位符；纯前端用，Phase 1 保持空串由调用方拼。 */
  mage_label_hint?: string
}

// ---------------------------------------------------------------------------
// 1) selectLatestMageReview
// ---------------------------------------------------------------------------

export function selectLatestMageReview(
  reviews: Array<Pick<QuestReview, 'ts' | 'reviewed_by' | 'review_source'>>,
  quest: { mage_id?: string | null },
): LatestMageReviewMatch | null {
  const mageId = String(quest.mage_id ?? '').trim()

  const hasBinding = mageId.length > 0
  if (!reviews || reviews.length === 0) return null

  // Phase 1.5 如果有 typed review_source，优先走它（不受 legacy ReviewedBy 写法影响）。
  type Candidate = { rec: typeof reviews[number]; kind: MageSourceKind; ts: number }
  const candidates: Candidate[] = []

  for (const r of reviews) {
    const ts = typeof r.ts === 'number' ? r.ts : Number(r.ts) || 0
    const src = r.review_source

    // 1.5 typed source 路径：明确是 mage actor 就拿。
    if (src && (src.source_role === 'mage' || src.source_class === 'mage' ||
               (hasBinding && src.source_adventurer_id === mageId))) {
      candidates.push({
        rec: r,
        kind: (src.source_adventurer_id && hasBinding && src.source_adventurer_id === mageId)
          ? 'exact_adventurer'
          : 'legacy_role_literal',
        ts,
      })
      continue
    }

    // Phase 1 兼容路径：ReviewedBy。
    const rb = String(r.reviewed_by ?? '')
    if (hasBinding && rb === mageId) {
      candidates.push({ rec: r, kind: 'exact_adventurer', ts })
      continue
    }
    if (hasBinding && rb === 'mage') {
      // domain service 历史写法：只有在有 mage_id 绑定的情况下才信任，
      // mage_id 为空时（比如系统级自动评审）绝不误判。
      candidates.push({ rec: r, kind: 'legacy_role_literal', ts })
      continue
    }
  }

  if (candidates.length === 0) return null

  // 有 exact_adventurer 评审时，legacy 记录是噪音（旧写法/溯源不全），不参与竞争。
  // 否则旧 quest 的 legacy 记录会因 ts 更大盖掉真实评审（如 4386：score=8 被未评分 legacy 盖掉）。
  const hasExact = candidates.some((c) => c.kind === 'exact_adventurer')
  const filtered = hasExact ? candidates.filter((c) => c.kind === 'exact_adventurer') : candidates

  // max(Ts) 优先。相同 Ts 时 exact_adventurer > legacy（极少发生，但要稳定）。
  filtered.sort((a, b) => {
    if (b.ts !== a.ts) return b.ts - a.ts
    return rank(a.kind) - rank(b.kind)
  })
  function rank(k: MageSourceKind) { return k === 'exact_adventurer' ? 0 : 1 }

  const top = filtered[0]
  return { record: top.rec as LatestMageReviewMatch['record'], source_kind: top.kind }
}

// ---------------------------------------------------------------------------
// 2) mageReviewTone（查表顺序唯一 · 不可调换前后顺序）
// ---------------------------------------------------------------------------

export function mageReviewTone(
  match: LatestMageReviewMatch | null,
  quest: { intensity?: string | null; mage_id?: string | null },
): Tone {
  const mageId = String(quest.mage_id ?? '').trim()
  const intensity = String(quest.intensity ?? '').toLowerCase()

  // —— 先处理降级态（没有 LatestMageReview 的情况）——
  if (!match) {
    if (intensity === 'quick') {
      return toneOf('no_review_quick')
    }
    if (!mageId) {
      return toneOf('no_mage')
    }
    // 有 mage_id 但没 review → 法师参与中
    return toneOf('reviewing')
  }

  // —— 有 match 时按 verdict × score 从上到下唯一查表 ——
  const verdict = match.record.verdict as Verdict
  const score = typeof match.record.score === 'number' ? match.record.score : 0

  if (verdict === 'reject')                     return toneOf('reject')
  if (verdict === 'request_changes')            return toneOf('request_changes')
  // pass: unscored 优先，避免 score=0 落到 score<5 分支
  const s = typeof score === 'number' ? score : 0
  if (s === 0)                                           return toneOf('pass_unscored')
  if (s < 5)                                             return toneOf('pass_low')
  if (s < 8)                                             return toneOf('pass_mid')
  /* score >= 8 */                                       return toneOf('pass_high')
}

// ---------------------------------------------------------------------------
// 3) mageReviewState
// ---------------------------------------------------------------------------

export function mageReviewState(
  match: LatestMageReviewMatch | null,
  quest: { phases?: QuestPhaseTask[] | null; mage_id?: string | null; intensity?: string | null },
): ReviewDisplayState {
  const intensity = String(quest.intensity ?? '').toLowerCase()
  const mageId = String(quest.mage_id ?? '').trim()

  let display: DisplayState
  if (match)                                                  display = 'reviewed'
  else if (intensity === 'quick')                              display = 'no_review_quick'
  else if (!mageId)                                            display = 'no_mage'
  else                                                         display = 'reviewing'

  const phases = Array.isArray(quest.phases) ? quest.phases : []

  // Phase 1.5: 如有 match.record.review_source，优先用它的 phase_idx/session_id
  const rs = match?.record.review_source
  if (rs && typeof rs.source_phase_idx === 'number') {
    return {
      display,
      phase_idx: rs.source_phase_idx,
      session_id: rs.source_session_id,
    }
  }

  // 从 quest.phases 里匹配 mage（role / class / adventurer_id）。
  // 多 phase pipeline 下 phase_idx 不一定是 1，角色匹配更稳。
  const found = phases.find((p) => {
    const role = String(p.role ?? '').toLowerCase()
    const cls  = String(p.class ?? '').toLowerCase()
    if (role === 'mage' || cls === 'mage') return true
    if (mageId && p.adventurer_id && p.adventurer_id === mageId) return true
    return false
  })
  if (found && typeof found.phase_idx === 'number' && Number.isFinite(found.phase_idx)) {
    return {
      display,
      phase_idx: found.phase_idx,
      session_id: found.session_id,
    }
  }

  // 防御性 fallback：老 quest 可能没有 phases[]，回退到 1。
  return { display, phase_idx: 1, session_id: undefined }
}

// ---------------------------------------------------------------------------
// 色板表（对应 mage-review-spec v0.2.2 §5.1）
// ---------------------------------------------------------------------------

const TONE_TABLE: Record<ToneId, Tone> = {
  reject: {
    id: 'reject',
    accent: '#7f1d1d',
    background: 'rgba(127,29,29,0.10)',
    banner_text: i18n.t('mageBanner.reject'),
    banner_level: 'danger',
  },
  request_changes: {
    id: 'request_changes',
    accent: '#dc2626',
    background: 'rgba(239,68,68,0.08)',
    banner_text: i18n.t('mageBanner.requestChanges'),
    banner_level: 'danger',
  },
  pass_low: {
    id: 'pass_low',
    accent: '#dc2626',
    background: 'rgba(239,68,68,0.08)',
    banner_text: i18n.t('mageBanner.lowScore'),
    banner_level: 'danger',
  },
  pass_mid: {
    id: 'pass_mid',
    accent: '#ca8a04',
    background: 'rgba(234,179,8,0.06)',
    banner_text: i18n.t('mageBanner.conditional'),
    banner_level: 'warn',
  },
  pass_high: {
    id: 'pass_high',
    accent: '#16a34a',
    background: 'rgba(34,197,94,0.06)',
    banner_text: '',
    banner_level: 'none',
  },
  pass_unscored: {
    id: 'pass_unscored',
    accent: '#64748b',
    background: 'rgba(100,116,139,0.06)',
    banner_text: i18n.t('mageBanner.passUnscored'),
    banner_level: 'info',
  },
  reviewing: {
    id: 'reviewing',
    accent: '#0ea5e9',
    background: 'rgba(14,165,233,0.06)',
    banner_text: i18n.t('mageBanner.reviewing'),
    banner_level: 'info',
  },
  no_mage: {
    id: 'no_mage',
    accent: '#64748b',
    background: 'rgba(100,116,139,0.06)',
    banner_text: i18n.t('mageBanner.noMage'),
    banner_level: 'warn',
  },
  no_review_quick: {
    id: 'no_review_quick',
    accent: '#64748b',
    background: 'rgba(100,116,139,0.06)',
    banner_text: i18n.t('mageBanner.quickMode'),
    banner_level: 'warn',
  },
}

function toneOf(id: ToneId): Tone { return TONE_TABLE[id] }

// 辅助：把 tone 转成 class name，方便 CSS 匹配。
export function toneClassName(prefix: string, tone: Tone): string {
  return `${prefix}-${tone.id}`
}

export function verdictLabel(v: Verdict): string {
  switch (v) {
    case 'pass': return i18n.t('verdict.pass')
    case 'request_changes': return i18n.t('verdict.requestChanges')
    case 'reject': return i18n.t('verdict.reject')
    default: return String(v)
  }
}
