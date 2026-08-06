import { readFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { callLlm, extractJsonFromLlmText } from './llm-cli.js';
import { containsBannedPhrase, hasEvidenceAnchor } from './banlist.js';
import { fetchSymbols, renderSymbolsForPrompt, type SymbolLookupRequest } from './symbol-lookup.js';
import { canUseTier2, DEFAULT_SEMANTIC_CATEGORIES } from './llm-judge.js';
import { renderMetricHardEvidenceFinding, isMetricHardEvidence } from './metric-narrative.js';
import type { Group } from './grouping.js';
import type { Finding, Severity, Location } from '../types.js';
import type { Crossed } from './signal-crossing.js';
import type { JudgeOutput } from './llm-judge.js';

const __dirname = dirname(fileURLToPath(import.meta.url));

export interface FuserOut {
  title: string;
  rootCause: string;
  impact: string;
  actions: string[];
}

interface FuserRound1Final { kind: 'final'; title: string; rootCause: string; impact: string; actions: string[]; }
interface FuserRound1Lookup { kind: 'lookup'; symbols: Array<{ file: string; line: number; reason?: string }>; }

function pickSeverity(group: Group): Severity {
  const order: Severity[] = ['critical', 'high', 'medium', 'low'];
  const max = group.violations.reduce<Severity>((best, v) => order.indexOf(v.severity) < order.indexOf(best) ? v.severity : best, 'low');
  return max;
}

function finalConfidence(base: 'high' | 'medium' | 'low', ruleWeight: number, agreement: number): number {
  const baseVal = base === 'high' ? 1.0 : base === 'medium' ? 0.7 : 0.4;
  return Number((baseVal * ruleWeight * agreement).toFixed(3));
}

function ruleSourceWeight(group: Group): number {
  const weights = group.violations.map((v) => v.ruleSource === 'repo-config' ? 1.0 : v.ruleSource === 'skill-generated' ? 0.8 : 0.7);
  return weights.reduce((a: number, b: number) => Math.min(a, b), 1.0);      // worst-case
}

function buildFuserPrompt(tpl: string, group: Group, crossed: Pick<Crossed, 'baseConfidence' | 'independentSources'>, judge: JudgeOutput | null, symbolsBlock: string): string {
  const evidenceJson = JSON.stringify({
    category: group.category,
    commonFile: group.commonFile,
    commonDir: group.commonDir,
    violations: group.violations.map((v) => ({ source: v.source, rule: v.ruleId, locations: v.locations, evidence: v.evidence, message: v.message })),
  }, null, 2);
  return tpl
    .replace('{{CATEGORY}}', group.category)
    .replace('{{CONFIDENCE}}', crossed.baseConfidence)
    .replace('{{VERDICT}}', judge?.verdict ?? 'n/a')
    .replace('{{SOURCES}}', crossed.independentSources.join(', '))
    .replace('{{EVIDENCE_JSON}}', evidenceJson)
    .replace('{{SYMBOLS_BLOCK}}', symbolsBlock);
}

export async function fuseGroup(
  group: Group,
  repoRoot: string,
  crossed: Pick<Crossed, 'baseConfidence' | 'independentSources'>,
  judge: JudgeOutput | null,
): Promise<Finding | null> {
  // Deterministic metric-based violations (code clones, cycles, god-packages,
  // layering violations, high fan-in/out) carry irrefutable numeric evidence
  // — the metric IS the finding. Do not route them through the LLM where they
  // routinely get folded into the "low-signal" bucket. Instead render a
  // Chinese narrative directly from the metrics and promote as high-confidence.
  if (isMetricHardEvidence(group)) {
    return renderMetricHardEvidenceFinding(group);
  }

  const promptTemplate = await readFile(join(__dirname, 'prompts', 'fuser.txt'), 'utf-8');
  const allowTier2 = canUseTier2(group, DEFAULT_SEMANTIC_CATEGORIES);
  const allowedLocations: Location[] = group.violations.flatMap((v) => v.locations);

  // ROUND 1
  const prompt1 = buildFuserPrompt(promptTemplate, group, crossed, judge, '(no symbols yet)');
  const r1 = await callLlm({ prompt: prompt1 });
  let final: FuserRound1Final;
  if (r1.status !== 'ok') {
    final = { kind: 'final', title: '<skip>', rootCause: `LLM call failed (${r1.status}): ${r1.error ?? ''}`, impact: '', actions: [] };
  } else {
    let parsed1: FuserRound1Final | FuserRound1Lookup | null;
    try { parsed1 = extractJsonFromLlmText(r1.rawText) as FuserRound1Final | FuserRound1Lookup; }
    catch { parsed1 = null; }
    if (parsed1 && parsed1.kind === 'final') {
      final = parsed1;
    } else if (parsed1 && parsed1.kind === 'lookup' && allowTier2) {
      const requests: SymbolLookupRequest[] = (parsed1.symbols ?? []);
      const symbols = await fetchSymbols({ repoRoot, allowedLocations, requests, maxLinesPerSymbol: 1000, maxSymbols: 50 });
      const prompt2 = buildFuserPrompt(promptTemplate, group, crossed, judge, renderSymbolsForPrompt(symbols));
      const r2 = await callLlm({ prompt: prompt2 });
      if (r2.status !== 'ok') {
        final = { kind: 'final', title: '<skip>', rootCause: `LLM round2 failed (${r2.status})`, impact: '', actions: [] };
      } else {
        try {
          const parsed2 = extractJsonFromLlmText(r2.rawText) as FuserRound1Final | FuserRound1Lookup;
          final = parsed2.kind === 'final' ? parsed2 : { kind: 'final', title: '<skip>', rootCause: 'round2 not final', impact: '', actions: [] };
        } catch {
          final = { kind: 'final', title: '<skip>', rootCause: 'round2 unparseable', impact: '', actions: [] };
        }
      }
    } else {
      final = { kind: 'final', title: '<skip>', rootCause: 'round1 unparseable or lookup-not-allowed', impact: '', actions: [] };
    }
  }

  // Max-coverage mode: never silently drop a group. Even when the LLM emitted
  // "<skip>" (or the judge voted noise), surface a low-confidence pass-through
  // finding built directly from raw violation evidence so the reader sees what
  // was triaged out and can disagree.
  const agreement = judge?.agreement ?? 1.0;
  let confidenceScore = finalConfidence(crossed.baseConfidence, ruleSourceWeight(group), agreement);
  let title = final.title;
  let rootCause = final.rootCause;
  let impact = final.impact;
  let actions = final.actions;
  let isPassthrough = false;
  if (!title || title === '<skip>') {
    isPassthrough = true;
    const repr = group.violations[0];
    title = `[LOW-SIGNAL] ${repr?.message ?? group.category + ' issue at ' + (group.commonDir ?? group.commonFile ?? 'unknown')}`;
    rootCause = `LLM judge classified this as low-signal/noise (verdict=${judge?.verdict ?? 'n/a'}). Raw evidence retained for manual review: ${group.violations.map((v) => `[${v.source}] ${v.message}`).slice(0, 5).join(' | ')}`;
    impact = 'Low signal — likely false positive but reported for transparency under max-coverage mode.';
    actions = ['Manually inspect the listed locations to confirm whether this is a true architecture issue or expected design.'];
    confidenceScore = Math.min(confidenceScore, 0.3);
  }
  const confidence: Finding['confidence'] = confidenceScore >= 0.8 ? 'high' : confidenceScore >= 0.5 ? 'medium' : 'low';

  // No slice cap: emit every distinct location the group has.
  const locations = [...new Map(group.violations.flatMap((v) => v.locations).map((l) => [l.file + (l.line ?? ''), l])).values()];

  return {
    id: group.id,
    category: group.category,
    title,
    rootCause,
    impact,
    actions,
    confidence,
    confidenceScore,
    severity: isPassthrough ? 'low' : pickSeverity(group),
    signals: group.violations.map((v) => ({ source: v.source, description: v.message || v.ruleId })),
    locations,
    violationIds: group.violations.map((v) => v.id),
  };
}
