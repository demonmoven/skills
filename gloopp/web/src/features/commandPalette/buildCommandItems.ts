import type {
  AdventurerFile,
  AutomationConfig,
  HumanExceptionItem,
  PromptTemplate,
  QuestMeta,
  SkillManifest,
} from '../../api/types'
import { hasApplyFailed, relativeTime, shortId, statusLabel } from '../../components/util'
import i18n from '../../i18n'
import type { CommandEntity, CommandItem, CommandIntent } from './types'
import { COMMAND_PRIORITY } from './types'

export type CommandPaletteTab =
  | 'quests'
  | 'inbox'
  | 'automations'
  | 'knowledge'
  | 'adventurers'
  | 'resources'
  | 'stats'
  | 'settings'

export type CommandPaletteActions = {
  navigate: (tab: CommandPaletteTab) => void
  createQuest: () => void
  createAutomation: () => void
  createAdventurer: () => void
  openQuest: (qid: string) => void
  openAutomation: (id: string) => void
  openAdventurer: (id: string) => void
  openSkill: (name: string) => void
  openPrompt: (name: string) => void
}

export type BuildCommandItemsInput = {
  quests: QuestMeta[]
  inbox: QuestMeta[]
  humanExceptions: HumanExceptionItem[]
  automations: AutomationConfig[]
  adventurers: AdventurerFile[]
  skills: SkillManifest[]
  prompts: PromptTemplate[]
  actions: CommandPaletteActions
}

const NAV_ENTRIES: Array<{
  key: CommandPaletteTab
  entity: CommandEntity
  intent: CommandIntent
  titleKey: string
  icon: string
  keywords: string[]
}> = [
  { key: 'quests', entity: 'quest', intent: 'monitor', titleKey: 'palette.cmd.openQuests', icon: 'grid', keywords: ['委托', 'feed', 'board', 'list', 'quest'] },
  { key: 'inbox', entity: 'inbox', intent: 'decide', titleKey: 'palette.cmd.openInbox', icon: 'inbox', keywords: ['收件箱', '待处理', '决策', 'inbox'] },
  { key: 'automations', entity: 'automation', intent: 'monitor', titleKey: 'palette.cmd.openAutomations', icon: 'bolt', keywords: ['自动化', 'automation', 'schedule', 'cron'] },
  { key: 'knowledge', entity: 'knowledge', intent: 'inspect', titleKey: 'palette.cmd.openKnowledge', icon: 'file-text', keywords: ['知识', '知识库', 'knowledge'] },
  { key: 'adventurers', entity: 'adventurer', intent: 'configure', titleKey: 'palette.cmd.openAdventurers', icon: 'users', keywords: ['冒险者', '执行者', 'adventurer', 'agent'] },
  { key: 'resources', entity: 'resource', intent: 'inspect', titleKey: 'palette.cmd.openResources', icon: 'package', keywords: ['资源', '技能', '提示词', 'resource', 'skill', 'prompt'] },
  { key: 'stats', entity: 'stats', intent: 'inspect', titleKey: 'palette.cmd.openStats', icon: 'board', keywords: ['统计', 'stats', 'metrics'] },
  { key: 'settings', entity: 'settings', intent: 'configure', titleKey: 'palette.cmd.openSettings', icon: 'settings', keywords: ['设置', 'settings', '配置'] },
]

