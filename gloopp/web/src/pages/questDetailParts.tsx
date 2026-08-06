import { useTranslation } from 'react-i18next'
import Icon from '../components/Icon'
import type { QuestMeta } from '../api/types'
import {
  questResourceUsage,
  questDurationUsageLabel,
  questTurnUsageLabel,
  type TimelineVariant,
} from '../domain/questSelectors'

export function PhaseNode({ variant, icon }: { variant: TimelineVariant; icon: string }) {
  return (
    <div className={'timeline-node ' + variant}>
      <Icon name={icon} size={14} />
    </div>
  )
}

export function PathValue({ value, fallback }: { value?: string; fallback: string }) {
  if (!value) {
    return <span className="faint">{fallback}</span>
  }
  return (
    <code className="path-value" title={value}>
      {value}
    </code>
  )
}

export function ResourceBudgetRows({ quest }: { quest: QuestMeta }) {
  const { t } = useTranslation()
  const usage = questResourceUsage(quest)
  return (
    <>
      <div className="info-kv">
        <span>{t('questResource.duration')}</span>
        <div className="side-meta-line">
          <span className="mono">{questDurationUsageLabel(quest)}</span>
          <em>{t('questResource.consumption')}</em>
        </div>
      </div>
      <div className="info-kv">
        <span>{t('questResource.turns')} <em className="muted">{t('questResource.safetyNet')}</em></span>
        <div className="side-meta-line">
          <span className="mono">{questTurnUsageLabel(quest)}</span>
          {usage.maxTurnsPerPhase ? <em>{t('questResource.perPhaseLimit')}</em> : <em>{t('questResource.noLimit')}</em>}
        </div>
      </div>
    </>
  )
}
