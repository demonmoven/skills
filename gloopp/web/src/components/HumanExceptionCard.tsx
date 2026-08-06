import type { HumanExceptionItem } from '../api/types'
import { relativeTime } from './util'
import Icon from './Icon'
import { useTranslation } from 'react-i18next'

type Props = {
  item: HumanExceptionItem
  onOpen: (item: HumanExceptionItem) => void
}

export default function HumanExceptionCard({ item, onOpen }: Props) {
  const { t } = useTranslation()
  return (
    <article className={'human-exception-card risk-' + item.risk_level}>
      <div className="inbox-icon">
        <Icon name={item.risk_level === 'high' ? 'triangle-alert' : 'circle-help'} />
      </div>
      <button type="button" className="inbox-main" onClick={() => onOpen(item)}>
        <div className="quest-card-top">
          <strong>{item.reason}</strong>
          <span className="chip source plain">{item.source_status}</span>
          <span className="mono push">{item.created_at_ms ? relativeTime(item.created_at_ms) : item.quest_id}</span>
        </div>
        <p>{item.recommended_action}</p>
        {item.available_actions?.length > 0 && (
          <div className="human-exception-actions">
            {item.available_actions.slice(0, 4).map((action) => (
              <span key={action} className="chip mono tiny">{action}</span>
            ))}
          </div>
        )}
        {item.evidence_summary && (
          <div className="inbox-source">
            <span>{t('inbox.humanException.evidence')}</span>
            <span className="mono">{item.evidence_summary}</span>
          </div>
        )}
      </button>
    </article>
  )
}
