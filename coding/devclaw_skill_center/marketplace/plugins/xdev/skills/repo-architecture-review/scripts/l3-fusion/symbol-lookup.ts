import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import type { Location } from '../types.js';

export interface SymbolLookupRequest {
  file: string;
  line?: number;       // optional: if provided, we locate the enclosing symbol; else return top of file
  symbol?: string;     // optional: if provided, we prefer matching a symbol definition by name
}

export interface FetchSymbolsOpts {
  repoRoot: string;
  allowedLocations: Location[];     // from the group's evidence — requests MUST match one of these (file-level)
  requests: SymbolLookupRequest[];
  maxLinesPerSymbol: number;         // default 150
  maxSymbols: number;                // default 3
}

export interface SymbolSource {
  file: string;
  startLine: number;
  endLine: number;
  source: string;
  truncated: boolean;
  language: 'python' | 'go' | 'other';
}

function detectLanguage(file: string): SymbolSource['language'] {
  if (file.endsWith('.py')) return 'python';
  if (file.endsWith('.go')) return 'go';
  return 'other';
}

function extractPythonEnclosing(lines: string[], anchorLine: number): { start: number; end: number } {
  // Walk backwards to find the nearest `def` or `class` at column 0 (top-level) or at lower indent than anchor.
  let start = anchorLine;
  let baseIndent = Infinity;
  for (let i = anchorLine - 1; i >= 0; i--) {
    const line = lines[i];
    if (!line.trim()) continue;
    const indent = line.match(/^\s*/)![0].length;
    const isDef = /^\s*(def|class)\s/.test(line);
    if (isDef && indent < baseIndent) {
      start = i;
      baseIndent = indent;
      if (indent === 0) break;       // top-level def
    }
  }
  const headerIndent = (lines[start].match(/^\s*/) ?? [''])[0].length;
  let end = lines.length - 1;
  for (let i = start + 1; i < lines.length; i++) {
    const line = lines[i];
    if (!line.trim()) continue;
    const indent = line.match(/^\s*/)![0].length;
    if (indent <= headerIndent && /^\s*(def|class|\S)/.test(line) && i !== start) {
      end = i - 1;
      break;
    }
  }
  return { start, end };
}

function extractGoEnclosing(lines: string[], anchorLine: number): { start: number; end: number } {
  // Walk backwards to find `func` / `type` declaration.
  let start = anchorLine;
  for (let i = anchorLine; i >= 0; i--) {
    const line = lines[i];
    if (/^(func|type)\s/.test(line)) { start = i; break; }
  }
  // Walk forward counting braces from the opening brace of the declaration.
  let depth = 0;
  let end = start;
  let sawOpen = false;
  for (let i = start; i < lines.length; i++) {
    for (const ch of lines[i]) {
      if (ch === '{') { depth++; sawOpen = true; }
      else if (ch === '}') depth--;
    }
    if (sawOpen && depth === 0) { end = i; break; }
  }
  return { start, end };
}

export async function fetchSymbols(opts: FetchSymbolsOpts): Promise<SymbolSource[]> {
  const allowedFiles = new Set(opts.allowedLocations.map((l) => l.file));
  const accepted = opts.requests.filter((r) => allowedFiles.has(r.file)).slice(0, opts.maxSymbols);
  const out: SymbolSource[] = [];
  for (const req of accepted) {
    let src: string;
    try { src = await readFile(join(opts.repoRoot, req.file), 'utf-8'); } catch { continue; }
    const lines = src.split('\n');
    const anchor = Math.max(0, Math.min((req.line ?? 1) - 1, lines.length - 1));
    const lang = detectLanguage(req.file);
    let range: { start: number; end: number };
    if (lang === 'python') range = extractPythonEnclosing(lines, anchor);
    else if (lang === 'go') range = extractGoEnclosing(lines, anchor);
    else range = { start: anchor, end: Math.min(anchor + 20, lines.length - 1) };

    let truncated = false;
    if (range.end - range.start + 1 > opts.maxLinesPerSymbol) {
      range.end = range.start + opts.maxLinesPerSymbol - 1;
      truncated = true;
    }
    const body = lines.slice(range.start, range.end + 1).join('\n');
    const sourceText = truncated ? body + '\n...（truncated）' : body;
    out.push({
      file: req.file,
      startLine: range.start + 1,
      endLine: range.end + 1,
      source: sourceText,
      truncated,
      language: lang,
    });
  }
  return out;
}

/** Convert fetched symbols into a prompt-ready block. */
export function renderSymbolsForPrompt(syms: SymbolSource[]): string {
  if (syms.length === 0) return '(no symbols available)';
  return syms.map((s) => `--- ${s.file}:${s.startLine}-${s.endLine} (${s.language}${s.truncated ? ', truncated' : ''}) ---\n${s.source}`).join('\n\n');
}
