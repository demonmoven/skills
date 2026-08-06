import { useEffect, useState } from 'react'
import { api } from '../api/client'
import type { UpdateStatus } from '../api/types'
import Icon from './Icon'
import { useTranslation } from 'react-i18next'

const DISMISS_KEY = 'gloop_update_banner_dismissed'

export default function UpgradeBanner() {
  const { t } = useTranslation()
  const [status, setStatus] = useState<UpdateStatus | null>(null)
  const [dismissed, setDismissed] = useState(() => {
    const raw = localStorage.getItem(DISMISS_KEY)
    if (!raw) return null as string | null
    try {
      const obj = JSON.parse(raw) as { latest: string; until: number }
      if (obj.until > Date.now()) return obj.latest
    } catch {
      /* ignore corrupt storage */
    }
    return null
  })

  useEffect(() => {
    let cancelled = false
    async function load() {
      try {
        const s = await api.get<UpdateStatus>('/api/update-status')
        if (!cancelled) setStatus(s)
      } catch {
        /* 版本提示只是附加值，失败不打扰用户 */
      }
    }
    load()
    // 页面活跃时每 10 分钟轮询一次——daemon 会做节流，所以这个频率安全。
    const t = window.setInterval(load, 10 * 60 * 1000)
    return () => {
      cancelled = true
      window.clearInterval(t)
    }
  }, [])

  function dismissForAWeek() {
    if (!status?.latest) return
    const payload = { latest: status.latest, until: Date.now() + 7 * 24 * 60 * 60 * 1000 }
    localStorage.setItem(DISMISS_KEY, JSON.stringify(payload))
    setDismissed(status.latest)
  }

  if (!status || status.state !== 'outdated') return null
  if (dismissed && dismissed === status.latest) return null

  const rawHint = status.upgrade_hint?.trim() || 'gloop update'
  const hint = rawHint.startsWith('npm i -g') ? 'gloop update' : rawHint
  const latest = status.latest || t('upgradeBanner.newVersion')
  const current = status.current || t('upgradeBanner.currentVersion')

  return (
    <div className="top-banner" role="status" aria-live="polite">
      <Icon name="alert-triangle" />
      <span>
        {t('upgradeBanner.messagePrefix')}
        <code style={{ margin: '0 4px' }}>{current}</code>
        →
        <code style={{ margin: '0 4px' }}>{latest}</code>
        {t('upgradeBanner.upgradeLabel')}
        <code style={{ margin: '0 6px' }}>{hint}</code>
      </span>
      <span className="banner-spacer" />
      <button
        type="button"
        className="banner-close"
        onClick={dismissForAWeek}
        aria-label={t('upgradeBanner.dismissAria')}
        title={t('upgradeBanner.dismissTitle')}
      >
        <Icon name="x" />
      </button>
    </div>
  )
}
