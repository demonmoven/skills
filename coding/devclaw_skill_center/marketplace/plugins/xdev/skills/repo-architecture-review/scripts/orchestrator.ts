import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import pLimit from 'p-limit';
import type { RepoProfile, RunMeta, ToolRunResult, Violation, Finding } from './types.js';
import { buildRepoProfile } from './l0-prescan/index.js';
import { pickAdaptersFor } from './l1-tools/registry.js';
import { runAdapterSafely, type AdapterContext } from './l1-tools/adapter.js';
import { computeL2Violations } from './l2-graph/violations.js';
import { fuseFindings, prepareGroupsForExternalLlm } from './l3-fusion/index.js';
import { loadBaseline, saveBaseline, diffBaseline } from './baseline/index.js';
import { shouldChunk } from './mapreduce/partition.js';
import { renderReport } from './l4-report/index.js';

export interface AnalyzeOpts {
  rootPath: string;
  output: string;
  noBaseline: boolean;
  mode: 'quick' | 'deep';
  resume: boolean;
  disabledToolIds: Set<string>;
  parallelism: number;
}

export interface AnalyzeResult {
  profile: RepoProfile;
  toolRuns: ToolRunResult[];
  graphViolations: Violation[];
  findings: Finding[];
  reportedFindings: Finding[];
  knownUnchanged: Finding[];
  runMeta: RunMeta;
  reportPath: string;
}

// ---------------------------------------------------------------------------
// Phase: tools — runs L0 (no LLM), L1, L2, grouping, signal-crossing.
// Produces intermediate artifacts under .architecture-review/tmp/ for the
// agent to read.  Metric-hard-evidence groups are rendered as deterministic
// findings; all other groups are written to pending-llm-groups.json for the
// agent to process via its own subagent capability.
// ---------------------------------------------------------------------------

export interface ToolsPhaseOpts {
  rootPath: string;
  disabledToolIds: Set<string>;
  parallelism: number;
}

export interface ToolsPhaseResult {
  tmpDir: string;
  profile: RepoProfile;
  toolRuns: ToolRunResult[];
  deterministicFindingsCount: number;
  pendingLlmGroupsCount: number;
}

export async function runToolsPhase(opts: ToolsPhaseOpts): Promise<ToolsPhaseResult> {
  const started = new Date();
  const tmpDir = join(opts.rootPath, '.architecture-review', 'tmp');
  await mkdir(tmpDir, { recursive: true });

  // L0: prescan without LLM (uses fallback layering convention)
  const profile = await buildRepoProfile(opts.rootPath, { skipLlm: true });
  await writeFile(join(tmpDir, 'repo-profile.json'), JSON.stringify(profile, null, 2));

  // Save layer-discovery inputs so the agent can do LLM layer discovery itself
  const goPackages = profile.topLevelPackages.go ?? [];
  const pyPackages = profile.topLevelPackages.python ?? [];
  const topDirs = goPackages.length > 0 ? goPackages : pyPackages;
  const primaryLang = profile.languages.find((l) => l.lang === 'go') ? 'go' : profile.languages.find((l) => l.lang === 'python') ? 'python' : 'unknown';
  await writeFile(join(tmpDir, 'layer-discovery-inputs.json'), JSON.stringify({
    lang: primaryLang,
    modulePath: profile.goModulePath ?? '',
    topDirs: topDirs.slice(0, 60),
    packagePaths: goPackages.slice(0, 80),
  }, null, 2));

  // L1: tool adapters
  const adapters = pickAdaptersFor(profile);
  const limit = pLimit(opts.parallelism);
  const ctx: AdapterContext = {
    profile,
    tmpDir,
    rulesFromBuiltin: true,
    timeoutMs: 300_000,
    disabledToolIds: opts.disabledToolIds,
  };

  // L1 + L2 in parallel
  const [toolRuns, graphViolations] = await Promise.all([
    Promise.all(adapters.map((a) => limit(() => runAdapterSafely(a, ctx)))),
    computeL2Violations(profile),
  ]);

  await writeFile(join(tmpDir, 'tool-runs.json'), JSON.stringify(toolRuns, null, 2));
  await writeFile(join(tmpDir, 'l2-violations.json'), JSON.stringify(graphViolations, null, 2));

  // Grouping + signal crossing + metric-hard-evidence separation
  const allViolations = [...toolRuns.flatMap((r) => r.violations), ...graphViolations];
  const { deterministicFindings, pendingLlmGroups } = prepareGroupsForExternalLlm(allViolations);

  await writeFile(join(tmpDir, 'deterministic-findings.json'), JSON.stringify(deterministicFindings, null, 2));
  await writeFile(join(tmpDir, 'pending-llm-groups.json'), JSON.stringify(pendingLlmGroups, null, 2));

  // Save run metadata for the report phase
  const finished = new Date();
  const runMeta: RunMeta = {
    startedAt: started.toISOString(),
    finishedAt: finished.toISOString(),
    wallClockMs: finished.getTime() - started.getTime(),
    toolRuns,
    generatedFiles: toolRuns.map((r) => r.generatedConfigPath).filter((x): x is string => !!x),
    coverageDowngrades: toolRuns.filter((r) => r.status === 'skipped').map((r) => `${r.toolId}: ${r.reason}`),
    stackDetection: profile.languages.map((l) => l.lang),
    mapReduceUsed: shouldChunk(profile),
  };
  await writeFile(join(tmpDir, 'run-meta.json'), JSON.stringify(runMeta, null, 2));

  return {
    tmpDir,
    profile,
    toolRuns,
    deterministicFindingsCount: deterministicFindings.length,
    pendingLlmGroupsCount: pendingLlmGroups.length,
  };
}

