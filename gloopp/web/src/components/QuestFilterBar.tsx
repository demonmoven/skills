import { useTranslation } from 'react-i18next'
import type { QuestStatus, QuestType } from '../api/types'
import type { QuestRange } from '../hooks/useDashboardData'
import { statusLabel, typeLabel } from './util'

type QuestFilters = {
  statuses: QuestStatus[]
  types: QuestType[]
  questIds: string[]
  projects: string[]
  failureReasons: string[]
  failureCategories: string[]
}

type Props = {
  questRange: QuestRange
  questFilters: QuestFilters
  hasStatsDrilldownFilter: boolean
  hasFilterValue: boolean
  onQuestRangeChange: (range: QuestRange) => void
  onToggleStatus: (status: QuestStatus) => void
  onToggleType: (type: QuestType) => void
  onReset: () => void
}

export default function QuestFilterBar({
  questRange,
  questFilters,
  hasStatsDrilldownFilter,
  hasFilterValue,
  onQuestRangeChange,
  onToggleStatus,
  onToggleType,
  onReset,
}: Props) {
  const { t } = useTranslation()
  return (
    <div className="filter-bar">
      <div className="filter-bar-group">
        <span>{t('quest.filter.time')}</span>
        <div className="filter-pill-row">
          {(['today', '7d', '30d', 'all'] as QuestRange[]).map((r) => (
            <button key={r} className={'filter-pill' + (questRange === r ? ' active' : '')} onClick={() => onQuestRangeChange(r)}>
              {t(`topbar.stats.range.${r}`)}
            </button>
          ))}
        </div>
      </div>
      <div className="filter-bar-sep" />
      <div className="filter-bar-group">
        <span>{t('quest.filter.status')}</span>
        <div className="filter-pill-row">
          {(['pending', 'running', 'reviewing', 'waiting_input', 'user_review', 'blocked', 'success', 'failed', 'cancelled'] as QuestStatus[]).map(s => (
            <button key={s} className={'filter-pill' + (questFilters.statuses.includes(s) ? ' active' : '')} onClick={() => onToggleStatus(s)}>
              {statusLabel(s)}
            </button>
          ))}
        </div>
      </div>
      <div className="filter-bar-sep" />
      <div className="filter-bar-group">
        <span>{t('quest.filter.type')}</span>
        <div className="filter-pill-row">
          {(['execute', 'design'] as QuestType[]).map(ty => (
            <button key={ty} className={'filter-pill' + (questFilters.types.includes(ty) ? ' active' : '')} onClick={() => onToggleType(ty)}>
              {typeLabel(ty)}
            </button>
          ))}
        </div>
      </div>
      {hasStatsDrilldownFilter && (
        <>
          <div className="filter-bar-sep" />
          <div className="filter-bar-group">
            <span>{t('quest.filter.diagnosis')}</span>
            <div className="filter-pill-row">
              {questFilters.questIds.map((id) => (
                <span key={'quest-' + id} className="filter-chip-static">{t('quest.filter.questChip', { id: id.slice(0, 8) })}</span>
              ))}
              {questFilters.projects.map((project) => (
                <span key={'project-' + project} className="filter-chip-static">{t('quest.filter.projectChip', { name: project })}</span>
              ))}
              {questFilters.failureReasons.map((reason) => (
                <span key={'reason-' + reason} className="filter-chip-static">{t('quest.filter.reasonChip', { reason })}</span>
              ))}
              {questFilters.failureCategories.map((category) => (
                <span key={'category-' + category} className="filter-chip-static">{t('quest.filter.categoryChip', { category })}</span>
              ))}
            </div>
          </div>
        </>
      )}
      {hasFilterValue && (
        <button className="filter-bar-reset" onClick={onReset}>{t('quest.filter.reset')}</button>
      )}
    </div>
  )
}
