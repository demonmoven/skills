import Popover from './Popover'
import i18n from '../i18n'

export type GroupBy = 'none' | 'status' | 'type' | 'phase'

const OPTIONS: { value: GroupBy; label: string }[] = [
  { value: 'status', label: i18n.t('groupPopover.status') },
  { value: 'phase', label: i18n.t('groupPopover.phase') },
  { value: 'type', label: i18n.t('groupPopover.type') },
  { value: 'none', label: i18n.t('groupPopover.none') },
]

type Props = {
  open: boolean
  onClose: () => void
  value: GroupBy
  onChange: (v: GroupBy) => void
}

export default function GroupPopover({ open, onClose, value, onChange }: Props) {
  return (
    <Popover open={open} onClose={onClose}>
      <div className="popover-pill-grid">
        {OPTIONS.map((opt) => (
          <button
            key={opt.value}
            className={'filter-pill' + (value === opt.value ? ' active' : '')}
            onClick={() => { onChange(opt.value); onClose() }}
          >
            {opt.label}
          </button>
        ))}
      </div>
    </Popover>
  )
}
