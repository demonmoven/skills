import type { AdventurerFile, ExecutorInfo } from '../api/types'
import Icon from '../components/Icon'
import { isSelectableAgentExecutor } from '../components/util'
import { useTranslation } from 'react-i18next'

export function QuestLoadingState() {
  const { t } = useTranslation()
  return (
    <div className="quest-state-panel loading">
      <Icon name="spinner" className="spin" />
      <strong>{t('questState.loading.title')}</strong>
      <span>{t('questState.loading.desc')}</span>
    </div>
  )
}

export function QuestEmptyState({
  executors,
  adventurers,
  onCreateQuest,
  onConfigureAgents,
  onCreateAdventurer,
}: {
  executors: ExecutorInfo[]
  adventurers: AdventurerFile[]
  onCreateQuest: () => void
  onConfigureAgents: () => void
  onCreateAdventurer: () => void
}) {
  const { t } = useTranslation()
  const hasEnabledAgent = executors.some(isSelectableAgentExecutor)
  const hasActiveWarrior = adventurers.some((item) => item.class === 'warrior' && item.status === 'active')
  const hasActiveMage = adventurers.some((item) => item.class === 'mage' && item.status === 'active')
  const missingAdventurer = !hasActiveWarrior || !hasActiveMage
  if (!hasEnabledAgent) {
    return (
      <div className="quest-state-panel">
        <Icon name="settings" />
        <strong>{t('questState.noAgent.title')}</strong>
        <span>{t('questState.noAgent.desc')}</span>
        <button type="button" className="button primary" onClick={onConfigureAgents}>
          <Icon name="settings" />
          {t('questState.noAgent.action')}
        </button>
      </div>
    )
  }
  if (missingAdventurer) {
    return (
      <div className="quest-state-panel">
        <Icon name="users" />
        <strong>{t('questState.noAdventurer.title')}</strong>
        <span>{t('questState.noAdventurer.desc')}</span>
        <button type="button" className="button primary" onClick={onCreateAdventurer}>
          <Icon name="plus" />
          {t('questState.noAdventurer.action')}
        </button>
      </div>
    )
  }
  return (
    <div className="quest-state-panel">
      <Icon name="clipboard-check" />
      <strong>{t('questState.empty.title')}</strong>
      <span>{t('questState.empty.desc')}</span>
      <button type="button" className="button primary" onClick={onCreateQuest}>
        <Icon name="plus" />
        {t('questState.empty.action')}
      </button>
    </div>
  )
}
