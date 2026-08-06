// scripts/l0-prescan/layer-discovery.ts
//
// Uses LLM to infer the layering convention of a Go/Python repository by
// analysing its directory structure and package names. This replaces the
// hardcoded `LAYER_ORDER` that previously assumed every Go repo follows the
// ByteDance handler → consumer → facade → service → repository → dao → dal
// convention.
//
// Why LLM instead of heuristics?
//   - Layering conventions are deeply domain-specific. A Hertz service uses
//     handler/service/dao; a DDD project uses cmd/app/domain/infra; a CLI tool
//     uses cmd/internal/pkg. Pattern-matching would need a growing list of
//     special cases. An LLM can read the actual directory names and infer the
//     intent — much like a new team member looking at the repo for the first time.
//
// The LLM call adds ~5–15 seconds to the pre-scan. Since this runs once per
// analysis (not per-tool), the cost is acceptable for a 2–6 minute pipeline.

import type { LayeringConvention } from '../types.js';
import { callLlm, extractJsonFromLlmText } from '../l3-fusion/llm-cli.js';

/** Well-known convention used as a fallback when LLM is unavailable. */
export const FALLBACK_CONVENTION: LayeringConvention = {
  hasLayering: true,
  layers: ['handler', 'consumer', 'facade', 'service', 'repository', 'dao', 'dal'],
  businessLayers: ['handler', 'consumer', 'facade', 'service'],
  infraLayers: ['repository', 'dao', 'dal'],
  conventionSummary: 'handler → consumer → facade → service → repository → dao → dal (ByteDance web-service default)',
  reasoning: 'Fallback: LLM layer discovery was unavailable; using the default ByteDance web-service convention.',
};

const DISCOVER_PROMPT = `You are a senior software architect. Given the following directory/package structure of a code repository, identify the **layering convention** (if one exists).

## Input
- Language: {lang}
- Module path: {modulePath}
- Top-level directories (first 2 levels):
{dirTree}

- Go package paths (internal only):
{packageList}

## Task
Determine whether this repository follows a layered architecture pattern. Common patterns include:
- Web-service style: handler → service → repository → dao → dal
- DDD style: cmd → application → domain → infrastructure
- Clean Architecture: delivery → usecase → repository → entity
- Flat / no layering

## Output
Return a single JSON object (no markdown fences):
{
  "hasLayering": true/false,
  "layers": ["outermost", "...", "innermost"],
  "businessLayers": ["layers whose siblings should be isolated"],
  "infraLayers": ["layers where sibling sharing is OK"],
  "conventionSummary": "human-readable summary, e.g. handler → service → dao",
  "reasoning": "brief explanation of how you inferred this"
}

Rules:
- "layers" must be ordered from outermost (entry point, e.g. handler/cmd) to innermost (infra, e.g. dal/db).
- Each layer name should be the actual directory segment used in this repo (e.g. "biz/handler" if the repo nests under biz/).
- If no clear layering exists, return hasLayering=false with empty arrays.
- Be conservative: only declare layers that are clearly part of a vertical dependency chain.
- The layer names should be bare segment names (e.g. "handler" not "biz/handler") — we match them against path segments.
`;

export async function discoverLayering(opts: {
  lang: string;
  modulePath: string;
  topDirs: string[];
  packagePaths: string[];
  skipLlm?: boolean;
}): Promise<LayeringConvention> {
  if (opts.skipLlm) {
    return FALLBACK_CONVENTION;
  }
  const dirTree = opts.topDirs.slice(0, 60).map((d) => `  ${d}`).join('\n') || '  (none)';
  const packageList = opts.packagePaths.slice(0, 80).map((p) => `  ${p}`).join('\n') || '  (none)';

  const prompt = DISCOVER_PROMPT
    .replace('{lang}', opts.lang)
    .replace('{modulePath}', opts.modulePath || '(unknown)')
    .replace('{dirTree}', dirTree)
    .replace('{packageList}', packageList);

  try {
    const result = await callLlm({ prompt, timeoutMs: 30_000 });
    if (result.status !== 'ok' || !result.rawText.trim()) {
      return FALLBACK_CONVENTION;
    }
    const parsed = extractJsonFromLlmText(result.rawText) as Record<string, unknown>;
    if (!parsed || typeof parsed !== 'object') return FALLBACK_CONVENTION;

    const hasLayering = !!parsed.hasLayering;
    if (!hasLayering) {
      return {
        hasLayering: false,
        layers: [],
        businessLayers: [],
        infraLayers: [],
        conventionSummary: 'No layering convention detected',
        reasoning: String(parsed.reasoning ?? 'LLM determined no clear layering exists.'),
      };
    }

    const layers = Array.isArray(parsed.layers) ? parsed.layers.map(String) : [];
    const businessLayers = Array.isArray(parsed.businessLayers) ? parsed.businessLayers.map(String) : [];
    const infraLayers = Array.isArray(parsed.infraLayers) ? parsed.infraLayers.map(String) : [];

    if (layers.length < 2) return FALLBACK_CONVENTION;

    return {
      hasLayering: true,
      layers,
      businessLayers: businessLayers.length > 0 ? businessLayers : layers.slice(0, Math.ceil(layers.length / 2)),
      infraLayers: infraLayers.length > 0 ? infraLayers : layers.slice(Math.ceil(layers.length / 2)),
      conventionSummary: String(parsed.conventionSummary ?? layers.join(' → ')),
      reasoning: String(parsed.reasoning ?? ''),
    };
  } catch {
    // LLM unavailable or returned garbage — fall back gracefully.
    return FALLBACK_CONVENTION;
  }
}
