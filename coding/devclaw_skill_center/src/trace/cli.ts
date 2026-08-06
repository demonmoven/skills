import { Command } from 'commander';
import { runForward } from './commands/forward.js';
import { runSessionStart } from './commands/session-start.js';
import { runAuthLogin, runAuthStatus, runAuthLogout } from './commands/auth.js';
import { runConfigSet, runConfigView } from './commands/config.js';
import { runAnalyze } from './commands/analyze.js';
import { runServe } from './commands/serve.js';
import { runTail } from './commands/tail.js';
import { runUrl } from './commands/url.js';
import { runGet } from './commands/get.js';
import { runDownload } from './commands/download.js';
import { runOpen } from './commands/open.js';

/**
 * Mount session-start / forward / auth / config directly onto `target`.
 * Helper used by registerTraceCommand to attach the actual leaf commands
 * to a `program.command('trace')` subcommand instance.
 */
function mountTraceCommands(target: Command): void {
  target
    .command('session-start')
    .description('Record Git baseline at the start of an AI coding session')
    .requiredOption('--session-id <id>', 'Session ID')
    .option('--cwd <dir>', 'Working directory', process.cwd())
    .action(async (opts: { sessionId: string; cwd: string }) => {
      await runSessionStart(opts.sessionId, opts.cwd);
    });

  target
    .command('forward')
    .description('Collect and upload a session JSONL file to TOS')
    .requiredOption('--file <path>', 'Path to the JSONL session file')
    .option('--source <tool>', 'Explicit tool type (claude-code, opencode)')
    .option('--cwd <dir>', 'Working directory for git context', process.cwd())
    .option('--dry-run', 'Run full pipeline but write to local dir instead of uploading')
    .option('--dry-run-output <dir>', 'Output directory for dry-run mode')
    .action(async (opts: { file: string; source?: string; cwd: string; dryRun?: boolean; dryRunOutput?: string }) => {
      const result = await runForward(opts.file, {
        source: opts.source,
        cwd: opts.cwd,
        dryRun: opts.dryRun,
        dryRunOutput: opts.dryRunOutput,
      });
      console.log(JSON.stringify(result, null, 2));
    });

  target
    .command('analyze')
    .description('Analyze a coding-agent JSONL session and render an interactive HTML report')
    .requiredOption('--file <path>', 'Path to the main LeadAgent JSONL file')
    .option('--agent <type>', 'Coding agent type: cc (claude-code). v1 only supports "cc".', 'cc')
    .option('--output <path>', 'Output HTML path (default: /tmp/xdev-analyze-<session>-<ts>.html)')
    .option('--no-open', 'Do not auto-open the report in browser')
    .action(async (opts: { file: string; agent: string; output?: string; open?: boolean }) => {
      await runAnalyze({
        file: opts.file,
        agent: opts.agent,
        output: opts.output,
        open: opts.open,
      });
    });

  target
    .command('serve')
    .description('Start the Web Dashboard for session analysis')
    .option('--port <port>', 'Server port', '3210')
    .option('--no-open', 'Do not open browser automatically')
    .action(async (opts: { port: string; open: boolean }) => {
      await runServe({ port: opts.port, open: opts.open });
    });

  target
    .command('url')
    .description('Generate a pre-signed URL for a TOS object (shareable, default 24h expiry)')
    .argument('<key>', 'TOS object key (leading / is allowed and will be trimmed)')
    .option('--expires <duration>', 'URL expiry (e.g. 3600, 1h, 24h, 7d)', '24h')
    .option('--open', 'open the URL in the default browser')
    .option('--copy', 'copy the URL to clipboard (macOS pbcopy / Linux xclip / Windows clip)')
    .action(async (key: string, opts: { expires?: string; open?: boolean; copy?: boolean }) => {
      await runUrl(key, opts);
    });

  target
    .command('get')
    .description('Download a TOS object to a local file')
    .argument('<key>', 'TOS object key (leading / is allowed and will be trimmed)')
    .option('-o, --output <path>', 'output path (default: ./<basename-from-key>)')
    .option('--gunzip', 'gunzip after download; strips trailing .gz from default output name')
    .action(async (key: string, opts: { output?: string; gunzip?: boolean }) => {
      await runGet(key, opts);
    });

  target
    .command('download')
    .description('Batch download recent forward:ok objects from hook logs')
    .option('-n, --lines <n>', 'number of most-recent unique sessions to act on', '5')
    .option('--save <dir>', 'save objects to directory (default action when none of --save/--open/--url given)')
    .option('--open', 'open each pre-signed URL in the default browser')
    .option('--url', 'print each pre-signed URL to stdout, one per line')
    .option('--with-diff', 'also include the associated diff.gz object for each session')
    .option('--tool <name>', 'filter by tool (claude-code / opencode / trae)')
    .option('--session <id>', 'filter by sessionId (prefix match allowed)')
    .option('--date <yyyy-mm-dd>', 'only look at logs of a specific date (default: scan recent days)')
    .option('--expires <duration>', 'URL expiry when --open or --url (default 24h)', '24h')
    .action(
      async (opts: {
        lines: string;
        save?: string;
        open?: boolean;
        url?: boolean;
        withDiff?: boolean;
        tool?: string;
        session?: string;
        date?: string;
        expires?: string;
      }) => {
        const n = parseInt(opts.lines, 10);
        if (isNaN(n) || n <= 0) throw new Error(`Invalid --lines value: ${opts.lines}`);
        const save = opts.save !== undefined ? opts.save : opts.open || opts.url ? undefined : '.';
        await runDownload({
          lines: n,
          save,
          open: opts.open,
          url: opts.url,
          withDiff: opts.withDiff,
          tool: opts.tool,
          session: opts.session,
          date: opts.date,
          expires: opts.expires,
        });
      },
    );

  target
    .command('open')
    .description('Fetch recent forward:ok objects and open each pre-signed URL in the default browser (傻瓜模式)')
    .option('-n, --lines <n>', 'number of recent unique sessions to open', '5')
    .option('--with-diff', 'also open diff.gz for each session')
    .option('--tool <name>', 'filter by tool (claude-code / opencode / trae)')
    .option('--session <id>', 'filter by sessionId (prefix match)')
    .option('--date <yyyy-mm-dd>', 'only scan logs of a specific date')
    .option('--expires <duration>', 'URL expiry (default 24h)', '24h')
    .action(
      async (opts: {
        lines: string;
        withDiff?: boolean;
        tool?: string;
        session?: string;
        date?: string;
        expires?: string;
      }) => {
        const n = parseInt(opts.lines, 10);
        if (isNaN(n) || n <= 0) throw new Error(`Invalid --lines value: ${opts.lines}`);
        await runOpen({
          lines: n,
          withDiff: opts.withDiff,
          tool: opts.tool,
          session: opts.session,
          date: opts.date,
          expires: opts.expires,
        });
      },
    );

  target
    .command('tail')
    .description('Tail xdev trace execution log (~/.trace/logs/hook.YYYY-MM-DD.log)')
    .option('-n, --lines <n>', 'number of lines to show', '10')
    .option('-f, --follow', "follow appended lines of today's log")
    .option('-F, --follow-retry', 'follow with automatic rollover when date changes')
    .option('--date <yyyy-mm-dd>', 'show log for a specific date instead of today')
    .action(async (opts: { lines: string; follow?: boolean; followRetry?: boolean; date?: string }) => {
      const lines = parseInt(opts.lines, 10);
      if (isNaN(lines) || lines < 0) {
        throw new Error(`Invalid --lines value: ${opts.lines}`);
      }
      await runTail({
        lines,
        follow: Boolean(opts.follow),
        followRetry: Boolean(opts.followRetry),
        date: opts.date,
      });
    });

  const auth = target
    .command('auth')
    .description('Manage authentication');

  auth
    .command('login')
    .description('Log in via Device Code SSO')
    .action(async () => {
      await runAuthLogin();
    });

  auth
    .command('status')
    .description('Show current authentication status')
    .action(async () => {
      await runAuthStatus();
    });

  auth
    .command('logout')
    .description('Log out and remove local credentials')
    .action(async () => {
      await runAuthLogout();
    });

  const config = target
    .command('config')
    .description('Manage configuration');

  config
    .command('set')
    .description('Set a configuration value')
    .argument('<key>', 'Configuration key (e.g. tos.bucket)')
    .argument('<value>', 'Configuration value')
    .action(async (key: string, value: string) => {
      await runConfigSet(key, value);
    });

  config
    .command('view')
    .description('View current effective configuration')
    .action(async () => {
      await runConfigView();
    });
}

/**
 * Mount trace as a `trace` subcommand group on the unified xdev CLI.
 * Use case: `xdev trace forward --file ...`
 */
export function registerTraceCommand(program: Command): void {
  const trace = program
    .command('trace')
    .description('AI Coding 会话 trace 采集与上报（formerly xtrace）');
  mountTraceCommands(trace);
}
