/**
 * lark-cli MCP Server (stdio transport)
 *
 * Wraps lark-cli as a Model Context Protocol server so Claude Desktop's
 * Chat and Cowork sidebars (which can't access the host shell directly) can
 * call lark-cli through the MCP Connector mechanism.
 *
 * Lifecycle: this is spawned as a child process by Claude Desktop based on
 * the entry in ~/Library/Application Support/Claude/claude_desktop_config.json:
 *
 *   {
 *     "mcpServers": {
 *       "lark-cli": {
 *         "command": "node",
 *         "args": ["~/.xdev/lark-mcp-server/index.mjs"]
 *       }
 *     }
 *   }
 *
 * Communication is line-delimited JSON-RPC 2.0 over stdin/stdout. All
 * diagnostic output MUST go to stderr — anything on stdout that isn't a
 * valid JSON-RPC message will corrupt the protocol stream.
 *
 * Why no @modelcontextprotocol/sdk dependency: the MCP protocol surface we
 * need (initialize / tools/list / tools/call / ping) is small enough to
 * implement directly. Keeps the deployed bundle self-contained without
 * pulling in zod and the rest of the SDK transitive deps.
 */

import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { createInterface } from 'node:readline';

const execFileAsync = promisify(execFile);

const SERVER_NAME = 'lark-cli';
const SERVER_VERSION = '1.0.0';
const PROTOCOL_VERSION = '2024-11-05';

// ─── Tool Definition ────────────────────────────────────────────────────

const LARK_CLI_TOOL = {
  name: 'lark_cli',
  description:
    '执行 lark-cli 命令操作飞书（日历、消息、文档、表格、邮件、任务、通讯录、知识库等）。\n' +
    '传入完整子命令和参数（不含 "lark-cli" 前缀）。lark-cli 大多数查询命令默认输出 JSON。\n\n' +
    '常用示例：\n' +
    '- 查日程：calendar +agenda\n' +
    '- 发消息：im +send --chat oc_xxx --text "内容"\n' +
    '- 读文档：docs +fetch --doc https://xxx.feishu.cn/docx/xxx\n' +
    '- 搜消息：im +search --query "关键词"\n' +
    '- 查同事：contact +search-user --query "姓名"\n' +
    '- 查邮件：mail +search --query "关键词"\n' +
    '- 创建文档：docs +create --title "标题" --markdown "# 内容"\n' +
    '- 通用 API：api GET /open-apis/calendar/v4/calendars\n\n' +
    '更多命令查 schema：schema <service.resource.method>\n' +
    '查所有命令：help',
  inputSchema: {
    type: 'object',
    properties: {
      command: {
        type: 'string',
        description: 'lark-cli 子命令及参数（不含 "lark-cli" 前缀），例如 "calendar +agenda --start 2026-04-09"',
      },
      as_identity: {
        type: 'string',
        enum: ['user', 'bot', 'auto'],
        description: '身份类型：user（用户身份）/ bot（应用身份）/ auto（自动）。默认 auto',
      },
    },
    required: ['command'],
  },
};

// ─── JSON-RPC Helpers ────────────────────────────────────────────────────

interface JsonRpcRequest {
  jsonrpc: '2.0';
  id?: string | number | null;
  method: string;
  params?: unknown;
}

interface JsonRpcSuccess {
  jsonrpc: '2.0';
  id: string | number | null;
  result: unknown;
}

interface JsonRpcError {
  jsonrpc: '2.0';
  id: string | number | null;
  error: { code: number; message: string; data?: unknown };
}

function send(message: JsonRpcSuccess | JsonRpcError): void {
  process.stdout.write(JSON.stringify(message) + '\n');
}

function makeResponse(id: string | number | null, result: unknown): JsonRpcSuccess {
  return { jsonrpc: '2.0', id, result };
}

