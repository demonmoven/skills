import DesignSelect from './DesignSelect'
import i18n from '../i18n'

export function cronPreset(value: string): 'daily' | 'daily9' | 'weekday' | 'weekly' | 'custom' {
  if (value === 'daily' || value === '0 0 * * *') return 'daily'
  if (value === 'daily9' || value === '0 9 * * *') return 'daily9'
  if (value === 'weekday' || value === '0 9 * * 1-5') return 'weekday'
  if (value === 'weekly' || value === '0 0 * * 1') return 'weekly'
  return 'custom'
}

export function cronValue(preset: string): string {
  if (preset === 'daily') return '0 0 * * *'
  if (preset === 'daily9') return '0 9 * * *'
  if (preset === 'weekday') return '0 9 * * 1-5'
  if (preset === 'weekly') return '0 0 * * 1'
  return ''
}

export default function CronField({
  value,
  onChange,
  disabled,
}: {
  value: string
  onChange: (value: string) => void
  disabled?: boolean
}) {
  const preset = cronPreset(value)
  return (
    <label className="field choice-field">
      <span>{i18n.t('automation.cron.rule')}</span>
      <DesignSelect
        compact
        value={preset}
        options={[
          { value: 'daily', label: i18n.t('automation.cron.daily') },
          { value: 'daily9', label: i18n.t('automation.cron.daily9') },
          { value: 'weekday', label: i18n.t('automation.cron.weekday') },
          { value: 'weekly', label: i18n.t('automation.cron.weekly') },
          { value: 'custom', label: i18n.t('automation.cron.custom') },
        ]}
        onChange={(next) => {
          if (next === 'custom') {
            if (preset !== 'custom') onChange('')
            return
          }
          onChange(cronValue(next))
        }}
        ariaLabel={i18n.t('automation.cron.rule')}
        disabled={disabled}
      />
      {preset === 'custom' && (
        <input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="0 9 * * *"
          className="mono"
          disabled={disabled}
        />
      )}
    </label>
  )
}
