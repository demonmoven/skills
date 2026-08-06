import { capabilityTierLabel } from './util'
import Icon from './Icon'
import i18n from '../i18n'

export function capabilityTierRank(tier: string | undefined): number {
  switch (tier) {
    case 'tier_a':
      return 0
    case 'tier_b':
      return 1
    case 'tier_c':
      return 3
    case 'test_only':
      return 4
    default:
      return 2
  }
}

export function compareCapabilityTiers(a: string | undefined, b: string | undefined): number {
  return capabilityTierRank(a) - capabilityTierRank(b)
}

export function capabilityTierAdvice(tier: string | undefined): string {
  switch (tier) {
    case 'tier_a':
      return i18n.t('agent.capabilityTierAdvice.tier_a')
    case 'tier_b':
      return i18n.t('agent.capabilityTierAdvice.tier_b')
    case 'tier_c':
      return i18n.t('agent.capabilityTierAdvice.tier_c')
    case 'test_only':
      return i18n.t('agent.capabilityTierAdvice.test_only')
    default:
      return i18n.t('agent.capabilityTierAdvice.unknown')
  }
}

export default function CapabilityTierBadge({ tier, compact = false }: { tier?: string; compact?: boolean }) {
  const tone = tier === 'tier_a' ? 'tier-a'
    : tier === 'tier_b' ? 'tier-b'
      : tier === 'tier_c' ? 'tier-c'
        : tier === 'test_only' ? 'test-only'
          : 'unknown'
  return (
    <span className={'capability-tier-badge ' + tone + (compact ? ' compact' : '')} title={capabilityTierAdvice(tier)}>
      <Icon name={tier === 'tier_c' ? 'triangle-alert' : 'shield'} size={compact ? 11 : 12} />
      {capabilityTierLabel(tier)}
    </span>
  )
}
