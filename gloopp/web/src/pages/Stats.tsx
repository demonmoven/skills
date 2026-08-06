import { useMemo } from 'react'
import type { PersonalDrilldown, PersonalStats } from '../api/types'
import Icon from '../components/Icon'
import i18n from '../i18n'
import { useTranslation } from 'react-i18next'

type Range = 'today' | '7d' | '30d' | 'all'

type Props = {
  stats: PersonalStats | null
  loading?: boolean
  range: Range
  onOpenQuest: (qid: string) => void
  onDrilldown: (drilldown: PersonalDrilldown, label?: string) => void
}

function StatTile({
  label,
  value,
  sub,
  icon,
  tone,
}: {
  label: string
  value: string | number
  sub?: string
  icon: string
  tone?: 'ok' | 'warn' | 'bad' | ''
}) {
  return (
    <div className="stat-tile panel">
      <div className="stat-tile-head">
        <span className="stat-tile-icon">
          <Icon name={icon} size={16} />
        </span>
        <span className="eyebrow">{label}</span>
      </div>
      <div className={'stat-tile-value mono ' + (tone || '')}>{value}</div>
      {sub && <div className="stat-tile-sub mono">{sub}</div>}
    </div>
  )
}

function BarRow({
  label,
  value,
  total,
  tone,
  sub,
  drilldown,
  onDrilldown,
}: {
  label: string
  value: number
  total: number
  tone?: string
  sub?: string
  drilldown?: PersonalDrilldown
  onDrilldown?: (drilldown: PersonalDrilldown, label?: string) => void
}) {
  const pct = total > 0 ? Math.min(100, Math.round((value / total) * 100)) : 0
  const clickable = !!drilldown && !!onDrilldown
  const content = (
    <>
      <div className="bar-row-head">
        <span>{label}</span>
        <span className="mono">
          {value}
          {sub ? <small className="muted"> · {sub}</small> : null}
        </span>
      </div>
      <div className="bar-row-track">
        <i className={tone || ''} style={{ width: `${pct}%` }} />
      </div>
    </>
  )
  if (clickable) {
    return (
      <button
        className="bar-row is-clickable"
        type="button"
        onClick={() => onDrilldown(drilldown, label)}
      >
        {content}
      </button>
    )
  }
  return (
    <div className="bar-row">
      {content}
    </div>
  )
}

function EmptyMetric({ label }: { label: string }) {
  return <div className="muted empty-inline">{label}</div>
}

function InsightRow({
  icon,
  label,
  value,
  tone,
  drilldown,
  onDrilldown,
}: {
  icon: string
  label: string
  value: string
  tone?: 'ok' | 'warn' | 'bad' | ''
  drilldown?: PersonalDrilldown
  onDrilldown?: (drilldown: PersonalDrilldown, label?: string) => void
}) {
  const content = (
    <>
      <span className="stats-insight-icon">
        <Icon name={icon} size={14} />
      </span>
      <span>{label}</span>
      <strong className="mono">{value}</strong>
    </>
  )
  if (drilldown && onDrilldown) {
    return (
      <button
        className={'stats-insight-row is-clickable ' + (tone || '')}
        type="button"
        onClick={() => onDrilldown(drilldown, label)}
      >
        {content}
      </button>
    )
  }
  return (
    <div className={'stats-insight-row ' + (tone || '')}>
      {content}
    </div>
  )
}

function humanDuration(ms: number | null | undefined): string {
  if (!ms || ms <= 0) return '—'
  const s = Math.max(1, Math.round(ms / 1000))
  if (s < 60) return i18n.t('stats.duration.second', { val: s })
  const m = Math.floor(s / 60)
  const r = s % 60
  if (m < 60) return r ? i18n.t('stats.duration.minSec', { min: m, sec: r }) : i18n.t('stats.duration.min', { val: m })
  const h = Math.floor(m / 60)
  const rm = m % 60
  return rm ? i18n.t('stats.duration.hourMin', { hour: h, min: rm }) : i18n.t('stats.duration.hour', { val: h })
}

