/**
 * Mage Review 纯函数断言（Phase 1 阻塞验收）。
 *
 * 执行方式：npm run test:mage-review
 * （package.json 里已配置：tsx src/__tests__/mage-review.test.ts）
 *
 * 不引入 vitest/jest/@testing-library/react，只用 Node 原生 assert +
 * 用例数组 for-loop 跑。20 个用例：A1-A8（selectLatestMageReview）
 * + B1-B9（mageReviewTone） + C1-C3（mageReviewState）。
 */

import assert from 'node:assert/strict'
import {
  selectLatestMageReview,
  mageReviewTone,
  mageReviewState,
} from '../domain/mageReview'

// ==========================================================================
// A 组：selectLatestMageReview（8 用例）
// ==========================================================================

type ReviewLite = {
  ts: number
  reviewed_by: string
  verdict: 'pass' | 'request_changes' | 'reject'
  score?: number
}

const A_CASES: Array<{
  name: string
  reviews: ReviewLite[]
  quest: { mage_id?: string | null }
  // 只断言 ts + source_kind；verdict/score 由调用点保证
  expected: { ts: number; kind: 'exact_adventurer' | 'legacy_role_literal' } | null
}> = [
  {
    name: 'A1 单条 exact 匹配',
    reviews: [{ ts: 100, reviewed_by: 'm-1', verdict: 'pass', score: 9 }],
    quest: { mage_id: 'm-1' },
    expected: { ts: 100, kind: 'exact_adventurer' },
  },
  {
    name: 'A2 单条 legacy 匹配',
    reviews: [{ ts: 100, reviewed_by: 'mage', verdict: 'pass', score: 8 }],
    quest: { mage_id: 'm-1' },
    expected: { ts: 100, kind: 'legacy_role_literal' },
  },
  {
    name: 'A3 exact 与 legacy 共存时优先 exact（legacy 最新也不盖掉真实评审）',
    reviews: [
      { ts: 90, reviewed_by: 'm-1', verdict: 'request_changes' },
      { ts: 200, reviewed_by: 'mage', verdict: 'pass', score: 7 },
    ],
    quest: { mage_id: 'm-1' },
    expected: { ts: 90, kind: 'exact_adventurer' },
  },
  {
    name: 'A4 无 mage_id 绑定，legacy reviewed_by 不应匹配',
    reviews: [{ ts: 100, reviewed_by: 'mage', verdict: 'pass' }],
    quest: { mage_id: '' },
    expected: null,
  },
  {
    name: 'A5 无 mage_id 绑定，exact reviewed_by 是另一 id，也不应匹配',
    reviews: [{ ts: 100, reviewed_by: 'm-1', verdict: 'pass' }],
    quest: { mage_id: '' },
    expected: null,
  },
  {
    name: 'A6 空 reviews',
    reviews: [],
    quest: { mage_id: 'm-1' },
    expected: null,
  },
  {
    name: 'A7 全部是 user/policy review，命中都不是 mage',
    reviews: [
      { ts: 100, reviewed_by: 'u-alice', verdict: 'pass' },
      { ts: 200, reviewed_by: 'policy:auto-complete', verdict: 'pass' },
    ],
    quest: { mage_id: 'm-1' },
    expected: null,
  },
  {
    name: 'A8 多轮返工，latest 优先（exact 匹配，request_changes → pass）',
    reviews: [
      { ts: 100, reviewed_by: 'm-1', verdict: 'request_changes', score: 4 },
      { ts: 200, reviewed_by: 'm-1', verdict: 'pass', score: 8 },
    ],
    quest: { mage_id: 'm-1' },
    expected: { ts: 200, kind: 'exact_adventurer' },
  },
]

// ==========================================================================
// B 组：mageReviewTone（9 用例，§5.1 查表顺序 1:1 覆盖）
// ==========================================================================

const B_CASES: Array<{
  name: string
  match: ReviewLite & { kind: 'exact' | 'legacy' } | null
  quest: { intensity?: string | null; mage_id?: string | null }
  expected_id: string
  expected_accent: string
}> = [
  {
    name: 'B1 reject → 深红',
    match: { ts: 100, reviewed_by: 'm-1', verdict: 'reject', score: 4, kind: 'exact' },
    quest: { intensity: 'standard', mage_id: 'm-1' },
    expected_id: 'reject', expected_accent: '#7f1d1d',
  },
  {
    name: 'B2 request_changes + score=6 → 红（修正 v0.2 F2 黄 bug）',
    match: { ts: 100, reviewed_by: 'mage', verdict: 'request_changes', score: 6, kind: 'legacy' },
    quest: { intensity: 'standard', mage_id: 'm-1' },
    expected_id: 'request_changes', expected_accent: '#dc2626',
  },
  {
    name: 'B3 pass + score=9 → 绿',
    match: { ts: 100, reviewed_by: 'm-1', verdict: 'pass', score: 9, kind: 'exact' },
    quest: { intensity: 'standard', mage_id: 'm-1' },
    expected_id: 'pass_high', expected_accent: '#16a34a',
  },
  {
    name: 'B4 pass + score=6 → 黄',
    match: { ts: 100, reviewed_by: 'm-1', verdict: 'pass', score: 6, kind: 'exact' },
    quest: { intensity: 'standard', mage_id: 'm-1' },
    expected_id: 'pass_mid', expected_accent: '#ca8a04',
  },
  {
    name: 'B5 pass + score=3 → 红（低分警示）',
    match: { ts: 100, reviewed_by: 'm-1', verdict: 'pass', score: 3, kind: 'exact' },
    quest: { intensity: 'standard', mage_id: 'm-1' },
    expected_id: 'pass_low', expected_accent: '#dc2626',
  },
  {
    name: 'B6 pass + score=0（未评分）→ 灰绿，NOT 红（修正 v0.2 F4 bug）',
    match: { ts: 100, reviewed_by: 'm-1', verdict: 'pass', score: 0, kind: 'exact' },
    quest: { intensity: 'standard', mage_id: 'm-1' },
    expected_id: 'pass_unscored', expected_accent: '#64748b',
  },
  {
    name: 'B7 null match + mage_id → 蓝（法师参与中）',
    match: null,
    quest: { intensity: 'standard', mage_id: 'm-1' },
    expected_id: 'reviewing', expected_accent: '#0ea5e9',
  },
  {
    name: 'B8 null match + quick → 灰',
    match: null,
    quest: { intensity: 'quick', mage_id: '' },
    expected_id: 'no_review_quick', expected_accent: '#64748b',
  },
  {
    name: 'B9 null match + 无 mage_id → 灰（no_mage）',
    match: null,
    quest: { intensity: 'standard', mage_id: '' },
    expected_id: 'no_mage', expected_accent: '#64748b',
  },
]