export function buildCommandItems({
  quests,
  inbox,
  humanExceptions,
  automations,
  adventurers,
  skills,
  prompts,
  actions,
}: BuildCommandItemsInput): CommandItem[] {
  const items: CommandItem[] = []

  for (const entry of NAV_ENTRIES) {
    items.push({
      id: `navigate:${entry.key}`,
      kind: 'navigate',
      entity: entry.entity,
      intent: entry.intent,
      section: 'Navigate',
      title: i18n.t(entry.titleKey),
      icon: entry.icon,
      keywords: entry.keywords,
      priority: COMMAND_PRIORITY.navigatePrimary,
      run: () => actions.navigate(entry.key),
    })
  }

  items.push({
    id: 'create:quest',
    kind: 'create',
    entity: 'quest',
    intent: 'create',
    section: 'Create',
    title: i18n.t('palette.cmd.createQuest'),
    icon: 'plus',
    keywords: ['新建', '委托', 'create', 'quest', 'new'],
    priority: COMMAND_PRIORITY.create,
    run: actions.createQuest,
  })
  items.push({
    id: 'create:automation',
    kind: 'create',
    entity: 'automation',
    intent: 'create',
    section: 'Create',
    title: i18n.t('palette.cmd.createAutomation'),
    icon: 'plus',
    keywords: ['新建', '自动化', 'create', 'automation', 'new'],
    priority: COMMAND_PRIORITY.create,
    run: actions.createAutomation,
  })
  items.push({
    id: 'create:adventurer',
    kind: 'create',
    entity: 'adventurer',
    intent: 'create',
    section: 'Create',
    title: i18n.t('palette.cmd.recruitAdventurer'),
    icon: 'plus',
    keywords: ['招募', '冒险者', '执行者', 'recruit', 'adventurer', 'new'],
    priority: COMMAND_PRIORITY.create,
    run: actions.createAdventurer,
  })

  const humanExceptionByQuestID = new Map(humanExceptions.map((item) => [item.quest_id, item]))
  for (const item of humanExceptions) {
    items.push({
      id: `open:inbox:${item.id || item.quest_id}`,
      kind: 'open',
      entity: 'inbox',
      intent: 'decide',
      section: 'Decide',
      title: i18n.t('palette.cmd.openDecision', { title: decisionTitle(item) }),
      subtitle: [statusLabelFromString(item.source_status), relativeTime(item.created_at_ms), item.reason].filter(Boolean).join(' · '),
      icon: item.source_status === 'waiting_input' ? 'message-circle' : 'triangle-alert',
      keywords: compactStrings(['待处理', '决策', 'inbox', item.quest_id, item.reason, item.recommended_action, item.source_status]),
      priority: COMMAND_PRIORITY.decideAttention,
      run: () => actions.navigate('inbox'),
    })
  }

  for (const quest of inbox) {
    if (humanExceptionByQuestID.has(quest.id)) continue
    const createdBy = String(quest.created_by || '')
    const isAutomationCandidate = createdBy.startsWith('automation:')
    items.push({
      id: `open:inbox:${quest.id}`,
      kind: 'open',
      entity: 'inbox',
      intent: 'decide',
      section: 'Decide',
      title: i18n.t('palette.cmd.openInboxCandidate', { title: questTitle(quest) }),
      subtitle: [isAutomationCandidate ? i18n.t('palette.meta.automationCandidate') : i18n.t('palette.meta.pendingDelegation'), shortId(quest.short_id || quest.id)].join(' · '),
      icon: isAutomationCandidate ? 'bolt' : 'inbox',
      keywords: compactStrings(['候选', '待确认', 'inbox', 'candidate', quest.id, quest.short_id, quest.query, createdBy]),
      priority: COMMAND_PRIORITY.decideAttention,
      run: () => actions.navigate('inbox'),
    })
  }

  for (const quest of quests) {
    const attention = isAttentionQuest(quest)
    items.push({
      id: `open:quest:${quest.id}`,
      kind: 'open',
      entity: 'quest',
      intent: attention ? 'decide' : 'monitor',
      section: 'Open',
      title: i18n.t('palette.cmd.openQuest', { title: questTitle(quest) }),
      subtitle: [
        '#' + shortId(quest.short_id || quest.id),
        statusLabel(quest.status),
        projectName(quest),
        relativeTime(quest.updated_at_ms || quest.created_at_ms),
      ].filter(Boolean).join(' · '),
      icon: attention ? 'bell-ring' : 'grid',
      keywords: compactStrings(['委托', 'quest', quest.id, quest.short_id, quest.query, quest.status, projectName(quest)]),
      priority: attention ? COMMAND_PRIORITY.openExact : COMMAND_PRIORITY.openNormal,
      run: () => actions.openQuest(quest.id),
    })
  }

  for (const automation of automations) {
    items.push({
      id: `open:automation:${automation.id}`,
      kind: 'open',
      entity: 'automation',
      intent: 'configure',
      section: 'Open',
      title: i18n.t('palette.cmd.openAutomation', { name: automation.name || automation.id }),
      subtitle: [
        automation.enabled ? i18n.t('palette.meta.enabled') : i18n.t('palette.meta.disabled'),
        automation.trigger || '',
        automation.description || '',
      ].filter(Boolean).join(' · '),
      icon: 'bolt',
      keywords: compactStrings(['自动化', 'automation', automation.id, automation.name, automation.description, automation.trigger, ...(automation.tags || [])]),
      priority: COMMAND_PRIORITY.openNormal,
      run: () => actions.openAutomation(automation.id),
    })
  }

  for (const adventurer of adventurers) {
    items.push({
      id: `open:adventurer:${adventurer.id}`,
      kind: 'open',
      entity: 'adventurer',
      intent: 'configure',
      section: 'Open',
      title: i18n.t('palette.cmd.openAdventurer', { name: adventurer.name || adventurer.id }),
      subtitle: [
        adventurer.class === 'mage' ? i18n.t('world.mage') : i18n.t('world.warrior'),
        adventurer.status ? i18n.t(`adventurer.status.${adventurer.status === 'pending_setup' ? 'pending' : adventurer.status}`) : '',
        adventurer.agent || '',
      ].filter(Boolean).join(' · '),
      icon: adventurer.class === 'mage' ? 'wand-sparkles' : 'sword',
      keywords: compactStrings(['冒险者', '执行者', 'adventurer', 'agent', adventurer.id, adventurer.name, adventurer.description, adventurer.class, adventurer.agent]),
      priority: COMMAND_PRIORITY.openNormal,
      run: () => actions.openAdventurer(adventurer.id),
    })
  }

  for (const skill of skills) {
    items.push({
      id: `open:resource:skill:${skill.name}`,
      kind: 'open',
      entity: 'resource',
      intent: 'inspect',
      section: 'Inspect',
      title: i18n.t('palette.cmd.openSkill', { name: skill.name }),
      subtitle: [i18n.t('nav.skills'), skill.category || '', skill.description || ''].filter(Boolean).join(' · '),
      icon: 'skills',
      keywords: compactStrings(['资源', '技能', 'skill', skill.name, skill.description, skill.category]),
      priority: COMMAND_PRIORITY.inspectResource,
      run: () => actions.openSkill(skill.name),
    })
  }

  for (const prompt of prompts) {
    items.push({
      id: `open:resource:prompt:${prompt.name}`,
      kind: 'open',
      entity: 'resource',
      intent: 'inspect',
      section: 'Inspect',
      title: i18n.t('palette.cmd.openPrompt', { name: prompt.name.split('/').pop() || prompt.name }),
      subtitle: [i18n.t('nav.prompts'), prompt.source, prompt.preview || ''].filter(Boolean).join(' · '),
      icon: 'message-square',
      keywords: compactStrings(['资源', '提示词', 'prompt', prompt.name, prompt.source, prompt.preview]),
      priority: COMMAND_PRIORITY.inspectResource,
      run: () => actions.openPrompt(prompt.name),
    })
  }

  return items
}

function questTitle(quest: QuestMeta): string {
  return quest.intent_summary || quest.query || quest.short_id || shortId(quest.id)
}

function projectName(quest: QuestMeta): string {
  const dir = quest.base_working_dir || ''
  return dir.split('/').filter(Boolean).pop() || ''
}

function isAttentionQuest(quest: QuestMeta): boolean {
  return quest.status === 'user_review' || quest.status === 'waiting_input' || quest.status === 'blocked' || hasApplyFailed(quest)
}

function decisionTitle(item: HumanExceptionItem): string {
  return item.reason || item.quest_id || i18n.t('palette.meta.decision')
}

function statusLabelFromString(status?: string): string {
  if (!status) return ''
  const key = `quest.status.${status}`
  const translated = i18n.t(key)
  return translated === key ? status : translated
}

function compactStrings(values: Array<string | undefined | null | false>): string[] {
  return values.filter((value): value is string => typeof value === 'string' && value.length > 0)
}
