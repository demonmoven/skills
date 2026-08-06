/**
 * FinalVerdictCard — 终态决策卡（Phase 1）。
 *
 * 展示 quest.final_verdict / final_comment / finalized_by / auto_passed_by_policy /
 * auto_completed_by_policy。这张卡的定位是 "当前 Quest 的真值结论"。
 * Mage Review（法师评审卡）是独立的"过程信号"，和它是两条线，互不推断。
 *
 * 原则（见 mage-review-spec v0.2.2 §0.1）：
 *   - finalized_by 只能是 user / policy / automation，**绝不可能是 mage**
 *   - 法师 verdict 绝不写进 final_*；终态决策是人类/策略的责任
 */

import type { CSSProperties, ReactNode } from 'react'
import Icon from './Icon'
import MarkdownRenderer from './MarkdownRenderer'
import type { FinalizedBy, QuestMeta, Verdict } from '../api/types'
import i18n from '../i18n'
import { useTranslation } from 'react-i18next'

type Props = {
  verdict?: Verdict | string | null
  comment?: string | null
  finalized_by?: FinalizedBy | null
  auto_passed_by_policy?: string | null
  auto_completed_by_policy?: string | null
  className?: string
  style?: CSSProperties
}

type FinalizedTone = {
  accent: string
  background: string
  chipClass: string  // 复用 styles.css 里的 chip 色板（ok/violet/amber/red）
  label: string      // "通过 / 要求返工 / 否决"
}

function finalizedTone(verdict: string | null | undefined): FinalizedTone {
  switch (verdict) {
    case 'pass':            return { accent: '#16a34a', background: 'rgba(22,163,74,0.06)', chipClass: 'chip-ok',     label: i18n.t('finalVerdict.verdict.pass') }
    case 'request_changes': return { accent: '#ea580c', background: 'rgba(234,88,12,0.08)',  chipClass: 'chip-amber',  label: i18n.t('finalVerdict.verdict.requestChanges') }
    case 'reject':          return { accent: '#b91c1c', background: 'rgba(185,28,28,0.08)',  chipClass: 'chip-red',    label: i18n.t('finalVerdict.verdict.reject') }
    default:                return { accent: '#64748b', background: 'rgba(100,116,139,0.06)',chipClass: 'chip-gray',   label: i18n.t('finalVerdict.verdict.handled') }
  }
}

function actorLabel(
  finalized_by: FinalizedBy | null | undefined,
  auto_passed: string | null | undefined,
  auto_completed: string | null | undefined,
): { icon: string; text: ReactNode } {
  if (finalized_by === 'user')       return { icon: 'user',          text: i18n.t('finalVerdict.actor.user') }
  if (finalized_by === 'automation') return { icon: 'zap',           text: i18n.t('finalVerdict.actor.automation') }
  if (finalized_by === 'policy') {
    const pname = auto_passed || auto_completed
    return {
      icon: 'shield',
      text: pname ? <>{i18n.t('finalVerdict.actor.policyAuto')}<span className="mono">{pname}</span></> : i18n.t('finalVerdict.actor.policyDecision'),
    }
  }
  return { icon: 'circle-check', text: finalized_by ? i18n.t('finalVerdict.actor.source', { source: finalized_by }) : i18n.t('finalVerdict.actor.terminal') }
}

export default function FinalVerdictCard({
  verdict,
  comment,
  finalized_by,
  auto_passed_by_policy,
  auto_completed_by_policy,
  className,
  style,
}: Props) {
  const { t } = useTranslation()
  // 没有终态（finalized_by 空）= Quest 尚未走完，不渲染这张卡。
  // 调用方也可以直接用 !!quest.finalized_by 来条件挂载。
  if (!finalized_by && !verdict) return null

  const tone = finalizedTone(verdict || null)
  const actor = actorLabel(finalized_by, auto_passed_by_policy, auto_completed_by_policy)
  const body = String(comment || '').trim()

  return (
    <section
      className={['panel', 'final-verdict-card', className].filter(Boolean).join(' ')}
      style={{
        ...style,
        borderColor: tone.accent,
        background: tone.background,
      }}
      data-testid="final-verdict-card"
      data-finalized-by={finalized_by || ''}
      data-verdict={verdict || ''}
    >
      <div className="panel-title final-verdict-head">
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, minWidth: 0 }}>
          <span style={{ display: 'inline-flex', color: tone.accent }} aria-hidden>
            <Icon name={actor.icon} size={15} />
          </span>
          <h2 style={{ margin: 0 }}>
            {t('finalVerdict.title')}
            <span className="mono faint" style={{ marginLeft: 8, fontSize: 12 }}>
              {actor.text}
            </span>
          </h2>
        </div>
        <span className={['verdict-chip', tone.chipClass].join(' ')}>
          <span className="verdict-chip-dot" style={{ background: tone.accent }} />
          {tone.label}
        </span>
      </div>

      {body && finalized_by !== 'user' ? (
        <MarkdownRenderer source={body} className="compact final-verdict-comment" stripFrontmatter />
      ) : finalized_by === 'user' ? (
        <p className="faint final-verdict-empty">{t('finalVerdict.userDone')}</p>
      ) : (
        <p className="faint final-verdict-empty">{t('finalVerdict.noComment')}</p>
      )}
    </section>
  )
}

// 小工具：从 QuestMeta 里把相关字段一次性抽出来，减少调用方样板代码。
export function finalVerdictPropsFromMeta(q: Pick<QuestMeta, 'final_verdict' | 'final_comment' | 'finalized_by' | 'auto_passed_by_policy' | 'auto_completed_by_policy'>): Props {
  return {
    verdict: q.final_verdict ?? null,
    comment: q.final_comment ?? null,
    finalized_by: q.finalized_by ?? null,
    auto_passed_by_policy: q.auto_passed_by_policy ?? null,
    auto_completed_by_policy: q.auto_completed_by_policy ?? null,
  }
}
