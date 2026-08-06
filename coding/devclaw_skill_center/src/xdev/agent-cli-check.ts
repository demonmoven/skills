import { execFileSync } from 'node:child_process';
import type { Agent } from './prompt.js';

interface AgentCliSpec {
  binary: string;
  installHint: string[];
}

const AGENT_CLI: Partial<Record<Agent, AgentCliSpec>> = {
  cc: {
    binary: 'claude',
    installHint: ['npm install -g @anthropic-ai/claude-code'],
  },
  cdx: {
    binary: 'codex',
    installHint: ['npm install -g @openai/codex'],
  },
  coco: {
    binary: 'coco',
    installHint: [
      'sh -c "$(curl -L https://code.byted.org/api/tos-proxy/download/adopt_coco.sh)" \\',
      '  && export PATH=~/.local/bin:$PATH',
    ],
  },
};

export function ensureAgentCliOrExit(agent: Agent): void {
  const spec = AGENT_CLI[agent];
  if (!spec) return;

  try {
    execFileSync(spec.binary, ['--version'], {
      stdio: ['ignore', 'pipe', 'pipe'],
      timeout: 5_000,
    });
    return;
  } catch (err) {
    const e = err as NodeJS.ErrnoException;
    if (e.code !== 'ENOENT') return;
    printMissingCliGuide(agent, spec);
    process.exit(1);
  }
}

function printMissingCliGuide(agent: Agent, spec: AgentCliSpec): void {
  console.error('');
  console.error(`[xdev] ⚠ 未检测到 ${spec.binary} 命令（${agent} 启动所需）。`);
  console.error('[xdev]');
  console.error('[xdev] 安装命令：');
  for (const line of spec.installHint) {
    console.error(`[xdev]   ${line}`);
  }
  console.error('[xdev]');
  console.error('[xdev] 装好后重新运行 xdev 即可。');
  console.error('[xdev] 详见 README 「前置依赖」章节。');
  console.error('');
}
