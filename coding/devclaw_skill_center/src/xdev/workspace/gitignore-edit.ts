import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

/**
 * Idempotently append `lines` to `<repoRoot>/.gitignore`. Lines already
 * present (compared by trim) are skipped. Returns the lines actually added,
 * so callers can detect whether the file changed and decide whether to
 * stage / commit.
 *
 * If the file doesn't exist it's created. If it does and is missing a
 * trailing newline, one is added before appending.
 */
export function appendGitignoreLines(repoRoot: string, lines: string[]): string[] {
  const path = join(repoRoot, '.gitignore');
  const existing = existsSync(path) ? readFileSync(path, 'utf-8') : '';

  const present = new Set(
    existing
      .split('\n')
      .map((l) => l.trim())
      .filter((l) => l.length > 0),
  );
  const additions = lines.filter((l) => !present.has(l.trim()));
  if (additions.length === 0) return [];

  let next = existing;
  if (next.length > 0 && !next.endsWith('\n')) next += '\n';
  next += additions.join('\n') + '\n';

  writeFileSync(path, next);
  return additions;
}
