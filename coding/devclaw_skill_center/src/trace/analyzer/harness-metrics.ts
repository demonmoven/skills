import type { NormalizedMessage, HarnessArtifact, HarnessMetrics } from './types.js';

// ─── Harness artifact category detection ───────────────────────────

type ArtifactCategory = HarnessArtifact['category'];

/**
 * Classify a file path as a harness artifact category, or return null
 * if the path is not a harness artifact.
 *
 * Rules:
 *   - AGENTS.md / CLAUDE.md / ARCHITECTURE.md / README.md at any depth → doc-root
 *   - docs/** EXCEPT docs/plans/** → subcategory by directory
 *   - .skills/ / .claude/skills/ / .opencode/skills/ → skill
 *   - docs/plans/** → NOT a harness artifact (exec plan working dir)
 */
export function classifyPath(filePath: string): ArtifactCategory | null {
  const p = normalizePath(filePath);

  // Root-level doc files (at any directory depth)
  const basename = p.split('/').pop() ?? '';
  if (/^(AGENTS|CLAUDE|ARCHITECTURE)\.md$/i.test(basename)) return 'doc-root';
  if (/^README\.md$/i.test(basename)) return 'doc-root';

  // docs/** but NOT docs/plans/**
  if (p.startsWith('docs/')) {
    if (p.startsWith('docs/plans/')) return null; // ExecPlan working dir, not harness
    if (p.startsWith('docs/rules/')) return 'doc-rules';
    if (p.startsWith('docs/guidance/')) return 'doc-guidance';
    if (p.startsWith('docs/reference/')) return 'doc-reference';
    if (p.startsWith('docs/quality/')) return 'doc-quality';
    // docs/AGENTS.md or other docs/ root files
    if (basename === 'AGENTS.md') return 'doc-root';
    return 'doc-other';
  }

  // Skill directories — only SKILL.md files count as a skill artifact.
  // Other files under a skill dir (references/, scripts/, etc.) are part of
  // the skill but not independent artifacts.
  if (p.startsWith('.skills/') || p.startsWith('.claude/skills/') || p.startsWith('.opencode/skills/')) {
    if (basename === 'SKILL.md') return 'skill';
    return null;
  }

  return null;
}

/**
 * For a file under a skill directory (e.g. `.skills/foo/references/bar.md`),
 * find the parent SKILL.md path that represents the skill artifact.
 * Returns null if the path is not under a skill directory.
 */
export function findParentSkillPath(filePath: string): string | null {
  const p = normalizePath(filePath);
  for (const prefix of ['.skills/', '.claude/skills/', '.opencode/skills/']) {
    if (p.startsWith(prefix)) {
      // Extract skill name: e.g. `.skills/bytedcli/references/foo.md` → `.skills/bytedcli/SKILL.md`
      const rest = p.slice(prefix.length);
      const skillName = rest.split('/')[0];
      if (skillName) return `${prefix}${skillName}/SKILL.md`;
    }
  }
  return null;
}

/**
 * Strip leading slashes, ./ prefixes, and common absolute path prefixes
 * to get a repo-relative path suitable for pattern matching.
 */
