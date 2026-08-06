/**
 * MageReviewFixtures — 开发态 fixture 页（Phase 1）。
 *
 * 仅在 DEV 模式 + ?magereview-fixtures=1 查询参数下激活。
 * 生产构建时，本文件被 tree-shake 移除（import.meta.env.DEV 为 false）。
 *
 * 6 个 fixture 覆盖 mageReviewTone 的全部 9 种 tone 里的关键路径：
 *   F1 pass_high          → pass + score=9 → 绿
 *   F2 request_changes_mid → request_changes + score=6, legacy → 红
 *   F3 reject_low         → reject + score=3 → 深红
 *   F4 pass_unscored      → pass + score=0 → 灰绿
 *   F5 no_mage            → 无 mage_id, 无 review → 灰
 *   F6 quick_no_review    → intensity=quick, 无 review → 灰
 */

import type { QuestReview, EvidenceRef, StructuredReview } from '../api/types'
import MageReviewBand from '../components/MageReviewBand'
import { selectLatestMageReview, type LatestMageReviewMatch } from '../domain/mageReview'

// —— Fixture 数据构造 ——
// 每条 review 的 ts 从 1000 递增，确保 selectLatestMageReview 取到正确的 latest。

function mkReview(over: Partial<QuestReview>): QuestReview {
  return {
    ts: 1000,
    verdict: 'pass',
    comment: '法师评审评语。',
    reviewed_by: '',
    ...over,
  }
}

const F1_REVIEW = mkReview({
  ts: 1100,
  verdict: 'pass',
  score: 9,
  comment: '代码质量高，逻辑清晰，测试覆盖充分。',
  repair_count: 0,
  total_issue_count: 0,
  reviewed_by: 'm-zenith',
})

// F7: structured review typed 卡片展示（Phase 2）
const F7_REVIEW = mkReview({
  ts: 1700,
  verdict: 'request_changes',
  score: 6,
  comment: '整体方向正确，但有以下异议需要处理。详见下方 typed 卡片。',
  repair_count: 2,
  total_issue_count: 3,
  reviewed_by: 'm-structured',
  structured_review: {
    schema_version: 'v1',
    reviewed_warrior_phase_idx: 0,
    reviewed_warrior_session_id: 'sess-fixture-001',
    reviewed_warrior_latest_turn: 4,
    coverage_ratio: [7, 9],
    confidence: 'med',
    free_comment_range: { start_char: 0, end_char: 12 },
    disagreements: [
      {
        id: 'd1',
        dimension: 'correctness',
        severity: 'high',
        target_kind: 'diff_file',
        claim: '正则边界缺失：WebRequest.id 的匹配在 UUID v7 + 多字节字符下会漏匹配',
        evidence: [
          { id: 'web/src/api/types.ts:42', kind: 'diff_file', field: 'line 42', snippet: 'const ID_RE = /[a-f0-9-]+/ // 缺少 {36} 长度约束和 anchors' },
        ],
        suggestion: '补 id 边界字符类 + 加 fuzz 测试用例',
      },
      {
        id: 'd2',
        dimension: 'boundary',
        severity: 'med',
        target_kind: 'artifact',
        claim: '空数组输入未处理，会导致 reduce 报错',
        evidence: [
          { id: 'art-output-json', kind: 'artifact', anchor: 'section-2.3' },
        ],
      },
      {
        id: 'd3',
        dimension: 'coherence',
        severity: 'low',
        target_kind: 'warrior_message',
        claim: '战士 turn 4 回复中遗漏了空数组的处理说明',
        evidence: [
          { id: 'msg-turn-4', kind: 'warrior_message' },
        ],
      },
    ],
    risks_extra: [
      {
        id: 'r1',
        kind: 'regression',
        severity: 'low',
        detail: '移除了 retry 中间件，线上已有依赖该行为的调用方可能受影响',
        evidence: [],
      },
    ],
    endorsements: [
      {
        id: 'e1',
        dimension: 'completeness',
        reason: '错误处理链路完整，覆盖了超时/重试/熔断',
        evidence: [],
      },
    ],
  } as any,
})

