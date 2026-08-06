import type { AdventurerFile, ExecutorInfo, QuestStatus, QuestType } from '../api/types'
import i18n from '../i18n'
export { hasApplyFailed } from '../domain/questSelectors'

export function statusLabel(s: QuestStatus): string {
  return i18n.t(`quest.status.${s}`) || s
}

export function typeLabel(t: QuestType): string {
  return i18n.t(`quest.type.${t}`) || t
}

export function agentKey(executor: ExecutorInfo): string {
  return executor.agent || executor.id.replace(/^exe_/, '')
}

export function isSelectableAgentExecutor(executor: ExecutorInfo): boolean {
  return executor.enabled !== false && executor.type !== 'mock' && agentKey(executor) !== 'mock'
}

export function isRunnableAdventurer(adventurer: AdventurerFile, executors: ExecutorInfo[]): boolean {
  return (
    adventurer.status === 'active' &&
    executors.some((executor) => isSelectableAgentExecutor(executor) && agentKey(executor) === adventurer.agent)
  )
}

export function agentLabel(agent: string): string {
  const key = `agent.label.${agent}`
  const translated = i18n.t(key)
  return translated === key ? agent : translated
}

export function modelPolicyLabel(executor: ExecutorInfo | null | undefined): string {
  if (!executor) return i18n.t('agent.modelPolicy.selectFirst')
  const model = (executor.default_model || '').trim()
  return model ? i18n.t('agent.modelPolicy.userDefault', { model }) : i18n.t('agent.modelPolicy.agentDefault')
}

export function capabilityTierLabel(tier: string | undefined): string {
  switch (tier) {
    case 'tier_a':
      return i18n.t('agent.capabilityTier.tier_a')
    case 'tier_b':
      return i18n.t('agent.capabilityTier.tier_b')
    case 'tier_c':
      return i18n.t('agent.capabilityTier.tier_c')
    case 'test_only':
      return i18n.t('agent.capabilityTier.test_only')
    default:
      return i18n.t('agent.capabilityTier.unknown')
  }
}

export function adventurerModelPolicyLabel(model: string | undefined, executor: ExecutorInfo | null | undefined): string {
  const override = (model || '').trim()
  if (override) return i18n.t('agent.modelPolicy.adventurerOverride', { model })
  return modelPolicyLabel(executor)
}

export function fmtTime(ms?: number): string {
  if (!ms) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(ms))
}

export function relativeTime(ms?: number): string {
  if (!ms) return '-'
  const delta = Date.now() - ms
  const minute = 60 * 1000
  const hour = 60 * minute
  const day = 24 * hour
  if (delta < minute) return i18n.t('time.justNow')
  if (delta < hour) return i18n.t('time.minutesAgo', { count: Math.max(1, Math.floor(delta / minute)) })
  if (delta < day) return i18n.t('time.hoursAgo', { count: Math.floor(delta / hour) })
  return i18n.t('time.daysAgo', { count: Math.floor(delta / day) })
}

/**
 * 将毫秒格式化为易读的时长字符串。
 * - 小于 1 分钟：显示秒数，如 "45s"
 * - 小于 1 小时：显示分钟，如 "32m"
 * - 大于等于 1 小时：显示小时+分钟（分钟不为0时），如 "2h 5m"、"3h"
 */
export function fmtDuration(ms?: number): string {
  if (ms == null || !isFinite(ms) || ms < 0) return '-'
  const totalSeconds = Math.floor(ms / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  if (hours > 0) {
    return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`
  }
  if (minutes > 0) {
    return `${minutes}m`
  }
  return `${seconds}s`
}

/**
 * 格式化"已用/上限"时长对，如 "2h 5m / 3h"。
 * 若上限不存在则只显示已用。
 */
export function fmtDurationPair(usedMs?: number, maxMs?: number): string {
  const used = fmtDuration(usedMs)
  if (maxMs == null || !isFinite(maxMs) || maxMs <= 0) return used
  return `${used} / ${fmtDuration(maxMs)}`
}

export function humanizeMilliseconds(value: string): string {
  return value.replace(/(\d+)ms\s*\/\s*(\d+)ms/g, (_, used, max) => {
    return fmtDurationPair(Number(used), Number(max))
  }).replace(/(\d+)ms/g, (_, ms) => {
    return fmtDuration(Number(ms))
  })
}

/**
 * 根据 quest 的结构化字段，生成人类友好的阻塞原因描述。
 * 如果没有足够结构化数据，则回退到原始 blocked_reason。
 */
export function fmtBlockedReason(quest: {
  blocked_reason?: string
  failure_attribution?: { reason?: unknown; stage?: string; category?: string }
}): string {
  const raw = quest.blocked_reason || i18n.t('blockedReason.default')
  return humanizeMilliseconds(raw)
}

export function shortId(id?: string): string {
  if (!id) return '-'
  return id.length > 9 ? id.slice(0, 9) : id
}

/**
 * 防抖：在连续调用中只执行最后一次，延迟 wait 毫秒。
 * 返回的函数带 .cancel() 可取消待执行的调用。
 */
export function debounce<T extends (...args: any[]) => void>(fn: T, wait = 200): T & { cancel: () => void } {
  let timer: ReturnType<typeof setTimeout> | null = null
  const debounced = function (this: any, ...args: any[]) {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => {
      timer = null
      fn.apply(this, args)
    }, wait)
  } as any
  debounced.cancel = () => {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
  }
  return debounced
}
