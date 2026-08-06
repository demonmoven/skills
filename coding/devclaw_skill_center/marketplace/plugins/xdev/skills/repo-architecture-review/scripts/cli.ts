#!/usr/bin/env node
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join, resolve } from 'node:path';
import { cpus } from 'node:os';
import { buildRepoProfile } from './l0-prescan/index.js';
import { runAnalyze, runToolsPhase, runReportPhase } from './orchestrator.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const pkg = JSON.parse(readFileSync(join(__dirname, '..', 'package.json'), 'utf-8')) as { version: string };

interface Flags {
  output?: string;
  noBaseline: boolean;
  mode: 'quick' | 'deep';
  resume: boolean;
  disable: Set<string>;
  phase: 'all' | 'tools' | 'report';
}

function parseFlags(argv: string[]): Flags {
  const out: Flags = { noBaseline: false, mode: 'deep', resume: false, disable: new Set(), phase: 'all' };
  for (let i = 0; i < argv.length; i++) {
    if (argv[i] === '--output') { out.output = argv[++i]; }
    else if (argv[i] === '--no-baseline') { out.noBaseline = true; }
    else if (argv[i] === '--mode') { out.mode = (argv[++i] as 'quick' | 'deep'); }
    else if (argv[i] === '--resume') { out.resume = true; }
    else if (argv[i] === '--disable-tool') { out.disable.add(argv[++i]); }
    else if (argv[i] === '--phase') { out.phase = (argv[++i] as 'all' | 'tools' | 'report'); }
  }
  return out;
}

async function main(argv: string[]): Promise<number> {
  if (argv.includes('--version')) {
    console.log(`repo-architecture-review ${pkg.version}`);
    return 0;
  }
  const [cmd, pathArg, ...rest] = argv;
  if (cmd === 'profile') {
    const target = resolve(pathArg ?? '.');
    const profile = await buildRepoProfile(target);
    console.log(JSON.stringify(profile, null, 2));
    return 0;
  }
  if (cmd === 'analyze') {
    const target = resolve(pathArg ?? '.');
    const flags = parseFlags(rest);
    const parallelism = Math.max(1, Number(process.env.ARCHREVIEW_PARALLELISM ?? '16'));
    const output = flags.output ?? join(target, `architecture-analysis-${new Date().toISOString().slice(0, 10)}.md`);

    if (flags.phase === 'tools') {
      // Phase 1: Run L0 (no LLM) + L1 + L2 + grouping.
      // Produces intermediate artifacts; no LLM calls.
      const result = await runToolsPhase({
        rootPath: target,
        disabledToolIds: flags.disable,
        parallelism,
      });
      console.log(`Tools phase complete. Artifacts in: ${result.tmpDir}`);
      console.log(`  Deterministic findings: ${result.deterministicFindingsCount}`);
      console.log(`  Pending LLM groups:     ${result.pendingLlmGroupsCount}`);
      console.log(`Next: process pending-llm-groups.json, write llm-findings.json, then run --phase report`);
      return 0;
    }

    if (flags.phase === 'report') {
      // Phase 3: Read findings and render report.
      // Expects deterministic-findings.json and optionally llm-findings.json.
      const reportPath = await runReportPhase({ rootPath: target, output });
      console.log(`Report: ${reportPath}`);
      return 0;
    }

    // phase === 'all': original full pipeline with internal LLM calls
    const result = await runAnalyze({
      rootPath: target,
      output,
      noBaseline: flags.noBaseline,
      mode: flags.mode,
      resume: flags.resume,
      disabledToolIds: flags.disable,
      parallelism,
    });
    console.log(`Report: ${result.reportPath}`);
    console.log(`Tool runs: ${result.toolRuns.length}`);
    return 0;
  }
  console.error('usage: archreview <profile|analyze> <path> [--output <file>] [--no-baseline] [--mode quick|deep] [--resume] [--disable-tool <id>] [--phase all|tools|report]');
  return 2;
}

main(process.argv.slice(2)).then(process.exit).catch((e) => {
  console.error(e);
  process.exit(1);
});