const F2_REVIEW = mkReview({
  ts: 1200,
  verdict: 'request_changes',
  score: 6,
  comment: '整体方向正确，但有三处需要返工：\n1. 缺少错误处理\n2. 边界条件未覆盖\n3. 命名不一致',
  repair_count: 3,
  total_issue_count: 5,
  reviewed_by: 'mage', // legacy 写法
})

const F3_REVIEW = mkReview({
  ts: 1300,
  verdict: 'reject',
  score: 3,
  comment: '方案存在根本性问题：与 PRD 核心原则冲突，系统不应做语义判断。',
  repair_count: 0,
  total_issue_count: 4,
  reviewed_by: 'm-strict',
})

const F4_REVIEW = mkReview({
  ts: 1400,
  verdict: 'pass',
  score: 0, // 未评分
  comment: '法师通过，但未提交评分。',
  reviewed_by: 'm-casual',
})

// F5 / F6 没有 review。

const FIXTURES: Array<{
  id: string
  label: string
  reviews: QuestReview[]
  quest: { mage_id?: string | null; intensity?: string | null; phases?: any[] }
}> = [
  {
    id: 'F1',
    label: 'pass_high · pass + score=9 → 绿',
    reviews: [F1_REVIEW],
    quest: { mage_id: 'm-zenith', intensity: 'standard' },
  },
  {
    id: 'F2',
    label: 'request_changes_mid · request_changes + score=6, legacy → 红',
    reviews: [F2_REVIEW],
    quest: { mage_id: 'm-1', intensity: 'standard' },
  },
  {
    id: 'F3',
    label: 'reject_low · reject + score=3 → 深红',
    reviews: [F3_REVIEW],
    quest: { mage_id: 'm-strict', intensity: 'standard' },
  },
  {
    id: 'F4',
    label: 'pass_unscored · pass + score=0 → 灰绿（非红）',
    reviews: [F4_REVIEW],
    quest: { mage_id: 'm-casual', intensity: 'standard' },
  },
  {
    id: 'F5',
    label: 'no_mage · 无 mage_id → 灰',
    reviews: [],
    quest: { mage_id: '', intensity: 'standard' },
  },
  {
    id: 'F6',
    label: 'quick_no_review · intensity=quick → 灰',
    reviews: [],
    quest: { mage_id: '', intensity: 'quick' },
  },
  {
    id: 'F7',
    label: 'structured_review · typed 卡片（异议+风险+亮点）',
    reviews: [F7_REVIEW],
    quest: { mage_id: 'm-structured', intensity: 'standard' },
  },
]

export default function MageReviewFixtures() {
  return (
    <div style={{ maxWidth: 900, margin: '0 auto', padding: '24px 16px', display: 'flex', flexDirection: 'column', gap: 24 }}>
      <header>
        <h1 style={{ fontSize: 20, margin: '0 0 4px' }}>Mage Review Fixtures</h1>
        <p style={{ fontSize: 13, color: '#64748b', margin: 0 }}>
          Phase 1 开发态验收页。6 个 fixture 覆盖 mageReviewTone 全部关键路径。
          生产构建不包含此页。
        </p>
      </header>

      {FIXTURES.map((f) => {
        const match: LatestMageReviewMatch | null = selectLatestMageReview(
          f.reviews,
          { mage_id: f.quest.mage_id ?? null },
        )
        return (
          <div key={f.id} style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            <div style={{ fontSize: 12, fontWeight: 600, color: '#64748b' }}>
              <span className="mono">{f.id}</span> · {f.label}
            </div>
            <MageReviewBand
              match={match}
              quest={f.quest as any}
              onJumpToEvidence={(ref: EvidenceRef, sr?: StructuredReview) => {
                // fixture 无真实 trace/artifact，仅演示可点击态
                // eslint-disable-next-line no-console
                console.info('[fixture] jump to evidence', ref.kind, ref.id, sr?.reviewed_warrior_session_id)
              }}
            />
          </div>
        )
      })}
    </div>
  )
}