function makeError(
  id: string | number | null,
  code: number,
  message: string,
  data?: unknown,
): JsonRpcError {
  return { jsonrpc: '2.0', id, error: { code, message, data } };
}

// ─── Command Parser ────────────────────────────────────────────────────

/**
 * Minimal POSIX-shell-like splitter for the user-provided `command` arg.
 *
 * Supports single quotes, double quotes, and backslash escapes. Does NOT
 * support shell expansion (no $VAR, no globs, no command substitution) —
 * the input is meant to be lark-cli args, not shell scripts.
 *
 * Example: 'docs +fetch --doc "https://x.feishu.cn/docx/abc"'
 *   → ['docs', '+fetch', '--doc', 'https://x.feishu.cn/docx/abc']
 */
function shellSplit(command: string): string[] {
  const args: string[] = [];
  let current = '';
  let inQuote: false | "'" | '"' = false;

  for (let i = 0; i < command.length; i++) {
    const c = command[i];
    if (inQuote) {
      if (c === inQuote) {
        inQuote = false;
      } else if (c === '\\' && inQuote === '"' && i + 1 < command.length) {
        current += command[++i];
      } else {
        current += c;
      }
    } else if (c === ' ' || c === '\t' || c === '\n') {
      if (current) {
        args.push(current);
        current = '';
      }
    } else if (c === "'" || c === '"') {
      inQuote = c as "'" | '"';
    } else if (c === '\\' && i + 1 < command.length) {
      current += command[++i];
    } else {
      current += c;
    }
  }
  if (current) args.push(current);
  return args;
}

// ─── Tool Handler ────────────────────────────────────────────────────

interface LarkCliArgs {
  command?: unknown;
  as_identity?: unknown;
}

interface ToolResult {
  content: Array<{ type: 'text'; text: string }>;
  isError?: boolean;
}

async function handleLarkCli(rawArgs: LarkCliArgs): Promise<ToolResult> {
  const command = rawArgs?.command;
  if (typeof command !== 'string' || !command.trim()) {
    return {
      content: [{ type: 'text', text: '错误: command 参数必须是非空字符串' }],
      isError: true,
    };
  }

  const cliArgs = shellSplit(command);
  const asIdentity = typeof rawArgs.as_identity === 'string' ? rawArgs.as_identity : undefined;
  if (asIdentity && asIdentity !== 'auto' && !cliArgs.includes('--as')) {
    cliArgs.push('--as', asIdentity);
  }
  // Note: we deliberately do NOT auto-inject `--format json` here. Most lark-cli
  // query commands default to JSON output already, and several non-query
  // subcommands (auth status, config view, schema, help, ...) do not accept
  // a --format flag at all — auto-injecting would break them with "unknown flag".
  // Claude can pass `--format json` explicitly when it needs to override.

  try {
    const { stdout, stderr } = await execFileAsync('lark-cli', cliArgs, {
      timeout: 30_000,
      maxBuffer: 10 * 1024 * 1024, // 10 MB
      env: process.env,
    });

    // lark-cli stderr is often informational (proxy detected, etc).
    // Only treat as error if stdout is empty.
    if (!stdout || !stdout.trim()) {
      if (stderr && stderr.trim()) {
        return formatPossibleAuthError(stderr);
      }
      return { content: [{ type: 'text', text: '(命令成功执行，但无输出)' }] };
    }

    return { content: [{ type: 'text', text: stdout.trim() }] };
  } catch (err) {
    const e = err as NodeJS.ErrnoException & { stderr?: string; stdout?: string; code?: string };

    // lark-cli not installed
    if (e.code === 'ENOENT') {
      return {
        content: [
          {
            type: 'text',
            text:
              'lark-cli 未安装或不在 PATH 中。\n' +
              '请运行：npm install -g @larksuite/cli\n' +
              '或在 macOS 上运行：xdev lark-mcp mac setup',
          },
        ],
        isError: true,
      };
    }

    const stderr = e.stderr ?? '';
    const stdout = e.stdout ?? '';
    const combined = stderr + '\n' + stdout;

    if (combined.includes('permission_violations') || combined.includes('console_url')) {
      return formatPossibleAuthError(combined);
    }

    return {
      content: [
        {
          type: 'text',
          text:
            `lark-cli 执行失败: ${e.message}\n\n` +
            (stdout ? `stdout:\n${stdout}\n\n` : '') +
            (stderr ? `stderr:\n${stderr}` : ''),
        },
      ],
      isError: true,
    };
  }
}

