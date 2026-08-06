export type CommandKind = 'navigate' | 'create' | 'open' | 'operate'
export type CommandEntity = 'quest' | 'inbox' | 'automation' | 'adventurer' | 'knowledge' | 'resource' | 'settings' | 'stats'
export type CommandIntent = 'monitor' | 'decide' | 'create' | 'configure' | 'inspect'
export type CommandSection = 'Decide' | 'Open' | 'Create' | 'Navigate' | 'Configure' | 'Inspect'

export const COMMAND_PRIORITY = {
  decideAttention: 10,
  openExact: 20,
  create: 30,
  navigatePrimary: 40,
  openNormal: 45,
  inspectResource: 50,
  configure: 60,
} as const

export type CommandItem = {
  id: string
  kind: CommandKind
  entity: CommandEntity
  intent: CommandIntent
  section?: CommandSection
  title: string
  subtitle?: string
  keywords?: string[]
  icon?: string
  priority: number
  disabledReason?: string
  run: () => void
}
