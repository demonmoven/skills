import { readdir, readFile, stat } from 'node:fs/promises';
import { join, relative, basename } from 'node:path';
import type { WorkDirNode } from './types.js';
import { logger } from '../utils/logger.js';

/**
 * Limits — keep the embedded docs/ tree from ballooning the HTML report.
 */
const MAX_FILE_SIZE = 500 * 1024;       // 500 KB per file
const MAX_TOTAL_SIZE = 10 * 1024 * 1024; // 10 MB total
const MAX_DEPTH = 6;

/**
 * Text file extensions we try to embed.
 * Everything else is represented as a leaf without content.
 */
const TEXT_EXTS = new Set([
  'md', 'mdx', 'txt', 'json', 'yaml', 'yml', 'toml',
  'ts', 'tsx', 'js', 'jsx', 'mjs', 'cjs',
  'go', 'py', 'rs', 'java', 'kt', 'rb', 'php', 'c', 'h', 'cpp', 'hpp',
  'sh', 'bash', 'zsh', 'fish',
  'html', 'css', 'scss', 'sass', 'less',
  'xml', 'svg',
  'sql', 'graphql',
  'conf', 'ini', 'env',
]);

/**
 * Walk <cwd>/docs/ recursively, return a tree with file contents embedded.
 *
 * Returns undefined if `cwd` is nullish or the `docs/` dir doesn't exist.
 * Never throws — returns undefined on any error.
 */
export async function loadWorkDir(cwd: string | undefined): Promise<WorkDirNode | undefined> {
  if (!cwd) return undefined;

  const docsDir = join(cwd, 'docs');
  try {
    const s = await stat(docsDir);
    if (!s.isDirectory()) return undefined;
  } catch {
    logger.debug(`No docs/ directory at ${docsDir}`);
    return undefined;
  }

  const state = { totalSize: 0 };
  const root = await walkDir(docsDir, docsDir, 0, state);
  if (!root) return undefined;
  logger.info(`Loaded work dir tree: ${countFiles(root)} files, ${state.totalSize} bytes embedded`);
  return root;
}

async function walkDir(
  absPath: string,
  docsRoot: string,
  depth: number,
  state: { totalSize: number },
): Promise<WorkDirNode | undefined> {
  if (depth > MAX_DEPTH) return undefined;

  let entries: string[];
  try {
    entries = await readdir(absPath);
  } catch (err) {
    logger.debug(`readdir failed for ${absPath}: ${(err as Error).message}`);
    return undefined;
  }

  const name = depth === 0 ? 'docs' : basename(absPath);
  const rel = depth === 0 ? '' : relative(docsRoot, absPath);
  const node: WorkDirNode = {
    name,
    relativePath: rel,
    isDir: true,
    children: [],
  };

  // Sort: dirs first, then files, both alphabetically
  const sortedChildren: WorkDirNode[] = [];

  const childrenRaw = await Promise.all(
    entries
      .filter((e) => !e.startsWith('.')) // skip hidden
      .map(async (entry) => {
        const childPath = join(absPath, entry);
        let s;
        try {
          s = await stat(childPath);
        } catch {
          return null;
        }
        if (s.isDirectory()) {
          return await walkDir(childPath, docsRoot, depth + 1, state);
        } else if (s.isFile()) {
          return await loadFile(childPath, docsRoot, s.size, state);
        }
        return null;
      }),
  );

  for (const child of childrenRaw) {
    if (child) sortedChildren.push(child);
  }

  // dirs first, then files, alpha sort within each
  sortedChildren.sort((a, b) => {
    if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
    return a.name.localeCompare(b.name);
  });

  node.children = sortedChildren;
  return node;
}

async function loadFile(
  absPath: string,
  docsRoot: string,
  size: number,
  state: { totalSize: number },
): Promise<WorkDirNode> {
  const name = basename(absPath);
  const rel = relative(docsRoot, absPath);
  const node: WorkDirNode = {
    name,
    relativePath: rel,
    isDir: false,
    size,
  };

  const ext = name.includes('.') ? name.split('.').pop()!.toLowerCase() : '';
  if (!TEXT_EXTS.has(ext)) {
    node.contentSkipped = 'binary/unknown extension';
    return node;
  }

  if (size > MAX_FILE_SIZE) {
    node.contentSkipped = `file too large (${(size / 1024).toFixed(1)} KB, limit ${MAX_FILE_SIZE / 1024} KB)`;
    return node;
  }

  if (state.totalSize + size > MAX_TOTAL_SIZE) {
    node.contentSkipped = 'total embedded size limit reached';
    return node;
  }

  try {
    const content = await readFile(absPath, 'utf-8');
    node.content = content;
    state.totalSize += size;
  } catch (err) {
    node.contentSkipped = `read error: ${(err as Error).message}`;
  }

  return node;
}

function countFiles(node: WorkDirNode): number {
  if (!node.isDir) return 1;
  let n = 0;
  for (const child of node.children ?? []) n += countFiles(child);
  return n;
}
