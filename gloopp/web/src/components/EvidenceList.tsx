import type { ReportEvidenceRef } from '../api/types'
import type { NativeToolItem } from '../features/quest/traceModel'
import { fmtTime } from './util'
import Icon from './Icon'

type Props = {
  reportEvidence?: ReportEvidenceRef[]
  nativeEvidence?: NativeToolItem[]
  onJumpToTrace?: () => void
}

export default function EvidenceList({ reportEvidence = [], nativeEvidence = [], onJumpToTrace }: Props) {
  const hasReportEvidence = reportEvidence.length > 0
  const hasTraceEvidence = nativeEvidence.length > 0
  if (!hasReportEvidence && !hasTraceEvidence) return null

  return (
    <section id="quest-evidence" className="panel evidence-list-panel mt-3">
      <div className="panel-title">
        <h2>Evidence</h2>
        <span className="mono">
          {reportEvidence.length} report refs · {nativeEvidence.length} trace tools
        </span>
      </div>

      {hasReportEvidence && (
        <div className="evidence-list-section">
          <div className="evidence-section-label">Report Evidence</div>
          <div className="evidence-list">
            {reportEvidence.map((item) => (
              <article className="evidence-card" key={item.id}>
                <div className="evidence-card-head">
                  <span className="report-chip">{item.kind}</span>
                  <span className="report-chip">{item.trust_tier}</span>
                  <span className="mono push">{fmtTime(item.created_at_ms)}</span>
                </div>
                <div className="evidence-card-main">
                  <strong>{item.source || item.id}</strong>
                  {item.command_id && <span className="mono">command {item.command_id}</span>}
                </div>
                <div className="evidence-card-meta">
                  {typeof item.exit_code === 'number' && <span>exit {item.exit_code}</span>}
                  {typeof item.duration_ms === 'number' && <span>{item.duration_ms}ms</span>}
                  {item.timed_out && <span>timed out</span>}
                  {item.output_path && <span className="mono">{item.output_path}</span>}
                </div>
                {item.snippet && <pre className="evidence-snippet">{item.snippet}</pre>}
              </article>
            ))}
          </div>
        </div>
      )}

      {hasTraceEvidence && (
        <div className="evidence-list-section">
          <div className="evidence-section-label">Trace Evidence</div>
          <div className="evidence-trace-strip">
            {nativeEvidence.slice(0, 6).map((item, index) => (
              <button type="button" className="evidence-trace-chip" onClick={onJumpToTrace} key={item.sid + ':' + item.row.seq + ':' + index}>
                <Icon name="search" size={12} />
                <span>{item.row.tool_name || 'tool'}</span>
                <em className="mono">{item.status}</em>
              </button>
            ))}
            {nativeEvidence.length > 6 && (
              <button type="button" className="evidence-trace-chip muted" onClick={onJumpToTrace}>
                +{nativeEvidence.length - 6} more
              </button>
            )}
          </div>
        </div>
      )}
    </section>
  )
}
