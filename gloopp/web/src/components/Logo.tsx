type Variant = 'knight' | 'mage' | 'client' | 'badge' | 'official' | 'guild' | 'wordmark' | 'mono' | 'mono-inverted'

type Props = {
  size?: number
  variant?: Variant
  className?: string
}

import { useTranslation } from 'react-i18next'

const variantSrc: Record<Exclude<Variant, 'badge' | 'wordmark'>, string> = {
  knight: '/favicon-knight.svg',
  mage: '/favicon-mage.svg',
  client: '/favicon-client.svg',
  official: '/icon-128.svg',
  guild: '/icon-guild-badge.svg',
  mono: '/icon-mono.svg',
  'mono-inverted': '/icon-mono-inverted.svg',
}

export default function Logo({ size = 28, variant = 'knight', className }: Props) {
  const { t } = useTranslation()
  if (variant === 'badge') {
    return (
      <span
        className={'logo-badge ' + (className || '')}
        aria-hidden="true"
        style={{ width: size, height: size, fontSize: Math.max(11, Math.floor(size * 0.48)) }}
      >
        G
      </span>
    )
  }
  if (variant === 'wordmark') {
    return (
      <img
        className={'logo-mark ' + (className || '')}
        src="/icon-wordmark.svg"
        height={size}
        alt={t('aria.logo')}
        aria-hidden="true"
        style={{ height: size, display: 'inline-block', flexShrink: 0 }}
      />
    )
  }
  const src = variantSrc[variant]
  return (
    <img
      className={'logo-mark ' + (className || '')}
      src={src}
      width={size}
      height={size}
      alt={t('aria.logoSlime')}
      aria-hidden="true"
      style={{ width: size, height: size, display: 'inline-block', flexShrink: 0 }}
    />
  )
}
