import pLimit from 'p-limit';
import type { Violation, Finding } from '../types.js';
import { groupViolations, type Group } from './grouping.js';
import { crossSignals, type Crossed } from './signal-crossing.js';
import { judgeGroup, shouldJudge } from './llm-judge.js';
import { fuseGroup } from './fuser.js';
import { isMetricHardEvidence, renderMetricHardEvidenceFinding } from './metric-narrative.js';

export interface FuseOptions {
  repoRoot: string;
  concurrency: number;
  selfConsistencyRounds: number;
  llmOptional: boolean;     // if true, skip LLM steps entirely (for fast tests)
}

/** Serialisable representation of a group pending external LLM processing. */
export interface PendingLlmGroup {
  id: string;
  category: string;
  commonFile?: string;
  commonDir?: string;
  violations: Violation[];
  crossed: {
    independentSources: string[];
    baseConfidence: 'high' | 'medium' | 'low';
  };
}

/** Result of separating violations into deterministic findings and LLM-pending groups. */
export interface PreparedGroups {
  /** Findings that can be rendered deterministically from metric hard-evidence. */
  deterministicFindings: Finding[];
  /** Groups that need LLM judge + fuser to produce findings. */
  pendingLlmGroups: PendingLlmGroup[];
}

/**
 * Separate violations into two buckets:
 * 1. Metric-hard-evidence groups → rendered as deterministic findings (no LLM needed)
 * 2. All other groups → serialised for external LLM processing (by the agent)
 */
export function prepareGroupsForExternalLlm(allViolations: Violation[]): PreparedGroups {
  const groups = groupViolations(allViolations);
  const deterministicFindings: Finding[] = [];
  const pendingLlmGroups: PendingLlmGroup[] = [];

  for (const g of groups) {
    const crossed = crossSignals(g);
    if (isMetricHardEvidence(g)) {
      deterministicFindings.push(renderMetricHardEvidenceFinding(g));
    } else {
      pendingLlmGroups.push({
        id: g.id,
        category: g.category,
        commonFile: g.commonFile,
        commonDir: g.commonDir,
        violations: g.violations,
        crossed: {
          independentSources: crossed.independentSources,
          baseConfidence: crossed.baseConfidence,
        },
      });
    }
  }

  return { deterministicFindings, pendingLlmGroups };
}

export async function fuseFindings(allViolations: Violation[], opts: FuseOptions): Promise<Finding[]> {
  const groups = groupViolations(allViolations);
  const limit = pLimit(opts.concurrency);
  const findings = await Promise.all(groups.map((g) => limit(async () => {
    const crossed = crossSignals(g);
    if (opts.llmOptional) {
      const repr = g.violations[0];
      return {
        id: g.id,
        category: g.category,
        title: repr?.message ?? `${g.category} issue`,
        rootCause: (repr?.evidence?.signals ?? []).join('; '),
        impact: '',
        actions: [],
        confidence: crossed.baseConfidence,
        confidenceScore: crossed.baseConfidence === 'high' ? 0.9 : crossed.baseConfidence === 'medium' ? 0.6 : 0.3,
        severity: repr?.severity ?? 'low',
        signals: g.violations.map((v) => ({ source: v.source, description: v.message })),
        locations: g.violations.flatMap((v) => v.locations),
        violationIds: g.violations.map((v) => v.id),
      } satisfies Finding;
    }
    // Max-coverage mode: judge every group and never drop it, even if the LLM
    // voted "noise". The verdict is still attached to the judge object so it
    // can be shown in the report for the reader to weigh.
    const judge = shouldJudge(g) ? await judgeGroup(g, opts.repoRoot, opts.selfConsistencyRounds) : null;
    return fuseGroup(g, opts.repoRoot, crossed, judge);
  })));
  const valid = findings.filter((f): f is Finding => f !== null);
  return rank(valid);
}

function severityRank(s: Finding['severity']): number {
  return { critical: 4, high: 3, medium: 2, low: 1 }[s];
}

function rank(findings: Finding[]): Finding[] {
  return [...findings].sort((a, b) => {
    if (severityRank(a.severity) !== severityRank(b.severity)) return severityRank(b.severity) - severityRank(a.severity);
    return b.confidenceScore - a.confidenceScore;
  });
}