function formatPossibleAuthError(output: string): ToolResult {
  // Try to extract the missing scope from the lark-cli error JSON.
  // lark-cli emits something like:
  //   {"ok":false,"error":{"permission_violations":[{"scopes":["calendar:calendar:readonly"]}]}}
  let hint = '请运行: lark-cli auth login';
  try {
    const scopeMatch = output.match(/"scopes"\s*:\s*\[\s*"([^"]+)"/);
    if (scopeMatch) {
      hint = `请运行: lark-cli auth login --scope "${scopeMatch[1]}"`;
    }
  } catch {
    // ignore
  }

  // Forward the original output too so Claude has full context
  return {
    content: [
      {
        type: 'text',
        text: `权限不足或未登录。${hint}\n\n原始输出:\n${output.trim()}`,
      },
    ],
    isError: true,
  };
}

// ─── Method Dispatch ────────────────────────────────────────────────────

async function handleRequest(
  message: JsonRpcRequest,
): Promise<JsonRpcSuccess | JsonRpcError | null> {
  const { method, id, params } = message;

  // Notifications (no id) get no response. The most common is
  // notifications/initialized, sent by the client after our initialize result.
  if (id === undefined || id === null) {
    return null;
  }

  switch (method) {
    case 'initialize':
      return makeResponse(id, {
        protocolVersion: PROTOCOL_VERSION,
        capabilities: { tools: {} },
        serverInfo: { name: SERVER_NAME, version: SERVER_VERSION },
      });

    case 'tools/list':
      return makeResponse(id, { tools: [LARK_CLI_TOOL] });

    case 'tools/call': {
      const p = params as { name?: string; arguments?: LarkCliArgs } | undefined;
      const toolName = p?.name;
      const toolArgs = p?.arguments ?? {};
      if (toolName !== 'lark_cli') {
        return makeError(id, -32602, `Unknown tool: ${String(toolName)}`);
      }
      const result = await handleLarkCli(toolArgs);
      return makeResponse(id, result);
    }

    case 'ping':
      return makeResponse(id, {});

    default:
      return makeError(id, -32601, `Method not found: ${method}`);
  }
}

// ─── Main Loop ────────────────────────────────────────────────────────────

async function main(): Promise<void> {
  // All diagnostic output goes to stderr; stdout is reserved for JSON-RPC.
  process.stderr.write(`[lark-cli-mcp] ${SERVER_NAME} v${SERVER_VERSION} starting (pid ${process.pid})\n`);

  const rl = createInterface({
    input: process.stdin,
    crlfDelay: Infinity,
  });

  for await (const line of rl) {
    const trimmed = line.trim();
    if (!trimmed) continue;

    let message: JsonRpcRequest;
    try {
      message = JSON.parse(trimmed) as JsonRpcRequest;
    } catch {
      send(makeError(null, -32700, 'Parse error'));
      continue;
    }

    try {
      const response = await handleRequest(message);
      if (response !== null) {
        send(response);
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      send(makeError(message.id ?? null, -32603, `Internal error: ${msg}`));
    }
  }

  process.stderr.write('[lark-cli-mcp] stdin closed, exiting\n');
}

main().catch((err) => {
  process.stderr.write(`[lark-cli-mcp] fatal: ${err instanceof Error ? err.message : String(err)}\n`);
  process.exit(1);
});
