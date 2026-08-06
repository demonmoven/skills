import { ReactNode, useState } from 'react'
import Icon from './Icon'
import i18n from '../i18n'

type Props = {
  hint: string
  children: ReactNode
  title?: string
}

export default function AdvancedSection({ hint, children, title }: Props) {
  const [open, setOpen] = useState(false)

  return (
    <details
      className="advanced-section"
      open={open}
      onToggle={(e) => setOpen(e.currentTarget.open)}
    >
      <summary>
        <span>
          <Icon name="sliders-horizontal" />
          {title || i18n.t('common.advancedSettings')}
        </span>
        <em>{hint}</em>
        <Icon name="chevron-down" />
      </summary>
      <div className="advanced-body">{children}</div>
    </details>
  )
}
