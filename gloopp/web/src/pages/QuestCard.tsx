import type { QuestMeta } from '../api/types'
import type { FieldKey } from '../components/FieldsPopover'
import Icon from '../components/Icon'
import QuestSummary from '../components/QuestSummary'
import { fmtBlockedReason, fmtDuration, hasApplyFailed, relativeTime, shortId, typeLabel } from '../components/util'
import i18n from '../i18n'
import { useTranslation } from 'react-i18next'
import {
  artifactKindCounts,
  deriveQuestAttention,
  effectTypeInfo,
  phaseDisplayName,
  policyChips,
  questDurationUsageLabel,
  questPhaseView,
  questTurnUsageLabel,
  questWaitingInputAgeMs,
} from '../domain/questSelectors'
import { getStuckDiagnosis } from '../features/quest/stuckDiagnosisMetrics'
import { failureSummary, finalizedByLabel, truncateText, questActivityTime } from './questsBoardHelpers'

type SignalChip = {
  key: string
  icon?: string
  label: string
  tone: string
  title?: string
}

const MAX_SIGNAL_CHIPS = 3

/**
 * 收敛 signal chip 至 ≤3，避免徽章墙。
 * 优先级：风险类(amber/external) > 策略自动完成 > diff 待审核 > 产物 > 策略建议通过 > finalized_by。
 * attention 卡片优先展示风险与 diff；普通卡片优先展示 effect 与产物。
 */
function pickSignalChips(
  quest: QuestMeta,
  effectInfo: ReturnType<typeof effectTypeInfo>,
  outputCounts: ReturnType<typeof artifactKindCounts>,
  policySignals: ReturnType<typeof policyChips>,
  isDone: boolean,
): SignalChip[] {
  const candidates: SignalChip[] = []

  // 风险类 effect（external_side_effect / context_store）优先级最高
  if (effectInfo && (effectInfo.tone === 'amber' || effectInfo.tone === 'violet')) {
    candidates.push({ key: 'effect', icon: effectInfo.icon, label: effectInfo.label, tone: effectInfo.tone })
  }
  // 策略自动完成（强信号）
  if (quest.auto_completed_by_policy) {
    candidates.push({ key: 'auto_complete', icon: 'zap', label: i18n.t('questCard.signal.autoComplete'), tone: 'blue' })
  }
  // diff 待审核（attention 强相关）
  if (quest.workspace_diff_pending) {
    candidates.push({ key: 'diff', icon: 'compare', label: i18n.t('questCard.signal.diffPending'), tone: 'blue' })
  }
  // 策略建议通过
  if (quest.auto_passed_by_policy) {
    candidates.push({ key: 'auto_pass', icon: 'circle-check', label: i18n.t('questCard.signal.autoPass'), tone: 'muted' })
  }
  // policy signals（非 auto_complete 的策略决策）
  for (const chip of policySignals) {
    if (chip.key !== 'policy' || !String(chip.label || '').includes(i18n.t('quest.policy.action.auto_complete'))) {
      candidates.push({ key: 'policy-' + chip.key, icon: chip.icon, label: chip.label, tone: chip.tone, title: chip.title })
    }
  }
  // 风险类之外的 effect（workspace_diff 等 blue 类）
  if (effectInfo && effectInfo.tone === 'blue') {
    candidates.push({ key: 'effect-blue', icon: effectInfo.icon, label: effectInfo.label, tone: effectInfo.tone })
  }
  // 产物计数
  if (outputCounts.total > 0) {
    candidates.push({ key: 'outputs', icon: 'file-text', label: i18n.t('questCard.signal.outputs', { count: outputCounts.total }), tone: 'muted', title: outputCounts.labels.join(' · ') })
  }
  // finalized_by（仅 ended）
  if (quest.finalized_by && (isDone || quest.status === 'failed')) {
    candidates.push({ key: 'finalized', label: finalizedByLabel(quest.finalized_by), tone: 'muted' })
  }

  return candidates.slice(0, MAX_SIGNAL_CHIPS)
}

