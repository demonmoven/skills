import { existsSync, mkdirSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import type { ScaffoldFlags } from './types.js';

/**
 * Create the workspace skeleton (AGENTS.md / docs/ / exec-plans/ /
 * onboarding/ / scripts/setup-workspace.sh) under `root`.
 *
 * All files are created as placeholders (single H1 heading or TODO comment
 * for shell scripts). Existing files are NOT overwritten — call site can
 * re-run safely without losing user content.
 */
export function createWorkspaceScaffold(root: string, flags: ScaffoldFlags = {}): string[] {
  const created: string[] = [];

  const writePlaceholder = (relPath: string, content: string): void => {
    const abs = join(root, relPath);
    if (existsSync(abs)) return;
    mkdirSync(dirname(abs), { recursive: true });
    writeFileSync(abs, content);
    created.push(relPath);
  };

  const writeKeepDir = (relDir: string): void => {
    const abs = join(root, relDir, '.gitkeep');
    if (existsSync(abs)) return;
    mkdirSync(join(root, relDir), { recursive: true });
    writeFileSync(abs, '');
    created.push(`${relDir}/.gitkeep`);
  };

  if (flags.agents_md !== false) {
    writePlaceholder('AGENTS.md', '# AGENTS\n\n> 跨仓库 Agent 入口（占位）。\n');
  }

  if (flags.docs_tree !== false) {
    writePlaceholder('docs/system-architecture.md', '# System Architecture\n\n> 系统架构总览（占位）。\n');
    writePlaceholder('docs/glossary.md', '# Glossary\n\n> 业务领域词汇表（占位）。\n');
    writePlaceholder('docs/conventions/error-codes.md', '# Error Codes\n');
    writePlaceholder('docs/conventions/idl-guidelines.md', '# IDL Guidelines\n');
    writePlaceholder('docs/conventions/cross-repo-workflow.md', '# Cross-Repo Workflow\n');
    writePlaceholder('docs/cross-repo/api-creativity-flow.md', '# API ↔ Creativity Flow\n');
    writePlaceholder('docs/cross-repo/agent-creativity-rpc-contract.md', '# Agent ↔ Creativity RPC Contract\n');
    writeKeepDir('docs/adr');
  }

  if (flags.exec_plans !== false) writeKeepDir('exec-plans');
  if (flags.onboarding !== false) writeKeepDir('onboarding');

  if (flags.setup_script !== false) {
    writePlaceholder(
      'scripts/setup-workspace.sh',
      [
        '#!/usr/bin/env bash',
        '# 占位脚本：未来填入「clone 全部业务仓库 → 创建结构 → commit + push」逻辑。',
        '# 现阶段 xdev workspace create 已经做了这些事情，此脚本作为团队新人 onboarding 用。',
        'set -euo pipefail',
        'echo "TODO: implement workspace bootstrap"',
        '',
      ].join('\n'),
    );
  }

  return created;
}
