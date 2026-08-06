import type { LoopStateSpine } from '../api/types'
import AgentMessageRenderer from './AgentMessageRenderer'

type Props = {
  spine: LoopStateSpine
}

export default function LoopStateCard({ spine }: Props) {
  return (
    <section id="quest-loop-state" className="panel loop-state-card mt-3">
      <div className="panel-title">
        <h2>Loop State</h2>
        <span className="mono">{spine.schema_version}</span>
      </div>
      <div className="review-meta-row">
        {spine.current_phase && <span className="report-chip">phase: {spine.current_phase}</span>}
        {spine.attempts?.length ? <span className="report-chip">attempts: {spine.attempts.length}</span> : null}
        {spine.confirmed_facts?.length ? <span className="report-chip">facts: {spine.confirmed_facts.length}</span> : null}
        {spine.failed_paths?.length ? <span className="report-chip risk">failed paths: {spine.failed_paths.length}</span> : null}
        {spine.open_blockers?.length ? <span className="report-chip required">blockers: {spine.open_blockers.length}</span> : null}
      </div>
      {spine.next_expected_action && (
        <AgentMessageRenderer
          content={spine.next_expected_action}
          className="loop-report-body"
          maxLength={1200}
        />
      )}
      {(spine.confirmed_facts?.length || spine.failed_paths?.length || spine.open_blockers?.length) ? (
        <div className="review-report-grid">
          {spine.confirmed_facts?.length ? (
            <div className="loop-report-list checked">
              <strong>Confirmed Facts</strong>
              <ul>{spine.confirmed_facts.map((item, index) => <li key={index}>{item}</li>)}</ul>
            </div>
          ) : null}
          {spine.failed_paths?.length ? (
            <div className="loop-report-list risk">
              <strong>Failed Paths</strong>
              <ul>{spine.failed_paths.map((item, index) => <li key={index}>{item}</li>)}</ul>
            </div>
          ) : null}
          {spine.open_blockers?.length ? (
            <div className="loop-report-list required">
              <strong>Open Blockers</strong>
              <ul>{spine.open_blockers.map((item, index) => <li key={index}>{item}</li>)}</ul>
            </div>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}
