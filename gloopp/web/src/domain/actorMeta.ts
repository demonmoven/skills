import type { ThreadAuthorRole } from '../api/types'

export type ActorTone = 'maker' | 'checker' | 'system' | 'human' | 'automation'

export type ActorMeta = {
  role: ThreadAuthorRole
  label: string
  icon: string
  tone: ActorTone
  known: boolean
}

const ACTOR_META: Record<string, ActorMeta> = {
  maker: { role: 'maker', label: 'Maker', icon: 'sword', tone: 'maker', known: true },
  checker: { role: 'checker', label: 'Checker', icon: 'wand-sparkles', tone: 'checker', known: true },
  system: { role: 'system', label: 'System', icon: 'shield', tone: 'system', known: true },
  human: { role: 'human', label: 'Human', icon: 'user', tone: 'human', known: true },
  automation: { role: 'automation', label: 'Automation', icon: 'bolt', tone: 'automation', known: true },
}

/** Returns meta only for known roles. Callers should fall back to their own display logic for unknown roles. */
export function actorMeta(role?: string | null): ActorMeta | null {
  if (!role) return null
  return ACTOR_META[role] || null
}

/** Fallback for surfaces that must display something even for unknown roles (e.g. ThreadLedger). */
export function fallbackActorMeta(role?: string | null): ActorMeta {
  const known = actorMeta(role)
  if (known) return known
  return { role: role || 'unknown', label: role || 'Unknown', icon: 'message-circle', tone: 'system', known: false }
}
