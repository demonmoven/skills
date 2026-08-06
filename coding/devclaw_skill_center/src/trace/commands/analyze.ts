import { access, constants, writeFile, mkdir } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { tmpdir } from 'node:os';
import { parseJsonl, loadSubagents } from '../analyze/parser.js';
import { processSession } from '../analyze/processor.js';
import { computeSummary } from '../analyze/summary.js';
import { loadWorkDir } from '../analyze/work-dir.js';
import { renderHtml } from '../analyze/html-renderer.js';
import { openInBrowser } from '../analyze/browser.js';
import { FileNotFoundError } from '../utils/errors.js';
import { logger } from '../utils/logger.js';

export interface AnalyzeOptions {
  agent: string;
  file: string;
  output?: string;
  open?: boolean;
}

const SUPPORTED_AGENTS = ['cc'] as const;

/**
 * Analyze a coding-agent session file and produce a single-file HTML report.
 *
 * v1 only supports Claude Code (`--agent cc`). Other agents (`cdx`, `coco`, ...)
 * will be added in future iterations.
 */
export async function runAnalyze(options: AnalyzeOptions): Promise<void> {
  // 1. Validate agent type
  if (!SUPPORTED_AGENTS.includes(options.agent as (typeof SUPPORTED_AGENTS)[number])) {
    throw new Error(
      `Unsupported --agent "${options.agent}". v1 supports: ${SUPPORTED_AGENTS.join(', ')}`,
    );
  }

  // 2. Validate input file
  const filePath = options.file;
  try {
    await access(filePath, constants.R_OK);
  } catch {
    throw new FileNotFoundError(filePath);
  }

  logger.info(`Analyzing: ${filePath}`);

  // 3. Parse main JSONL
  const mainEntries = await parseJsonl(filePath);
  logger.info(`Parsed ${mainEntries.length} entries from main JSONL`);

  // 4. Load subagent traces (if any)
  const subagents = await loadSubagents(filePath);
  logger.info(`Loaded ${subagents.size} subagent trace(s)`);

  // 5. Process into enriched model
  const result = processSession(mainEntries, subagents, filePath);
  logger.info(`Processed ${result.mainEntries.length} main entries, ${result.subagents.size} subagents`);

  // 6. Load work dir (docs/ under session cwd)
  const workDir = await loadWorkDir(result.cwd);
  if (workDir) {
    result.workDir = workDir;
    logger.info(`Work dir loaded from ${result.cwd}/docs/`);
  } else if (result.cwd) {
    logger.info(`No docs/ directory under session cwd (${result.cwd})`);
  }

  // 7. Compute summary
  const summary = computeSummary(result);
  result.summary = summary;
  logger.info(
    `Summary: ${summary.totalTurns} turns, ${summary.totalToolCalls} tool calls, ` +
      `${summary.tokenTotals.input + summary.tokenTotals.output} tokens, ` +
      `${summary.errorCount} errors`,
  );

  // 7. Render HTML
  const html = renderHtml(result);

  // 8. Determine output path
  const outputPath = options.output ?? defaultOutputPath(filePath);
  await mkdir(dirname(outputPath), { recursive: true });
  await writeFile(outputPath, html, 'utf-8');
  logger.info(`Report written: ${outputPath}`);

  // 9. Open in browser unless --no-open
  if (options.open !== false) {
    try {
      await openInBrowser(outputPath);
      logger.info('Opened in browser.');
    } catch (err) {
      logger.warn(`Could not auto-open browser: ${(err as Error).message}`);
      logger.warn(`Open manually: file://${outputPath}`);
    }
  }
}

function defaultOutputPath(inputPath: string): string {
  const base = inputPath.split('/').pop()?.replace(/\.jsonl$/, '') ?? 'trace';
  const ts = new Date().toISOString().replace(/[:.]/g, '-');
  return join(tmpdir(), `xdev-analyze-${base}-${ts}.html`);
}