// ---------------------------------------------------------------------------
// Phase: report — reads findings.json (agent-provided or merged) and renders
// the final Markdown report.  The agent is expected to have already processed
// pending-llm-groups.json and written llm-findings.json.
// ---------------------------------------------------------------------------

export interface ReportPhaseOpts {
  rootPath: string;
  output: string;
}

export async function runReportPhase(opts: ReportPhaseOpts): Promise<string> {
  const tmpDir = join(opts.rootPath, '.architecture-review', 'tmp');

  const profile: RepoProfile = JSON.parse(await readFile(join(tmpDir, 'repo-profile.json'), 'utf-8'));
  const toolRuns: ToolRunResult[] = JSON.parse(await readFile(join(tmpDir, 'tool-runs.json'), 'utf-8'));
  const runMeta: RunMeta = JSON.parse(await readFile(join(tmpDir, 'run-meta.json'), 'utf-8'));
  const deterministicFindings: Finding[] = JSON.parse(await readFile(join(tmpDir, 'deterministic-findings.json'), 'utf-8'));

  // Read agent-produced LLM findings (may not exist if there were no pending groups)
  let llmFindings: Finding[] = [];
  try {
    llmFindings = JSON.parse(await readFile(join(tmpDir, 'llm-findings.json'), 'utf-8'));
  } catch {
    // No LLM findings file — that's fine if there were no pending groups
  }

  const findings = rankFindings([...deterministicFindings, ...llmFindings]);
  await writeFile(join(tmpDir, 'findings.json'), JSON.stringify(findings, null, 2));

  // Save baseline
  await saveBaseline(opts.rootPath, { createdAt: runMeta.startedAt, findings });

  const graphViolations: Violation[] = JSON.parse(await readFile(join(tmpDir, 'l2-violations.json'), 'utf-8'));

  const result: AnalyzeResult = {
    profile,
    toolRuns,
    graphViolations,
    findings,
    reportedFindings: findings,
    knownUnchanged: [],
    runMeta,
    reportPath: opts.output,
  };

  const reportContent = renderReport(result);
  await writeFile(opts.output, reportContent);
  return opts.output;
}

function rankFindings(findings: Finding[]): Finding[] {
  const severityRank = (s: Finding['severity']): number => ({ critical: 4, high: 3, medium: 2, low: 1 }[s]);
  return [...findings].sort((a, b) => {
    if (severityRank(a.severity) !== severityRank(b.severity)) return severityRank(b.severity) - severityRank(a.severity);
    return b.confidenceScore - a.confidenceScore;
  });
}

// ---------------------------------------------------------------------------
// Full pipeline (original behavior, backward-compatible)
// ---------------------------------------------------------------------------

export async function runAnalyze(opts: AnalyzeOpts): Promise<AnalyzeResult> {
  const started = new Date();
  const tmpDir = join(opts.rootPath, '.architecture-review', 'tmp');
  await mkdir(tmpDir, { recursive: true });

  const profile = await buildRepoProfile(opts.rootPath);
  await writeFile(join(tmpDir, 'repo-profile.json'), JSON.stringify(profile, null, 2));

  const adapters = pickAdaptersFor(profile);
  const limit = pLimit(opts.parallelism);

  const ctx: AdapterContext = {
    profile,
    tmpDir,
    rulesFromBuiltin: true,
    timeoutMs: 300_000,
    disabledToolIds: opts.disabledToolIds,
  };

  const [toolRuns, graphViolations] = await Promise.all([
    Promise.all(adapters.map((a) => limit(() => runAdapterSafely(a, ctx)))),
    computeL2Violations(profile),
  ]);

  await writeFile(join(tmpDir, 'tool-runs.json'), JSON.stringify(toolRuns, null, 2));
  await writeFile(join(tmpDir, 'l2-violations.json'), JSON.stringify(graphViolations, null, 2));

  const allViolations = [...toolRuns.flatMap((r) => r.violations), ...graphViolations];
  // Always run full LLM fusion — no env-flag short-circuit. Max coverage mode.
  const findings = await fuseFindings(allViolations, {
    repoRoot: opts.rootPath,
    concurrency: opts.parallelism,
    selfConsistencyRounds: opts.mode === 'quick' ? 1 : 5,
    llmOptional: false,
  });
  await writeFile(join(tmpDir, 'findings.json'), JSON.stringify(findings, null, 2));

  // Baseline diff removed — max-coverage mode reports every finding every run.
  // Still write the baseline artefact so it's available for downstream tooling.
  const reportedFindings = findings;
  const knownUnchanged: Finding[] = [];
  await saveBaseline(opts.rootPath, { createdAt: started.toISOString(), findings });
  await writeFile(join(tmpDir, 'findings-reported.json'), JSON.stringify(reportedFindings, null, 2));

  const finished = new Date();
  const runMeta: RunMeta = {
    startedAt: started.toISOString(),
    finishedAt: finished.toISOString(),
    wallClockMs: finished.getTime() - started.getTime(),
    toolRuns,
    generatedFiles: toolRuns.map((r) => r.generatedConfigPath).filter((x): x is string => !!x),
    coverageDowngrades: toolRuns.filter((r) => r.status === 'skipped').map((r) => `${r.toolId}: ${r.reason}`),
    stackDetection: profile.languages.map((l) => l.lang),
    mapReduceUsed: shouldChunk(profile),
  };

  const result: AnalyzeResult = { profile, toolRuns, graphViolations, findings, reportedFindings, knownUnchanged, runMeta, reportPath: opts.output };
  const reportContent = renderReport(result);
  await writeFile(opts.output, reportContent);

  return result;
}