// ==========================================================================
// C 组：mageReviewState（3 用例，phase 身份解耦）
// ==========================================================================

const C_CASES: Array<{
  name: string
  match: unknown | null
  phases: Array<{
    phase_idx?: number
    role?: string
    class?: string
    adventurer_id?: string
    session_id?: string
  }>
  quest: { mage_id?: string | null }
  expected: { display: string; phase_idx: number; session_id?: string }
}> = [
  {
    name: 'C1 多 phase 场景，mage 在 phase=2（不是 1），role=mage',
    match: null,
    phases: [
      { phase_idx: 0, role: 'warrior', adventurer_id: 'w-1' },
      { phase_idx: 1, role: 'auditor' },
      { phase_idx: 2, role: 'mage', session_id: 's-42' },
    ],
    quest: { mage_id: 'm-1' },
    expected: { display: 'reviewing', phase_idx: 2, session_id: 's-42' },
  },
  {
    name: 'C2 role=reviewer，但 adventurer_id === mage_id（职业和身份双重匹配）',
    match: null,
    phases: [
      { phase_idx: 0, role: 'warrior', adventurer_id: 'w-1' },
      { phase_idx: 1, role: 'reviewer', adventurer_id: 'm-1', session_id: 's-9' },
    ],
    quest: { mage_id: 'm-1' },
    expected: { display: 'reviewing', phase_idx: 1, session_id: 's-9' },
  },
  {
    name: 'C3 phases 空 → fallback 到 phase=1（防御性）',
    match: null,
    phases: [],
    quest: { mage_id: 'm-1' },
    expected: { display: 'reviewing', phase_idx: 1, session_id: undefined },
  },
]

// ==========================================================================
// 执行
// ==========================================================================

let pass = 0
let total = 0

function runCase(label: string, fn: () => void) {
  total++
  try {
    fn()
    pass++
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    console.error(`FAIL · ${label}\n    ${msg}`)
  }
}

for (const c of A_CASES) {
  runCase(`A/${c.name}`, () => {
    const got = selectLatestMageReview(c.reviews, c.quest)
    if (c.expected === null) {
      assert.equal(got, null, `期望 null，但得到 ts=${got?.record.ts} kind=${got?.source_kind}`)
    } else {
      assert.ok(got !== null, '期望有 match，但返回 null')
      assert.equal(got!.record.ts, c.expected.ts)
      assert.equal(got!.source_kind, c.expected.kind)
    }
  })
}

for (const c of B_CASES) {
  runCase(`B/${c.name}`, () => {
    const matchLite = c.match as ReviewLite & { kind: 'exact' | 'legacy' } | null
    const match = matchLite
      ? {
          record: {
            ts: matchLite.ts,
            reviewed_by: matchLite.reviewed_by,
            verdict: matchLite.verdict,
            score: matchLite.score,
            comment: '',
          } as import('../api/types').QuestReview,
          source_kind: matchLite.kind === 'exact' ? 'exact_adventurer' : 'legacy_role_literal' as const,
        } as const
      : null
    const tone = mageReviewTone(match, c.quest)
    assert.equal(tone.id, c.expected_id, `tone.id mismatch`)
    assert.equal(tone.accent, c.expected_accent, `tone.accent mismatch for id=${tone.id}`)
  })
}

for (const c of C_CASES) {
  runCase(`C/${c.name}`, () => {
    const state = mageReviewState(c.match as any, { phases: c.phases as any, mage_id: c.quest.mage_id })
    assert.equal(state.display, c.expected.display, `display: got ${state.display} vs ${c.expected.display}`)
    assert.equal(state.phase_idx, c.expected.phase_idx, `phase_idx: got ${state.phase_idx} vs ${c.expected.phase_idx}`)
    assert.equal(state.session_id, c.expected.session_id,
      `session_id: got ${String(state.session_id)} vs ${String(c.expected.session_id)}`)
  })
}

const TOTAL = A_CASES.length + B_CASES.length + C_CASES.length
console.log(`\nmage-review tests: PASSED ${pass} / ${TOTAL}`)
if (pass !== TOTAL) process.exit(1)
