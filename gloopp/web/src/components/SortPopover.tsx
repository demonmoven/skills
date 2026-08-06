import Icon from './Icon'
import Popover from './Popover'
import i18n from '../i18n'
import { useTranslation } from 'react-i18next'

export type SortBy = 'created_at_ms' | 'updated_at_ms' | 'mage_score'
export type SortDir = 'asc' | 'desc'

const SORT_OPTIONS: { value: SortBy; label: string }[] = [
  { value: 'created_at_ms', label: i18n.t('sortPopover.createdAt') },
  { value: 'updated_at_ms', label: i18n.t('sortPopover.updatedAt') },
  { value: 'mage_score', label: i18n.t('sortPopover.mageScore') },
]

type Props = {
  open: boolean
  onClose: () => void
  sortBy: SortBy
  sortDir: SortDir
  onChangeBy: (v: SortBy) => void
  onChangeDir: (v: SortDir) => void
}

export default function SortPopover({ open, onClose, sortBy, sortDir, onChangeBy, onChangeDir }: Props) {
  const { t } = useTranslation()
  return (
    <Popover open={open} onClose={onClose}>
      <div className="popover-pill-grid">
        {SORT_OPTIONS.map((opt) => (
          <button
            key={opt.value}
            className={'filter-pill' + (sortBy === opt.value ? ' active' : '')}
            onClick={() => onChangeBy(opt.value)}
          >
            {opt.label}
          </button>
        ))}
      </div>
      <div className="popover-divider" />
      <div className="popover-label">{t('sortPopover.direction')}</div>
      <div className="popover-pill-grid">
        <button
          className={'filter-pill' + (sortDir === 'desc' ? ' active' : '')}
          onClick={() => onChangeDir('desc')}
        >
          <Icon name="arrow-down" size={11} /> {t('sortPopover.desc')}
        </button>
        <button
          className={'filter-pill' + (sortDir === 'asc' ? ' active' : '')}
          onClick={() => onChangeDir('asc')}
        >
          <Icon name="arrow-up" size={11} /> {t('sortPopover.asc')}
        </button>
      </div>
    </Popover>
  )
}
