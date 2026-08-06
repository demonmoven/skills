import { ContextWorkbench } from '../components/ContextSettingsSection'
import Icon from '../components/Icon'
import { useTranslation } from 'react-i18next'

type Props = {
  onError: (msg: string) => void
  onOpenQuest: (qid: string) => void
}

export default function Knowledge({ onError, onOpenQuest }: Props) {
  const { t } = useTranslation()
  return (
    <div className="knowledge-surface">
      <section className="knowledge-hero">
        <div>
          <span className="settings-overview-kicker">
            <Icon name="file-text" />
            {t('knowledge.kicker')}
          </span>
          <h2>Knowledge</h2>
          <p>
            {t('knowledge.desc')}
          </p>
        </div>
        <div className="knowledge-format">
          <strong>{t('knowledge.formatLabel')}</strong>
          <span>{t('knowledge.formatValue')}</span>
          <em>{t('knowledge.formatNote')}</em>
        </div>
      </section>
      <ContextWorkbench onError={onError} onOpenQuest={onOpenQuest} />
    </div>
  )
}
