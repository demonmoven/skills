import { useTranslation } from 'react-i18next'
import Icon from './Icon'
import type { ImpactSummary } from '../api/types'

type Props = {
  impact: ImpactSummary | null | undefined
  className?: string
  compact?: boolean
}

function isEmpty(i: ImpactSummary | null | undefined): boolean {
  if (!i) return true
  return !i.what_changed && !(i.affected?.length) && !(i.not_touched?.length) && !(i.caveats?.length)
}

export default function ImpactSummaryCard({ impact, className, compact }: Props) {
  const { t } = useTranslation()
  if (isEmpty(impact)) return null
  const i = impact!

  if (compact) {
    return (
      <div className={'impact-summary-card compact' + (className ? ' ' + className : '')}>
        <Icon name="zap" size={14} />
        <span className="impact-compact-text">
          {i.what_changed || t('impact.noSummary')}
          {i.affected && i.affected.length > 0 && <em className="impact-compact-affected"> {t('impact.affectedCount', { count: i.affected.length })}</em>}
        </span>
      </div>
    )
  }

  return (
    <section className={'panel impact-summary-card' + (className ? ' ' + className : '')} aria-label={t('impact.aria')}>
      <div className="panel-title">
        <h2><Icon name="zap" size={14} />{t('impact.title')}</h2>
        <span className="changes-subtitle">{t('impact.subtitle')}</span>
      </div>
      <div className="impact-body">
        {i.what_changed && (
          <div className="impact-row impact-what">
            <span className="impact-label">{t('impact.label.what')}</span>
            <span className="impact-value">{i.what_changed}</span>
          </div>
        )}
        {i.affected && i.affected.length > 0 && (
          <div className="impact-row">
            <span className="impact-label">{t('impact.label.affected')}</span>
            <div className="impact-tags">
              {i.affected.map((a: string, idx: number) => <span className="chip tiny impact-tag affected" key={idx}>{a}</span>)}
            </div>
          </div>
        )}
        {i.not_touched && i.not_touched.length > 0 && (
          <div className="impact-row">
            <span className="impact-label">{t('impact.label.untouched')}</span>
            <div className="impact-tags">
              {i.not_touched.map((a: string, idx: number) => <span className="chip tiny impact-tag not-touched" key={idx}>{a}</span>)}
            </div>
          </div>
        )}
        {i.caveats && i.caveats.length > 0 && (
          <div className="impact-row impact-caveats">
            <span className="impact-label">{t('impact.label.caveats')}</span>
            <ul className="impact-caveat-list">
              {i.caveats.map((c: string, idx: number) => <li key={idx}>{c}</li>)}
            </ul>
          </div>
        )}
      </div>
    </section>
  )
}