export function normalizePath(filePath: string): string {
  let p = filePath;
  // Strip absolute paths — keep only the part after common root markers
  // e.g. /Users/foo/project/docs/rules/x.md → docs/rules/x.md
  // Priority: try directory prefixes first (match at any position including 0),
  // then basename files (only when idx > 0 to avoid stripping the filename itself)
  const dirMarkers = ['docs/', '.skills/', '.claude/', '.opencode/'];
  for (const marker of dirMarkers) {
    const idx = p.indexOf(marker);
    if (idx >= 0) {
      p = p.slice(idx);
      // Strip leading ./ or /
      return p.replace(/^\.\//, '').replace(/^\//, '');
    }
  }
  const fileMarkers = ['AGENTS.md', 'CLAUDE.md', 'ARCHITECTURE.md', 'README.md'];
  for (const marker of fileMarkers) {
    const idx = p.indexOf(marker);
    if (idx >= 0) {
      p = p.slice(idx);
      return p.replace(/^\.\//, '').replace(/^\//, '');
    }
  }
  // Strip leading ./ or /
  return p.replace(/^\.\//, '').replace(/^\//, '');
}

// ─── Read tool names ───────────────────────────────────────────────

const READ_TOOLS = new Set(['Read', 'read']);
const EDIT_TOOLS = new Set(['Edit', 'Write', 'edit', 'write']);
const SKILL_TOOLS = new Set(['Skill', 'skill']);

// ─── Main extraction ──────────────────────────────────────────────

/**
 * Extract harness-related metrics from normalized messages.
 *
 * Scans all tool calls and assistant content to build:
 *   - A list of harness artifacts with read/mention counts
 *   - Skill invocation statistics
 *   - ExecPlan usage flag
 *   - Whether docs were read before code edits
 */
export function extractHarnessMetrics(
  messages: NormalizedMessage[],
  repoHarnessFiles?: string[],
): HarnessMetrics {
  const artifactMap = new Map<string, HarnessArtifact>();
  const skillCounter = new Map<string, number>();
  let execPlanUsed = false;
  let firstDocReadTurn: number | null = null;
  let firstCodeEditTurn: number | null = null;

  let turnIndex = 0;

  for (const msg of messages) {
    // Scan system and user messages for injected harness references
    // (e.g. "Instructions from: docs/reference/xxx.md" in Claude Code system prompts,
    //  or AGENTS.md content auto-loaded as system context)
    if (msg.role === 'system' || msg.role === 'user') {
      if (msg.content) {
        scanForInjectedReferences(msg.content, artifactMap);
      }
      continue;
    }

    // From here: msg.role === 'assistant'
    turnIndex++;

    // Scan tool calls
    if (msg.toolCalls) {
      for (const tc of msg.toolCalls) {
        // Skill invocations
        if (SKILL_TOOLS.has(tc.name)) {
          const skillName = extractSkillName(tc.input);
          if (skillName) {
            skillCounter.set(skillName, (skillCounter.get(skillName) ?? 0) + 1);
          }
          continue;
        }

        const filePath = tc.input;
        if (!filePath) continue;

        // Check for ExecPlan usage (docs/plans/ operations)
        const np = normalizePath(filePath);
        if (np.startsWith('docs/plans/')) {
          execPlanUsed = true;
        }

        // Classify as harness artifact
        const category = classifyPath(filePath);
        if (category) {
          const key = np;
          let artifact = artifactMap.get(key);
          if (!artifact) {
            artifact = { path: key, category, readCount: 0, mentionCount: 0, injectedCount: 0, firstReadTurn: null };
            artifactMap.set(key, artifact);
          }

          if (READ_TOOLS.has(tc.name)) {
            artifact.readCount++;
            if (artifact.firstReadTurn === null) artifact.firstReadTurn = turnIndex;
            if (firstDocReadTurn === null) firstDocReadTurn = turnIndex;
          }
        } else {
          // Check if this is a skill sub-file (e.g. .skills/foo/references/bar.md)
          // and attribute the Read to the parent SKILL.md artifact
          const parentSkill = findParentSkillPath(filePath);
          if (parentSkill && READ_TOOLS.has(tc.name)) {
            let artifact = artifactMap.get(parentSkill);
            if (!artifact) {
              artifact = { path: parentSkill, category: 'skill', readCount: 0, mentionCount: 0, injectedCount: 0, firstReadTurn: null };
              artifactMap.set(parentSkill, artifact);
            }
            artifact.readCount++;
            if (artifact.firstReadTurn === null) artifact.firstReadTurn = turnIndex;
            if (firstDocReadTurn === null) firstDocReadTurn = turnIndex;
          } else if (EDIT_TOOLS.has(tc.name)) {
            // Track first code edit (non-harness file edit)
            if (firstCodeEditTurn === null) firstCodeEditTurn = turnIndex;
          }
        }
      }
    }

    // Scan assistant content for mentions of harness file basenames
    if (msg.content) {
      for (const [key, artifact] of artifactMap) {
        const basename = key.split('/').pop() ?? '';
        if (basename && msg.content.includes(basename)) {
          artifact.mentionCount++;
        }
      }
      // Also detect mentions of common harness files not yet in the map
      for (const name of ['AGENTS.md', 'CLAUDE.md', 'ARCHITECTURE.md', 'invariants.md', 'golden-principles.md']) {
        if (msg.content.includes(name) && !Array.from(artifactMap.values()).some(a => a.path.endsWith(name))) {
          // We know it was mentioned but never accessed via tool call
          // Create a placeholder entry
          const category = classifyPath(name);
          if (category) {
            const existing = artifactMap.get(name);
            if (existing) {
              existing.mentionCount++;
            } else {
              artifactMap.set(name, { path: name, category, readCount: 0, mentionCount: 1, injectedCount: 0, firstReadTurn: null });
            }
          }
        }
      }
    }
  }

  // Merge in repo-scanned harness files that were never seen in the trace
  const repoScanned = Array.isArray(repoHarnessFiles) && repoHarnessFiles.length > 0;
  if (repoScanned) {
    for (const filePath of repoHarnessFiles!) {
      const np = normalizePath(filePath);
      if (!artifactMap.has(np)) {
        const category = classifyPath(filePath);
        if (category) {
          artifactMap.set(np, { path: np, category, readCount: 0, mentionCount: 0, injectedCount: 0, firstReadTurn: null });
        }
      }
    }
  }

  const isUsed = (a: HarnessArtifact) => a.readCount > 0 || a.mentionCount > 0 || a.injectedCount > 0;

  const artifacts = Array.from(artifactMap.values()).sort((a, b) => {
    // Sort: used first (by readCount desc), then unused (by category)
    const aUsed = isUsed(a) ? 1 : 0;
    const bUsed = isUsed(b) ? 1 : 0;
    if (aUsed !== bUsed) return bUsed - aUsed;
    return b.readCount - a.readCount;
  });

  const usedCount = artifacts.filter(isUsed).length;

  const skillInvocations = Array.from(skillCounter.entries())
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count);

  const docReadBeforeEdit = firstDocReadTurn !== null &&
    firstCodeEditTurn !== null &&
    firstDocReadTurn < firstCodeEditTurn;

  return {
    artifacts,
    totalArtifacts: artifacts.length,
    usedArtifacts: usedCount,
    unusedArtifacts: artifacts.length - usedCount,
    skillInvocations,
    execPlanUsed,
    docReadBeforeEdit,
    repoScanned,
  };
}

/**
 * Scan system/user message content for harness file references injected into the prompt.
 *
 * Detected patterns:
 *   - "Instructions from: <path>" (Claude Code / OpenCode system prompt injection)
 *   - Paths matching harness artifact patterns embedded in message text
 */
function scanForInjectedReferences(
  content: string,
  artifactMap: Map<string, HarnessArtifact>,
): void {
  // Pattern 1: "Instructions from: <path>" lines
  const instructionsPattern = /Instructions from:\s*(.+)/g;
  let match;
  while ((match = instructionsPattern.exec(content)) !== null) {
    const rawPath = match[1]!.trim();
    const category = classifyPath(rawPath);
    if (category) {
      const key = normalizePath(rawPath);
      let artifact = artifactMap.get(key);
      if (!artifact) {
        artifact = { path: key, category, readCount: 0, mentionCount: 0, injectedCount: 0, firstReadTurn: null };
        artifactMap.set(key, artifact);
      }
      artifact.injectedCount++;
    }
  }

  // Pattern 2: Detect harness file paths embedded anywhere in system/user content
  // Match paths like docs/reference/xxx.md, AGENTS.md, .skills/xxx/SKILL.md etc.
  const pathPattern = /(?:docs\/(?!plans\/)[\w./-]+\.md|(?:AGENTS|CLAUDE|ARCHITECTURE|README)\.md|\.skills\/[\w./-]+\.md|\.claude\/skills\/[\w./-]+\.md|\.opencode\/skills\/[\w./-]+\.md)/g;
  while ((match = pathPattern.exec(content)) !== null) {
    const rawPath = match[0]!;
    const category = classifyPath(rawPath);
    if (category) {
      const key = normalizePath(rawPath);
      // Only count if not already counted by "Instructions from:" in this same content
      // (avoid double-counting when the path appears in both patterns)
      if (!artifactMap.has(key)) {
        artifactMap.set(key, { path: key, category, readCount: 0, mentionCount: 0, injectedCount: 1, firstReadTurn: null });
      }
      // If already exists but injectedCount is still 0 from a tool call, mark it as injected too
      else if (artifactMap.get(key)!.injectedCount === 0) {
        artifactMap.get(key)!.injectedCount++;
      }
    }
  }
}

/**
 * Extract skill name from a Skill tool call input.
 * Input format varies: could be JSON like {"name":"exec-plan"} or just a string.
 */
function extractSkillName(input: string): string | null {
  if (!input) return null;
  // Try JSON parse
  try {
    const parsed = JSON.parse(input) as Record<string, unknown>;
    if (typeof parsed['name'] === 'string') return parsed['name'];
  } catch { /* not JSON */ }
  // Might be a plain skill name
  if (input.length < 100 && /^[\w-]+$/.test(input.trim())) return input.trim();
  return null;
}
