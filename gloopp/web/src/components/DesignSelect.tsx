import { useMemo } from 'react'
import Icon from './Icon'
import i18n from '../i18n'

export type DesignSelectOption = {
  value: string
  label: string
}

type Props = {
  value: string
  options: DesignSelectOption[]
  onChange: (value: string) => void
  ariaLabel?: string
  compact?: boolean
  disabled?: boolean
}

export default function DesignSelect({ value, options, onChange, ariaLabel, compact, disabled }: Props) {
  const selected = useMemo(
    () => options.find((option) => option.value === value) || options[0],
    [options, value],
  )

  return (
    <details className={'design-select' + (compact ? ' compact' : '') + (disabled ? ' disabled' : '')} name="design-select">
      <summary aria-label={`${ariaLabel || i18n.t('common.action.select')} ${selected?.label || i18n.t('common.state.selectPlaceholder')}`}>
        <span>{selected?.label || i18n.t('common.state.selectPlaceholder')}</span>
        <Icon name="chevron-down" />
      </summary>
      <div className="design-select-menu">
        {options.map((option) => (
          <button
            className={option.value === value ? 'active' : ''}
            disabled={disabled}
            key={option.value}
            type="button"
            onClick={(event) => {
              if (disabled) return
              onChange(option.value)
              event.currentTarget.closest('details')?.removeAttribute('open')
            }}
          >
            <span>{option.label}</span>
            {option.value === value && <Icon name="check" />}
          </button>
        ))}
      </div>
    </details>
  )
}
