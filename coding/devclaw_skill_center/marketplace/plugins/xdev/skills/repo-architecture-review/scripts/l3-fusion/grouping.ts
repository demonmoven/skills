import { dirname } from 'node:path';
import type { Violation, Category } from '../types.js';

export interface Group {
  id: string;
  category: Category;
  commonFile?: string;
  commonDir?: string;
  violations: Violation[];
}

// Rule IDs that carry irrefutable per-occurrence metric evidence. Each
// occurrence must stay as its own group so the report shows every clone,
// every cycle, every god-package individually.
const KEEP_PER_VIOLATION_RULE_IDS = new Set([
  'go-arch:code-clone',
  'go-arch:cycle',
  'go-arch:god-package',
  'go-arch:horizontal-coupling',
  'goda:high-fan-in',
  'high-fan-in',
]);

function isPerViolationRule(v: Violation): boolean {
  return KEEP_PER_VIOLATION_RULE_IDS.has(v.ruleId);
}

// Layering violations share a single narrative per (srcLayer -> depLayer)
// pair — emit one aggregated group so a repo with 30 similar edges shows up
// as one finding with all edges as evidence, not as 30 near-identical rows.
function isLayeringRule(v: Violation): boolean {
  return v.ruleId.startsWith('go-arch:layering:');
}

export function groupViolations(vs: Violation[]): Group[] {
  const out: Group[] = [];
  const toMerge: Violation[] = [];
  const layeringBuckets = new Map<string, Violation[]>();

  for (const v of vs) {
    if (isLayeringRule(v)) {
      const key = v.ruleId; // e.g. "go-arch:layering:dal->repository"
      if (!layeringBuckets.has(key)) layeringBuckets.set(key, []);
      layeringBuckets.get(key)!.push(v);
    } else if (isPerViolationRule(v)) {
      const file = v.locations[0]?.file ?? '<unknown>';
      out.push({
        id: `grp:${v.ruleId}|${v.id}`,
        category: v.category,
        commonFile: file,
        commonDir: dirname(file),
        violations: [v],
      });
    } else {
      toMerge.push(v);
    }
  }

  for (const [key, group] of layeringBuckets) {
    out.push({
      id: `grp:${key}`,
      category: 'layering',
      commonDir: undefined,
      commonFile: undefined,
      violations: group,
    });
  }

  // Existing directory-merge behavior for non-metric (noisy) rules.
  const byKey = new Map<string, Violation[]>();
  for (const v of toMerge) {
    const file = v.locations[0]?.file ?? '<unknown>';
    const keyFile = `${v.category}|${file}`;
    if (!byKey.has(keyFile)) byKey.set(keyFile, []);
    byKey.get(keyFile)!.push(v);
  }
  const merged = new Map<string, Violation[]>();
  for (const [key, group] of byKey) {
    const [category, file] = key.split('|');
    const dir = dirname(file);
    const mergeKey = `${category}|${dir}`;
    if (!merged.has(mergeKey)) merged.set(mergeKey, []);
    merged.get(mergeKey)!.push(...group);
  }
  for (const [mergeKey, violations] of merged) {
    const [category, dir] = mergeKey.split('|');
    const files = [...new Set(violations.map((v) => v.locations[0]?.file).filter(Boolean))];
    out.push({
      id: `grp:${mergeKey}`,
      category: category as Category,
      commonDir: dir,
      commonFile: files.length === 1 ? files[0] : undefined,
      violations,
    });
  }
  return out;
}