function formatRelative(ts: number | null | undefined): string {
  if (!ts) return '—'
  const diff = Date.now() - ts
  const abs = Math.abs(diff)
  const m = Math.floor(abs / 60000)
  if (m < 1) return i18n.t('stats.relative.justNow')
  if (m < 60) return i18n.t('stats.relative.minutesAgo', { val: m })
  const h = Math.floor(m / 60)
  if (h < 24) return i18n.t('stats.relative.hoursAgo', { val: h })
  const d = Math.floor(h / 24)
  if (d < 30) return i18n.t('stats.relative.daysAgo', { val: d })
  return new Date(ts).toLocaleDateString()
}

function pct(value: unknown): string {
  const n = Number(value || 0)
  if (!Number.isFinite(n) || n <= 0) return '0%'
  return `${Math.round(n * 100)}%`
}

export default function Stats({ stats, loading, range, onOpenQuest, onDrilldown }: Props) {
  const { t } = useTranslation()
  const s = (stats || {}) as Record<string, any>

  const data = useMemo(() => {
    const statusCounts = (s.status_counts || {}) as Record<string, unknown>
    const total = Number(s.total_quests ?? s.total_runs ?? s.quests ?? 0)
    const success = Number(s.success_quests ?? s.succeeded ?? statusCounts.success ?? 0)
    const failed = Number(s.failed_quests ?? s.failed ?? statusCounts.failed ?? 0)
    const applied = Number(s.applied_quests ?? s.applied ?? 0)
    const cancelled = Number(s.cancelled_quests ?? s.cancelled ?? statusCounts.cancelled ?? 0)
    const reviewing = Number(s.reviewing_quests ?? s.reviewing ?? statusCounts.reviewing ?? 0)
    const running = Number(s.running_quests ?? s.running ?? statusCounts.running ?? 0)
    const pending = Number(s.pending_quests ?? s.pending ?? statusCounts.pending ?? 0)
    const userReview = Number(s.user_review_quests ?? s.user_review ?? statusCounts.user_review ?? 0)
    const blocked = Number(s.blocked_quests ?? s.blocked ?? statusCounts.blocked ?? 0)

    const totalTurns = Number(s.total_turns ?? s.tool_calls ?? 0) || 0
    const autoClosed = Number(s.auto_closed_quests ?? 0) || 0
    const successRate = total > 0 ? Math.round((success / total) * 100) : 0
    const autoCloseRate = success > 0 ? Math.round((autoClosed / success) * 100) : 0
    const p95Ms = Number(s.p95_duration_ms ?? s.p95_ms ?? 0) || 0
    const p50Ms = Number(s.p50_duration_ms ?? s.p50_ms ?? 0) || 0
    const tokens = Number(s.total_tokens ?? s.tokens ?? 0) || 0
    const tokenInput = Number(s.input_tokens ?? s.total_tokens_in ?? 0) || 0
    const avgTurnsPerRun =
      total > 0 && totalTurns > 0 ? (totalTurns / total).toFixed(1) : null

    return {
      total,
      success,
      failed,
      applied,
      cancelled,
      reviewing,
      running,
      pending,
      userReview,
      blocked,
      totalTurns,
      autoClosed,
      successRate,
      autoCloseRate,
      p95Ms,
      p50Ms,
      tokens,
      tokenInput,
      avgTurnsPerRun,
    }
  }, [s])

  const statusEntries: { label: string; value: number; tone: string; drilldown: PersonalDrilldown }[] = [
    { label: t('stats.status.success'), value: data.success, tone: 'ok', drilldown: { query: { status: ['success'] } } },
    { label: t('stats.status.running'), value: data.running, tone: 'ink', drilldown: { query: { status: ['running'] } } },
    { label: t('stats.status.reviewing'), value: data.reviewing, tone: 'violet', drilldown: { query: { status: ['reviewing'] } } },
    { label: t('stats.status.userReview'), value: data.userReview, tone: 'amber', drilldown: { query: { status: ['user_review'] } } },
    { label: t('stats.status.blocked'), value: data.blocked, tone: 'amber', drilldown: { query: { status: ['blocked'] } } },
    { label: t('stats.status.failed'), value: data.failed, tone: 'bad', drilldown: { query: { status: ['failed'] } } },
    { label: t('stats.status.cancelled'), value: data.cancelled, tone: 'ink', drilldown: { query: { status: ['cancelled'] } } },
    { label: t('stats.status.pending'), value: data.pending, tone: 'ink', drilldown: { query: { status: ['pending'] } } },
  ].filter((r) => r.value > 0)

  const statusTotal = Math.max(1, statusEntries.reduce((acc, r) => acc + r.value, 0))

  const phaseStats = s.phase_stats || s.phase_time_ms
  const phaseEntries: { label: string; value: number; tone: string }[] = []
  if (Array.isArray(phaseStats)) {
    const tonesByPhase: Record<string, string> = {
      warrior: 'amber',
      mage: 'violet',
      mage_review: 'violet',
      user_review: 'ink',
      apply: 'ok',
      rework: 'bad',
    }
    for (const row of phaseStats) {
      const label = String(row.phase || row.name || row.label || t('stats.phase.unknown'))
      const v = Number(row.total_ms ?? row.avg_ms ?? row.p95_ms ?? row.value ?? 0)
      if (v > 0) {
        phaseEntries.push({
          label,
          value: Math.round(v / 1000),
          tone: tonesByPhase[label] || 'ink',
        })
      }
    }
  } else if (phaseStats && typeof phaseStats === 'object') {
    const order: [string, string, string][] = [
      ['warrior', t('stats.phase.warrior'), 'amber'],
      ['mage', t('stats.phase.mage'), 'violet'],
      ['user_review', t('stats.phase.userReview'), 'ink'],
      ['apply', t('stats.phase.apply'), 'ok'],
      ['rework', t('stats.phase.rework'), 'bad'],
    ]
    let max = 0
    for (const [key, label, tone] of order) {
      const v = Number(phaseStats[key + '_ms'] ?? phaseStats[key] ?? 0)
      if (v > 0) {
        const sec = Math.round(v / 1000)
        phaseEntries.push({ label, value: sec, tone })
      }
    }
  }
  const phaseTotal = Math.max(1, phaseEntries.reduce((a, b) => a + b.value, 0))

  const failureRaw = s.failure_reasons || s.failure_stats
  const failureEntries: { label: string; value: number; tone: string; drilldown?: PersonalDrilldown }[] = []
  if (Array.isArray(failureRaw)) {
    for (const r of failureRaw) {
      const label = r.reason || r.label || r.name || t('stats.phase.unknown')
      const v = Number(r.count ?? r.value ?? 0)
      if (v > 0) failureEntries.push({ label, value: v, tone: v > 5 ? 'bad' : 'amber', drilldown: r.drilldown })
    }
  } else if (failureRaw && typeof failureRaw === 'object') {
    for (const [k, v] of Object.entries(failureRaw)) {
      const n = Number(v)
      if (n > 0) failureEntries.push({ label: k, value: n, tone: n > 5 ? 'bad' : 'amber', drilldown: { query: { failure_reason: [k] } } })
    }
  }
  const failureTotal = Math.max(1, failureEntries.reduce((a, b) => a + b.value, 0))

  const failureCategoryRaw = s.failure_categories
  const failureCategoryEntries: { label: string; value: number; tone: string; drilldown?: PersonalDrilldown }[] = []
  if (Array.isArray(failureCategoryRaw)) {
    for (const r of failureCategoryRaw) {
      const label = r.name || r.category || r.label || t('stats.phase.unknown')
      const v = Number(r.count ?? r.value ?? 0)
      if (v > 0) failureCategoryEntries.push({ label, value: v, tone: v > 5 ? 'bad' : 'amber', drilldown: r.drilldown })
    }
  } else if (failureCategoryRaw && typeof failureCategoryRaw === 'object') {
    for (const [k, v] of Object.entries(failureCategoryRaw)) {
      const n = Number(v)
      if (n > 0) failureCategoryEntries.push({ label: k, value: n, tone: n > 5 ? 'bad' : 'amber', drilldown: { query: { failure_category: [k] } } })
    }
  }
  const failureCategoryTotal = Math.max(1, failureCategoryEntries.reduce((a, b) => a + b.value, 0))

  const projectRaw = s.project_stats || s.projects || s.repos || s.by_project
  const projectEntries: { label: string; value: number; tone: string; drilldown?: PersonalDrilldown }[] = []
  const tones = ['ink', 'violet', 'amber', 'ok', 'bad']
  if (Array.isArray(projectRaw)) {
    projectRaw.slice(0, 8).forEach((r, i) => {
      const label = r.name || r.project || r.repo || r.label
      const v = Number(r.count ?? r.value ?? 0)
      if (v > 0) projectEntries.push({ label, value: v, tone: tones[i % tones.length], drilldown: r.drilldown })
    })
  } else if (projectRaw && typeof projectRaw === 'object') {
    let i = 0
    for (const [k, v] of Object.entries(projectRaw)) {
      const n = Number(v)
      if (n > 0) {
        projectEntries.push({ label: k, value: n, tone: tones[i % tones.length], drilldown: { query: { project: [k] } } })
        i++
      }
    }
  }
  const projectTotal = Math.max(1, projectEntries.reduce((a, b) => a + b.value, 0))

  const slowRaw = s.slow_ops || s.slow_operations
  const slowRows: { op: string; p95: string; count: number; drilldown?: PersonalDrilldown }[] = []
  if (Array.isArray(slowRaw)) {
    for (const r of slowRaw) {
      const op = r.op || r.operation || r.name || r.kind
      const p95 = r.p95 || humanDuration(Number(r.p95_ms ?? r.duration_ms ?? 0))
      const count = Number(r.count ?? r.runs ?? 0)
      if (op) slowRows.push({ op, p95, count, drilldown: r.drilldown })
    }
  }

  const recentRaw: any[] = s.recent_runs || s.recent_quests || []
  const recentRows: { q: string; status: string; when: string; id?: string }[] = []
  if (Array.isArray(recentRaw)) {
    for (const r of recentRaw.slice(0, 8)) {
      const q = r.query || r.title || r.name
      const status = (r.status || (r.final_verdict === 'pass' ? 'success' : r.final_verdict === 'reject' ? 'failed' : r.completed_at_ms ? 'success' : 'running')) as string
      const when = formatRelative(r.completed_at_ms ?? r.updated_at_ms ?? r.started_at_ms ?? r.created_at_ms)
      if (!q) continue
      recentRows.push({ q, status, when, id: r.quest_id || r.id })
    }
  }

  const rangeLabel: Record<Range, string> = {
    today: t('stats.range.today'),
    '7d': t('stats.range.7d'),
    '30d': t('stats.range.30d'),
    all: t('stats.range.all'),
  }
  const attention = data.failed + data.blocked + data.userReview
  const active = data.running + data.reviewing + data.userReview + data.pending
  const healthTone =
    data.blocked > 0 || data.failed > 0 || (data.total > 0 && data.successRate < 50)
      ? 'bad'
      : data.userReview > 0 || data.reviewing > 0 || (data.total > 0 && data.successRate < 80)
        ? 'warn'
        : data.total > 0
          ? 'ok'
          : ''
  const healthTitle =
    healthTone === 'bad'
      ? t('stats.health.needsAttention')
      : healthTone === 'warn'
        ? t('stats.health.needsConfirm')
        : healthTone === 'ok'
          ? t('stats.health.stable')
          : t('stats.health.waiting')
  const healthSummary =
    data.total === 0
      ? t('stats.health.summaryEmpty')
      : attention > 0
        ? t('stats.health.summaryAttention', { count: attention })
        : active > 0
          ? t('stats.health.summaryActive', { count: active, rate: data.successRate })
          : t('stats.health.summaryStable', { count: data.total, rate: data.successRate })
  const topFailure = failureEntries[0]
  const topProject = projectEntries[0]
  const topSlow = slowRows[0]
  const anomalySignals = Array.isArray(s.anomaly_signals) ? s.anomaly_signals : []
  const signalByKey = new Map(anomalySignals.map((item: any) => [String(item.key), item]))
  const loopHealth = (s.loop_health || {}) as Record<string, unknown>

  return (
    <div className="stats-page">
      {loading && !stats && (
        <div className="panel">
          <EmptyMetric label={t('stats.empty.loading')} />
        </div>
      )}

      <section className={'stats-health panel ' + healthTone}>
        <div className="stats-health-main">
          <span className="eyebrow">
            {t('stats.health.eyebrow')} · {rangeLabel[range]}
            {loading && stats && <Icon name="spinner" className="spin" size={12} />}
          </span>
          <h2>{healthTitle}</h2>
          <p>{healthSummary}</p>
        </div>
        <div className="stats-health-signals">
          <InsightRow
            icon="shield-alert"
            label={t('stats.insight.attention')}
            value={t('stats.insight.attentionCount', { count: attention })}
            tone={attention > 0 ? 'bad' : 'ok'}
            drilldown={{ query: { status: ['blocked', 'failed', 'user_review'] } }}
            onDrilldown={onDrilldown}
          />
          <InsightRow
            icon="loader"
            label={t('stats.insight.active')}
            value={t('stats.insight.attentionCount', { count: active })}
            tone={active > 0 ? 'warn' : ''}
          />
          <InsightRow
            icon="timer"
            label={t('stats.insight.slowestP95')}
            value={topSlow?.p95 || humanDuration(data.p95Ms)}
            tone={topSlow || data.p95Ms ? 'warn' : ''}
          />
        </div>
      </section>

      <div className="stats-hero">
        <StatTile
          label={t('stats.tile.runs')}
          value={data.totalTurns || data.total || 0}
          sub={
            data.avgTurnsPerRun
              ? t('stats.tile.avgTurns', { avg: data.avgTurnsPerRun })
              : t('stats.tile.rangeLabel', { range })
          }
          icon="clipboard-check"
        />
        <StatTile
          label={t('stats.tile.successRate')}
          value={`${data.successRate}%`}
          sub={`${data.success} / ${data.total || 0}`}
          icon="circle-check"
          tone={data.successRate >= 80 ? 'ok' : data.successRate >= 50 ? 'warn' : 'bad'}
        />
        <StatTile
          label={t('stats.tile.autoClose')}
          value={`${data.autoCloseRate}%`}
          sub={t('stats.tile.autoClosedSub', { count: data.autoClosed, total: data.success })}
          icon="zap"
          tone={data.autoCloseRate >= 70 ? 'ok' : data.autoCloseRate >= 40 ? 'warn' : 'bad'}
        />
        <StatTile
          label={t('stats.tile.p95')}
          value={data.p95Ms ? humanDuration(data.p95Ms) : '—'}
          sub={data.p50Ms ? t('stats.tile.p50', { val: humanDuration(data.p50Ms) }) : t('stats.tile.p95Range')}
          icon="clock"
        />
        <StatTile
          label={t('stats.tile.tokens')}
          value={data.tokens > 1_000_000 ? (data.tokens / 1_000_000).toFixed(2) + 'M' : data.tokens > 1_000 ? (data.tokens / 1000).toFixed(0) + 'K' : data.tokens}
          sub={data.tokenInput ? t('stats.tile.tokenInput', { val: (data.tokenInput / 1000).toFixed(0) }) : t('stats.tile.tokenTotal')}
          icon="code"
        />
      </div>

      <section className="panel stats-loop-health">
        <div className="panel-title">
          <h2>{t('stats.section.loopHealth')}</h2>
          <span>{t('stats.section.loopHealthSubtitle')}</span>
        </div>
        <div className="loop-health-grid">
          <InsightRow
            icon="radio"
            label={t('term.firstOutputCoverage')}
            value={pct(loopHealth.first_output_coverage)}
            tone={Number(loopHealth.first_output_coverage || 0) >= 0.8 ? 'ok' : 'warn'}
          />
          <InsightRow
            icon="file-text"
            label={t('term.makerReportCoverage')}
            value={pct(loopHealth.maker_report_coverage)}
            tone={Number(loopHealth.maker_report_coverage || 0) >= 0.8 ? 'ok' : 'warn'}
          />
          <InsightRow
            icon="shield"
            label={t('term.reviewReportCoverage')}
            value={pct(loopHealth.review_report_coverage)}
            tone={Number(loopHealth.review_report_coverage || 0) >= 0.8 ? 'ok' : 'warn'}
          />
          <InsightRow
            icon="search"
            label={t('term.evidenceBackedReview')}
            value={pct(loopHealth.evidence_backed_review_coverage)}
            tone={Number(loopHealth.evidence_backed_review_coverage || 0) >= 0.8 ? 'ok' : 'warn'}
          />
          <InsightRow
            icon="inbox"
            label={t('term.humanExceptions')}
            value={String(loopHealth.human_exception_open_count || 0)}
            tone={Number(loopHealth.human_exception_open_count || 0) > 0 ? 'warn' : 'ok'}
            drilldown={{ query: { status: ['blocked', 'waiting_input', 'user_review'] } }}
            onDrilldown={onDrilldown}
          />
          <InsightRow
            icon="timer"
            label={t('term.lateFailures')}
            value={String(loopHealth.late_failure_count || 0)}
            tone={Number(loopHealth.late_failure_count || 0) > 0 ? 'bad' : 'ok'}
            drilldown={{ query: { status: ['failed', 'blocked'] } }}
            onDrilldown={onDrilldown}
          />
          <InsightRow
            icon="bolt"
            label={t('term.noFindingRate')}
            value={pct(loopHealth.automation_no_finding_rate)}
            tone={Number(loopHealth.automation_no_finding_rate || 0) > 0.5 ? 'warn' : 'ok'}
          />
          <InsightRow
            icon="triangle-alert"
            label={t('term.tierCUsage')}
            value={pct(loopHealth.tier_c_usage)}
            tone={Number(loopHealth.tier_c_usage || 0) > 0 ? 'warn' : 'ok'}
          />
        </div>
      </section>

      <div className="stats-workbench">
        <main className="stats-diagnostics">
          <div className="stats-section-head">
            <div>
              <span className="eyebrow">{t('stats.section.diagnosis')}</span>
              <h2>{t('stats.section.bottleneck')}</h2>
            </div>
            <span className="mono">{t('stats.section.sampleCount', { count: statusTotal })}</span>
          </div>

          <div className="stats-layout">
            <section className="panel stats-card stats-card-primary">
              <div className="panel-title">
                <h2>{t('stats.section.statusDist')}</h2>
                <span className="mono">{t('stats.section.runCount', { count: statusTotal })}</span>
              </div>
              <div className="bar-list">
                {statusEntries.length === 0 ? (
                  <EmptyMetric label={t('stats.empty.noStatus')} />
                ) : (
                  statusEntries.map((r) => (
                    <BarRow
                      key={r.label}
                      label={r.label}
                      value={r.value}
                      total={statusTotal}
                      tone={r.tone}
                      drilldown={r.drilldown}
                      onDrilldown={onDrilldown}
                    />
                  ))
                )}
              </div>
            </section>

            <section className="panel stats-card">
              <div className="panel-title">
                <h2>{t('stats.section.phaseTime')}</h2>
                <span>{t('stats.section.cumulativeSec')}</span>
              </div>
              <div className="bar-list">
                {phaseEntries.length === 0 ? (
                  <EmptyMetric label={t('stats.empty.noPhaseTime')} />
                ) : (
                  phaseEntries.map((r) => (
                    <BarRow
                      key={r.label}
                      label={r.label}
                      value={r.value}
                      total={phaseTotal}
                      tone={r.tone}
                    />
                  ))
                )}
              </div>
            </section>

            <section className="panel stats-card">
              <div className="panel-title">
                <h2>{t('stats.section.failureReason')}</h2>
                <span>{topFailure ? topFailure.label : t('stats.section.topReason')}</span>
              </div>
              <div className="bar-list">
                {failureEntries.length === 0 ? (
                  <EmptyMetric label={t('stats.empty.noFailureReason')} />
                ) : (
                  failureEntries.map((r) => (
                    <BarRow
                      key={r.label}
                      label={r.label}
                      value={r.value}
                      total={failureTotal}
                      tone={r.tone}
                      drilldown={r.drilldown}
                      onDrilldown={onDrilldown}
                    />
                  ))
                )}
              </div>
            </section>

            <section className="panel stats-card">
              <div className="panel-title">
                <h2>{t('stats.section.failureCategory')}</h2>
                <span>{topFailure ? topFailure.label : t('stats.section.topCategory')}</span>
              </div>
              <div className="bar-list">
                {failureCategoryEntries.length === 0 ? (
                  <EmptyMetric label={t('stats.empty.noFailureCategory')} />
                ) : (
                  failureCategoryEntries.map((r) => (
                    <BarRow
                      key={r.label}
                      label={r.label}
                      value={r.value}
                      total={failureCategoryTotal}
                      tone={r.tone}
                      drilldown={r.drilldown}
                      onDrilldown={onDrilldown}
                    />
                  ))
                )}
              </div>
            </section>

            <section className="panel stats-card">
              <div className="panel-title">
                <h2>{t('stats.section.projectDist')}</h2>
                <span>{topProject ? topProject.label : t('stats.section.byRepo')}</span>
              </div>
              <div className="bar-list">
                {projectEntries.length === 0 ? (
                  <EmptyMetric label={t('stats.empty.noProject')} />
                ) : (
                  projectEntries.map((r) => (
                    <BarRow
                      key={r.label}
                      label={r.label}
                      value={r.value}
                      total={projectTotal}
                      tone={r.tone}
                      drilldown={r.drilldown}
                      onDrilldown={onDrilldown}
                    />
                  ))
                )}
              </div>
            </section>
          </div>
        </main>

        <aside className="stats-insight-rail">
          <section className="panel stats-risk-panel">
            <div className="panel-title">
              <h2>{t('stats.section.anomalyPriority')}</h2>
              <span>{attention > 0 ? t('stats.section.signalCount', { count: attention }) : t('stats.section.noAnomaly')}</span>
            </div>
            <div className="stats-risk-list">
              <InsightRow
                icon="shield-alert"
                label={t('stats.section.blockedQuests')}
                value={`${data.blocked}`}
                tone={data.blocked > 0 ? 'bad' : 'ok'}
                drilldown={signalByKey.get('blocked')?.drilldown || { query: { status: ['blocked'] } }}
                onDrilldown={onDrilldown}
              />
              <InsightRow
                icon="x"
                label={t('stats.section.failedQuests')}
                value={`${data.failed}`}
                tone={data.failed > 0 ? 'bad' : 'ok'}
                drilldown={signalByKey.get('failed')?.drilldown || { query: { status: ['failed'] } }}
                onDrilldown={onDrilldown}
              />
              <InsightRow
                icon="circle-check"
                label={t('stats.section.pendingReview')}
                value={`${data.userReview}`}
                tone={data.userReview > 0 ? 'warn' : ''}
                drilldown={signalByKey.get('user_review')?.drilldown || { query: { status: ['user_review'] } }}
                onDrilldown={onDrilldown}
              />
              <InsightRow
                icon="clock"
                label={t('stats.section.topFailureReason')}
                value={topFailure?.label || '—'}
                tone={topFailure ? 'warn' : ''}
                drilldown={topFailure?.drilldown}
                onDrilldown={onDrilldown}
              />
            </div>
          </section>

          <section className="panel stats-list-panel">
            <div className="panel-title">
              <h2>{t('stats.section.slowOps')}</h2>
              <span>P95</span>
            </div>
            <div className="slow-table">
              {slowRows.length === 0 ? (
                <EmptyMetric label={t('stats.empty.noSlowOps')} />
              ) : (
                slowRows.map((r, i) => (
                  <div
                    className={'slow-row' + (r.drilldown ? ' is-clickable' : '')}
                    key={r.op + i}
                    role={r.drilldown ? 'button' : undefined}
                    tabIndex={r.drilldown ? 0 : undefined}
                    onClick={() => r.drilldown && onDrilldown(r.drilldown, r.op)}
                    onKeyDown={(event) => {
                      if (!r.drilldown) return
                      if (event.key === 'Enter' || event.key === ' ') {
                        event.preventDefault()
                        onDrilldown(r.drilldown, r.op)
                      }
                    }}
                  >
                    <span className="mono">{r.op}</span>
                    <span className="mono">{r.p95}</span>
                    <span className="mono">{t('stats.slowOps.count', { val: r.count })}</span>
                  </div>
                ))
              )}
            </div>
          </section>

          <section className="panel stats-list-panel">
            <div className="panel-title">
              <h2>{t('stats.section.recentRuns')}</h2>
              <span>{t('stats.section.recentCount', { count: recentRows.length })}</span>
            </div>
            <div className="stats-recent-runs">
              <div className="stats-run-list">
                {recentRows.length === 0 ? (
                  <EmptyMetric label={t('stats.empty.noRuns')} />
                ) : recentRows.map((r, i) => {
                  const tone =
                    r.status === 'success' || r.status === 'applied'
                      ? 'success'
                      : r.status === 'failed' || r.status === 'reject'
                        ? 'failed'
                        : r.status === 'blocked'
                          ? 'pending'
                          : r.status === 'reviewing' || r.status === 'user_review'
                            ? 'pending'
                            : 'pending'
                  const icon =
                    r.status === 'success' || r.status === 'applied'
                      ? 'circle-check'
                      : r.status === 'failed' || r.status === 'reject'
                        ? 'x'
                        : r.status === 'blocked'
                          ? 'shield-alert'
                          : 'loader'
                  return (
                    <div
                      className="stats-run-item"
                      key={r.id || `${r.q}-${i}`}
                      role="button"
                      tabIndex={0}
                      onClick={() => r.id && onOpenQuest(r.id)}
                      onKeyDown={(event) => {
                        if (!r.id) return
                        if (event.key === 'Enter' || event.key === ' ') {
                          event.preventDefault()
                          onOpenQuest(r.id)
                        }
                      }}
                    >
                      <Icon
                        name={icon}
                        className={
                          'icon ' +
                          (tone === 'success'
                            ? 'success'
                            : tone === 'failed'
                              ? 'failed'
                              : 'pending')
                        }
                      />
                      <span className="title">{r.q}</span>
                      <span className="time">{r.when}</span>
                    </div>
                  )
                })}
              </div>
            </div>
          </section>
        </aside>
      </div>
    </div>
  )
}
