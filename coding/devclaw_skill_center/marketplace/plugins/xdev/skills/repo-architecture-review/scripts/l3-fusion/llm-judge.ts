import { readFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { callLlm, extractJsonFromLlmText } from './llm-cli.js';
import { containsBannedPhrase, hasEvidenceAnchor } from './banlist.js';
import { fetchSymbols, renderSymbolsForPrompt, type SymbolLookupRequest } from './symbol-lookup.js';
import type { Group } from './grouping.js';
import type { Location } from '../types.js';

const __dirname = dirname(fileURLToPath(import.meta.url));

export interface JudgeOutput {
  verdict: 'real' | 'insufficient' | 'noise';
  reasoning: string;
  confidence: 'high' | 'medium' | 'low';
  agreement?: number;
  votes?: JudgeOutput[];
}

export interface SemanticCategoriesConfig {
  categories: Set<string>;                  // categories eligible for LLM judgment
  tier2Categories: Set<string>;             // subset eligible for Tier-2 symbol lookup
}

export const DEFAULT_SEMANTIC_CATEGORIES: SemanticCategoriesConfig = {
  categories: new Set(['layering', 'doc-drift', 'test-org', 'data-layer', 'security-boundary', 'config']),
  tier2Categories: new Set(['layering', 'data-layer', 'security-boundary', 'test-org']),
};

// Max-coverage mode: judge every group regardless of category, and always allow
// Tier-2 (symbol lookup in round 2) so the LLM can drill into code when helpful.
// The category whitelists above are retained so callers that want stricter
// behavior can still pass a custom SemanticCategoriesConfig.
export function shouldJudge(_group: Group, _cfg: SemanticCategoriesConfig = DEFAULT_SEMANTIC_CATEGORIES): boolean {
  return true;
}

export function canUseTier2(_group: Group, _cfg: SemanticCategoriesConfig = DEFAULT_SEMANTIC_CATEGORIES): boolean {
  return true;
}

export function selectConsistent(votes: JudgeOutput[]): JudgeOutput {
  const counts = new Map<string, number>();
  for (const v of votes) counts.set(v.verdict, (counts.get(v.verdict) ?? 0) + 1);
  let bestVerdict: JudgeOutput['verdict'] = 'insufficient';
  let best = 0;
  for (const [verdict, count] of counts) {
    if (count > best) { best = count; bestVerdict = verdict as JudgeOutput['verdict']; }
  }
  const picked = votes.find((v) => v.verdict === bestVerdict)!;
  return { ...picked, agreement: best / votes.length, votes };
}

interface Round1Final { kind: 'final'; verdict: JudgeOutput['verdict']; reasoning: string; confidence: JudgeOutput['confidence']; }
interface Round1Lookup { kind: 'lookup'; symbols: Array<{ file: string; line: number; reason?: string }>; }

function buildPrompt(promptTemplate: string, evidenceJson: string, symbolsBlock: string): string {
  return promptTemplate.replace('{{EVIDENCE_JSON}}', evidenceJson).replace('{{SYMBOLS_BLOCK}}', symbolsBlock);
}

async function judgeOnce(group: Group, repoRoot: string, allowTier2: boolean, promptTemplate: string): Promise<JudgeOutput> {
  const evidenceJson = JSON.stringify({
    category: group.category,
    commonFile: group.commonFile,
    commonDir: group.commonDir,
    violations: group.violations.map((v) => ({
      source: v.source, rule: v.ruleId, locations: v.locations, evidence: v.evidence, message: v.message,
    })),
  }, null, 2);

  const allowedLocations: Location[] = group.violations.flatMap((v) => v.locations);

  // ROUND 1
  let prompt = buildPrompt(promptTemplate, evidenceJson, '(no symbols yet — request with kind=lookup if needed)');
  const r1 = await callLlm({ prompt });
  if (r1.status !== 'ok') return { verdict: 'insufficient', reasoning: `llm error: ${r1.error ?? ''}`, confidence: 'low' };
  let parsed1: Round1Final | Round1Lookup;
  try { parsed1 = extractJsonFromLlmText(r1.rawText) as Round1Final | Round1Lookup; }
  catch { return { verdict: 'insufficient', reasoning: 'unparseable round1', confidence: 'low' }; }

  if (parsed1.kind === 'final') {
    let v: JudgeOutput = { verdict: parsed1.verdict, reasoning: parsed1.reasoning, confidence: parsed1.confidence };
    if (containsBannedPhrase(v.reasoning)) v.verdict = 'insufficient';
    if (v.verdict === 'real' && !hasEvidenceAnchor(v.reasoning)) v.verdict = 'insufficient';
    return v;
  }

  // ROUND 2: only if Tier-2 allowed for this category.
  if (!allowTier2) return { verdict: 'insufficient', reasoning: 'tier-2 lookup not permitted for this category', confidence: 'low' };

  const requests: SymbolLookupRequest[] = (parsed1.symbols ?? []);
  const symbols = await fetchSymbols({
    repoRoot, allowedLocations, requests, maxLinesPerSymbol: 1000, maxSymbols: 50,
  });
  const symbolsBlock = renderSymbolsForPrompt(symbols);
  const prompt2 = buildPrompt(promptTemplate, evidenceJson, symbolsBlock);
  const r2 = await callLlm({ prompt: prompt2 });
  if (r2.status !== 'ok') return { verdict: 'insufficient', reasoning: `llm error round2: ${r2.error ?? ''}`, confidence: 'low' };
  let parsed2: Round1Final;
  try {
    const parsed = extractJsonFromLlmText(r2.rawText) as Round1Final | Round1Lookup;
    if (parsed.kind !== 'final') return { verdict: 'insufficient', reasoning: 'round2 did not produce final', confidence: 'low' };
    parsed2 = parsed;
  } catch { return { verdict: 'insufficient', reasoning: 'unparseable round2', confidence: 'low' }; }
  let v: JudgeOutput = { verdict: parsed2.verdict, reasoning: parsed2.reasoning, confidence: parsed2.confidence };
  if (containsBannedPhrase(v.reasoning)) v.verdict = 'insufficient';
  if (v.verdict === 'real' && !hasEvidenceAnchor(v.reasoning)) v.verdict = 'insufficient';
  return v;
}

export async function judgeGroup(group: Group, repoRoot: string, rounds = 5, cfg: SemanticCategoriesConfig = DEFAULT_SEMANTIC_CATEGORIES): Promise<JudgeOutput> {
  const promptTemplate = await readFile(join(__dirname, 'prompts', 'extractor.txt'), 'utf-8');
  const allowTier2 = canUseTier2(group, cfg);
  const votes: JudgeOutput[] = [];
  for (let i = 0; i < rounds; i++) {
    votes.push(await judgeOnce(group, repoRoot, allowTier2, promptTemplate));
  }
  return selectConsistent(votes);
}
