import { simpleGit } from 'simple-git';

export async function resolveRenames(repoRoot: string, oldPaths: string[]): Promise<Map<string, string>> {
  const git = simpleGit(repoRoot);
  const out = new Map<string, string>();

  try {
    // Get all rename records across the entire repo history
    const log = await git.raw(['log', '--all', '--oneline', '--name-status']);
    const lines = log.split('\n');

    // Build a map of old path -> new path from all rename records
    const renames = new Map<string, string>();
    for (const line of lines) {
      if (/^R\d+/.test(line)) {
        const parts = line.split('\t');
        if (parts.length >= 3) {
          renames.set(parts[1], parts[2]);
        }
      }
    }

    // For each input path, return the new path if found
    for (const p of oldPaths) {
      out.set(p, renames.get(p) ?? p);
    }
  } catch {
    // On error, return identity mapping
    for (const p of oldPaths) {
      out.set(p, p);
    }
  }

  return out;
}
