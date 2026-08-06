import { runCmd } from '../util/exec.js';

export interface LlmCallOpts {
  prompt: string;
  timeoutMs?: number;
  cliBinary?: string;                 // 'claude' by default
  model?: string;                     // e.g. 'claude-opus-4-7-1m'
  input?: string;                     // additional stdin content
}

export interface LlmCallResult {
  status: 'ok' | 'failed' | 'timeout';
  rawText: string;
  durationMs: number;
  error?: string;
}

export async function callLlm(opts: LlmCallOpts): Promise<LlmCallResult> {
  const bin = opts.cliBinary ?? 'claude';
  const args = ['-p', opts.prompt];
  // Max-coverage mode: default to the strongest long-context Claude model so
  // judgments see the full evidence block even when it grows large. Callers can
  // still override via opts.model.
  const model = opts.model ?? process.env.ARCHREVIEW_LLM_MODEL ?? 'claude-opus-4-7[1m]';
  args.push('--model', model);
  const r = await runCmd(bin, args, { timeoutMs: opts.timeoutMs ?? 600_000, input: opts.input });
  if (r.status === 'timeout') return { status: 'timeout', rawText: '', durationMs: r.durationMs, error: 'timeout' };
  if (r.status === 'failed') return { status: 'failed', rawText: r.stdout, durationMs: r.durationMs, error: r.stderr };
  return { status: 'ok', rawText: r.stdout, durationMs: r.durationMs };
}

export function extractJsonFromLlmText(text: string): unknown {
  const fence = text.match(/```(?:json)?\s*([\s\S]*?)(?:```|$)/);
  let payload = (fence ? fence[1] : text).trim();
  if (payload.startsWith('`') && payload.endsWith('`')) payload = payload.slice(1, -1).trim();
  try { return JSON.parse(payload); } catch { /* fall through */ }
  const repaired = repairTruncatedJson(payload);
  try { return JSON.parse(repaired); } catch { throw new Error(`LLM output not JSON: ${text.slice(0, 200)}`); }
}

export function repairTruncatedJson(s: string): string {
  let depth = 0, bracketDepth = 0, inStr = false, esc = false;
  for (const ch of s) {
    if (esc) { esc = false; continue; }
    if (ch === '\\') { esc = true; continue; }
    if (ch === '"') { inStr = !inStr; continue; }
    if (inStr) continue;
    if (ch === '{') depth++;
    if (ch === '}') depth--;
    if (ch === '[') bracketDepth++;
    if (ch === ']') bracketDepth--;
  }
  let tail = s;
  if (inStr) tail += '"';
  while (bracketDepth-- > 0) tail += ']';
  while (depth-- > 0) tail += '}';
  return tail.replace(/,\s*([}\]])/g, '$1');
}
