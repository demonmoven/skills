import Popover from './Popover'
import i18n from '../i18n'
import { useTranslation } from 'react-i18next'

export type FieldKey = 'type' | 'statusHint' | 'time' | 'mageScore' | 'adventurer' | 'reworkCount' | 'diffStat' | 'applied'

const FIELD_OPTIONS: { key: FieldKey; label: string }[] = [
  { key: 'type', label: i18n.t('fieldsPopover.type') },
  { key: 'statusHint', label: i18n.t('fieldsPopover.statusHint') },
  { key: 'time', label: i18n.t('fieldsPopover.time') },
  { key: 'mageScore', label: i18n.t('fieldsPopover.mageScore') },
  { key: 'adventurer', label: i18n.t('fieldsPopover.adventurer') },
  { key: 'reworkCount', label: i18n.t('fieldsPopover.reworkCount') },
  { key: 'diffStat', label: i18n.t('fieldsPopover.diffStat') },
  { key: 'applied', label: i18n.t('fieldsPopover.applied') },
]

type Props = {
  open: boolean
  onClose: () => void
  visible: Set<FieldKey>
  onChange: (next: Set<FieldKey>) => void
}

export default function FieldsPopover({ open, onClose, visible, onChange }: Props) {
  const { t } = useTranslation()
  function toggle(key: FieldKey) {
    const next = new Set(visible)
    if (next.has(key)) next.delete(key)
    else next.add(key)
    onChange(next)
  }

  return (
    <Popover open={open} onClose={onClose}>
      <div className="popover-label">{t('fieldsPopover.label')}</div>
      <div className="popover-pill-grid">
        {FIELD_OPTIONS.map((opt) => (
          <button
            key={opt.key}
            className={'filter-pill' + (visible.has(opt.key) ? ' active' : '')}
            onClick={() => toggle(opt.key)}
          >
            {opt.label}
          </button>
        ))}
      </div>
    </Popover>
  )
}