export default function QuestCard({ quest, onOpen, visibleFields }: { quest: QuestMeta; onOpen: (qid: string) => void; visibleFields: Set<FieldKey> }) {
  const { t } = useTranslation()
  const questAttention = deriveQuestAttention(quest)
  const hasAttentionBar = questAttention.kind !== 'none'
  const isAttention = quest.status === 'user_review'
  const isWaitingInput = quest.status === 'waiting_input'
  const applyFailed = hasApplyFailed(quest)
  const isBlocked = quest.status === 'blocked'
  const humanException = quest.human_exception
  const failure = isBlocked ? failureSummary(quest) : null
  const isDone = quest.status === 'success'
  const isRunning = quest.status === 'running'
  const effectInfo = effectTypeInfo(quest.effect_type)
  const outputCounts = artifactKindCounts(quest)
  const policySignals = policyChips(quest)
  const phaseView = questPhaseView(quest)
  const stuckDiagnosis = getStuckDiagnosis(quest)
  const designSummary = quest.design_summary?.split('\n').find((line) => line.trim())?.trim()
  const reviewRequestedChanges = quest.final_verdict === 'request_changes'
  const reviewRejected = quest.final_verdict === 'reject'
  const reviewPassed = quest.final_verdict === 'pass'
  const reviewLabel = reviewRequestedChanges
    ? t('questCard.review.needChanges')
    : reviewRejected
      ? t('questCard.review.rejected')
      : reviewPassed
        ? t('questCard.review.passed')
        : t('questCard.review.needHuman')
  const reviewIcon = reviewPassed ? 'circle-check' : reviewRejected ? 'x' : reviewRequestedChanges ? 'rotate-ccw' : 'bell-ring'

  const showSignalRow = visibleFields.has('statusHint') && (
    !!effectInfo ||
    outputCounts.total > 0 ||
    !!designSummary ||
    !!quest.auto_passed_by_policy ||
    !!quest.auto_completed_by_policy ||
    !!quest.finalized_by ||
    !!humanException ||
    policySignals.length > 0
  )
  const signalChips = showSignalRow ? pickSignalChips(quest, effectInfo, outputCounts, policySignals, isDone) : []
  return (
    <button
      className={
        'quest-card status-card-' +
        quest.status.replace('_', '-') +
        (hasAttentionBar ? ' has-attention-bar' : '') +
        (isDone && !applyFailed ? ' is-muted' : '') +
        (applyFailed ? ' apply-failed' : '')
      }
      onClick={() => onOpen(quest.id)}
      title={quest.query}
    >
      {hasAttentionBar && <span className="attention-bar" />}
      <div className="quest-card-top">
        <span className="mono">#{quest.short_id || shortId(quest.id)}</span>
        <div className="quest-card-top-tags">
          {visibleFields.has('type') && (
            <span className={quest.type === 'design' ? 'design-kind' : ''}>
              {quest.type === 'design' && <span className="kind-dot" />}
              {typeLabel(quest.type)}
            </span>
          )}
          {quest.intensity === 'quick' && (
            <span className="intensity-badge quick">
              <Icon name="bolt" size={11} />
              {t('questCard.intensityQuick')}
            </span>
          )}
        </div>
      </div>
      <QuestSummary query={quest.query} variant="card" />
      {phaseView.available && visibleFields.has('statusHint') && (
        <div className="phase-line">
          <Icon name={phaseView.name.includes(i18n.t('quest.phase.roleReview')) ? 'wand-sparkles' : 'swords'} />
          <span>{phaseView.name}</span>
          <em className="mono">Phase {phaseView.currentIdx + 1}/{phaseView.count}</em>
        </div>
      )}
      {phaseView.reworkCount > 0 && visibleFields.has('statusHint') && (
        <div className="phase-line muted">
          <Icon name="rotate-ccw" />
          <span>{t('questCard.reworkCount', { count: phaseView.reworkCount })}</span>
        </div>
      )}
      {isRunning && visibleFields.has('statusHint') && (
        <div className="budget-line">
          <Icon name="play" />
          <span>{t('questCard.running')}</span>
          <em className="mono">{questDurationUsageLabel(quest)}</em>
        </div>
      )}
      {isWaitingInput && visibleFields.has('statusHint') && (
        <div className="review-line warning">
          <Icon name="message-circle" />
          <span>{quest.waiting_input?.question_text || t('questCard.waitingInputFallback')}</span>
          <em className="mono">{fmtDuration(questWaitingInputAgeMs(quest))}</em>
        </div>
      )}
      {isWaitingInput && quest.waiting_input && visibleFields.has('statusHint') && (
        <div className="failure-summary-line">
          <span>{[quest.waiting_input.asker || 'agent', quest.waiting_input.phase_type].filter(Boolean).join(' · ')}</span>
        </div>
      )}
      {isAttention && visibleFields.has('statusHint') && (
        <div className={'review-line' + (reviewPassed ? '' : ' warning')}>
          <Icon name={reviewIcon} />
          <span>{reviewLabel}</span>
          {quest.mage_score != null && <em className="mono">{quest.mage_score}/10</em>}
        </div>
      )}
      {isBlocked && visibleFields.has('statusHint') && (
        <div className="blocked-line">
          <Icon name="triangle-alert" />
          <span>{failure?.title || t('questCard.blockedFallback')}</span>
          <em className="mono">{questDurationUsageLabel(quest)}</em>
        </div>
      )}
      {isBlocked && failure && visibleFields.has('statusHint') && (
        <div className="failure-summary-line">
          <span>{failure.meta}</span>
          {(quest.resume_count || failure.recoverable != null) && (
            <em>{failure.recoverable ? t('questCard.recoverable') : t('questCard.needHumanJudgment')}</em>
          )}
        </div>
      )}
      {applyFailed && visibleFields.has('statusHint') && (
        <div className="blocked-line">
          <Icon name="triangle-alert" />
          <span>{quest.apply_error ? truncateText(quest.apply_error, 80) : t('questCard.applyFailedFallback')}</span>
          {quest.finalized_by && <em className="mono">{quest.finalized_by}</em>}
        </div>
      )}
      {humanException && visibleFields.has('statusHint') && (
        <div className={'human-exception-line risk-' + humanException.risk_level}>
          <Icon name={humanException.risk_level === 'high' ? 'triangle-alert' : 'circle-help'} />
          <span>{humanException.recommended_action}</span>
          <em className="mono">{humanException.source_status}</em>
        </div>
      )}
      {stuckDiagnosis?.source === 'runtime' && visibleFields.has('statusHint') && (
        <div className="human-exception-line runtime-warning">
          <Icon name="radio" />
          <span>{stuckDiagnosis.recommendedAction}</span>
          <em className="mono">{stuckDiagnosis.kind}</em>
        </div>
      )}
      {isAttention && visibleFields.has('statusHint') && signalChips.length > 0 && (
        <div className="quest-signal-row">
          {signalChips.map((chip) => (
            <span className={'quest-signal-chip ' + chip.tone} key={chip.key} title={chip.title}>
              {chip.icon && <Icon name={chip.icon} size={12} />}
              {chip.label}
            </span>
          ))}
        </div>
      )}
      {isDone && quest.applied && visibleFields.has('applied') && (
        <div className="applied-line">
          <Icon name="git-merge" />
          <span>{t('questCard.applied')}</span>
        </div>
      )}
      {!isAttention && showSignalRow && signalChips.length > 0 && (
        <div className="quest-signal-row">
          {signalChips.map((chip) => (
            <span className={'quest-signal-chip ' + chip.tone} key={chip.key} title={chip.title}>
              {chip.icon && <Icon name={chip.icon} size={12} />}
              {chip.label}
            </span>
          ))}
        </div>
      )}
      {designSummary && visibleFields.has('statusHint') && (
        <div className="design-summary-line">{designSummary}</div>
      )}
      {visibleFields.has('mageScore') && quest.mage_score != null && (
        <div className="review-line">
          <Icon name="circle-check" />
          <span>{t('questCard.mageScore')}</span>
          <em className="mono">{quest.mage_score}/10</em>
        </div>
      )}
      <div className="quest-card-foot">
        <span>
          <Icon name={quest.type === 'design' ? 'mage' : 'sword'} />
          {quest.status === 'reviewing' ? t('questCard.footReviewing') : quest.status === 'running' ? t('questCard.footWarrior') : quest.warrior_id || quest.mage_id || t('questCard.footWarriorFallback')}
        </span>
        {visibleFields.has('time') && (
          <span className="mono">{relativeTime(questActivityTime(quest))}</span>
        )}
      </div>
      {(isAttention || isWaitingInput || applyFailed || stuckDiagnosis) && (
        <div className="quest-actions-mini">
          <span>
            <Icon name={stuckDiagnosis?.source === 'runtime' ? 'radio' : applyFailed ? 'triangle-alert' : isWaitingInput ? 'message-circle' : 'check'} />
            {stuckDiagnosis ? t('questCard.miniAction.diagnosis') : applyFailed ? t('questCard.miniAction.handle') : isWaitingInput ? t('questCard.miniAction.view') : t('questCard.miniAction.handle')}
          </span>
          <span className="square-action">
            <Icon name="chevron-right" />
          </span>
        </div>
      )}
    </button>
  )
}
