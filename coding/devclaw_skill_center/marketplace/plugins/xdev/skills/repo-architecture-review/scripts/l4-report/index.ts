import type { AnalyzeResult } from '../orchestrator.js';
import { renderSummary } from './sections/summary.js';
import { renderFindings } from './sections/findings.js';
import { renderActionRoadmap } from './sections/action-roadmap.js';
import { renderMetrics } from './sections/metrics.js';
import { renderKnownIssues } from './sections/known-issues.js';
import { renderAudit } from './sections/audit.js';

export function renderReport(r: AnalyzeResult): string {
  // Report structure:
  // 1. Dashboard summary — quick health overview, top problems
  // 2. Action roadmap — prioritized table of what to do first
  // 3. Detailed findings — full narrative per issue
  // 4. Panoramic metrics — repo-wide stats
  // 5. Known issues (if incremental)
  // 6. Execution audit — tool status and timing
  return [
    renderSummary(r),
    renderActionRoadmap(r.reportedFindings),
    renderFindings(r.reportedFindings),
    renderMetrics(r),
    renderKnownIssues(r.knownUnchanged),
    renderAudit(r),
  ].join('\n');
}
