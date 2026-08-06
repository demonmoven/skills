import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parseJsonl } from './parser.js';
import { detectFormat, normalize, normalizeSessionTree } from './normalizer.js';
import type { SessionTree } from './types.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const fixturesDir = join(__dirname, '..', 'test', 'fixtures');

describe('detectFormat', () => {
  it('detects Claude Code format', () => {
    const content = readFileSync(join(fixturesDir, 'claude-rich-sample.jsonl'), 'utf-8');
    const raw = parseJsonl(content);
    expect(detectFormat(raw)).toBe('claude-code');
  });

  it('detects OpenCode format', () => {
    const content = readFileSync(join(fixturesDir, 'opencode-rich-sample.jsonl'), 'utf-8');
    const raw = parseJsonl(content);
    expect(detectFormat(raw)).toBe('opencode');
  });
});

describe('normalize Claude Code', () => {
  it('normalizes Claude Code messages correctly', () => {
    const content = readFileSync(join(fixturesDir, 'claude-rich-sample.jsonl'), 'utf-8');
    const raw = parseJsonl(content);
    const messages = normalize(raw, 'claude-code');

    expect(messages.length).toBeGreaterThan(0);

    // First message should be user
    expect(messages[0]!.role).toBe('user');
    expect(messages[0]!.content).toBe('Fix the bug in auth.ts');

    // Second message should be assistant with tool call and thinking
    const assistant1 = messages[1]!;
    expect(assistant1.role).toBe('assistant');
    expect(assistant1.thinking).toBeTruthy();
    expect(assistant1.toolCalls).toHaveLength(1);
    expect(assistant1.toolCalls![0]!.name).toBe('Read');
    expect(assistant1.model).toContain('sonnet');
    expect(assistant1.usage).toBeDefined();
    expect(assistant1.usage!.inputTokens).toBe(1000);
    expect(assistant1.usage!.outputTokens).toBe(500);

    // Tool results are merged back onto the assistant message (mergeToolResults post-processing)
    // so the assistant message now has both toolCalls and toolResults
    expect(assistant1.toolResults).toHaveLength(1);
    expect(assistant1.toolResults![0]!.isError).toBe(false);

    // The pure tool_result user message is removed; next message is the second assistant
    const msg2 = messages[2]!;
    expect(msg2.role).toBe('assistant');
  });

  it('skips meta messages', () => {
    const raw = parseJsonl('{"type":"user","isMeta":true,"sessionId":"s","timestamp":"2026-01-01T00:00:00Z","message":{"role":"user","content":"meta"}}');
    const messages = normalize(raw, 'claude-code');
    expect(messages).toHaveLength(0);
  });
});

describe('normalize OpenCode', () => {
  it('normalizes OpenCode messages correctly', () => {
    const content = readFileSync(join(fixturesDir, 'opencode-rich-sample.jsonl'), 'utf-8');
    const raw = parseJsonl(content);
    const messages = normalize(raw, 'opencode');

    expect(messages.length).toBeGreaterThan(0);

    // First message should be user
    expect(messages[0]!.role).toBe('user');
    expect(messages[0]!.content).toBe('Refactor the config module');

    // Second message should be assistant
    expect(messages[1]!.role).toBe('assistant');
  });

  it('attaches tool_use lines to previous assistant message', () => {
    const content = readFileSync(join(fixturesDir, 'opencode-rich-sample.jsonl'), 'utf-8');
    const raw = parseJsonl(content);
    const messages = normalize(raw, 'opencode');

    // The first assistant message should have tool calls attached
    const firstAssistant = messages.find(m => m.role === 'assistant');
    expect(firstAssistant).toBeDefined();
    expect(firstAssistant!.toolCalls).toBeDefined();
    expect(firstAssistant!.toolCalls!.length).toBeGreaterThan(0);
    expect(firstAssistant!.toolCalls![0]!.name).toBe('read');
  });

  it('handles tool_result with is_error', () => {
    const content = readFileSync(join(fixturesDir, 'opencode-rich-sample.jsonl'), 'utf-8');
    const raw = parseJsonl(content);
    const messages = normalize(raw, 'opencode');

    // Find a user message with error tool result
    const withErrors = messages.filter(m =>
      m.toolResults?.some(tr => tr.isError)
    );
    expect(withErrors.length).toBeGreaterThan(0);
  });
});

describe('normalizeSessionTree', () => {
  const mainContent = readFileSync(join(fixturesDir, 'claude-subagent-sample.jsonl'), 'utf-8');
  const subContent = readFileSync(join(fixturesDir, 'subagent-explore.jsonl'), 'utf-8');

  it('normalizes main session and subagent messages', () => {
    const tree: SessionTree = {
      mainJsonlContent: mainContent,
      subagents: [{
        agentId: 'explore-1',
        jsonlContent: subContent,
        meta: { agentType: 'Explore', description: 'Find config files' },
      }],
    };

    const { mainMessages, subagentMessages } = normalizeSessionTree(tree, 'claude-code');

    expect(mainMessages.length).toBeGreaterThan(0);
    // Main messages should not have subagentId
    expect(mainMessages.every(m => m.subagentId === undefined)).toBe(true);

    expect(subagentMessages.size).toBe(1);
    const subMsgs = subagentMessages.get('explore-1')!;
    expect(subMsgs.length).toBeGreaterThan(0);
    // All subagent messages should have subagentId set
    expect(subMsgs.every(m => m.subagentId === 'explore-1')).toBe(true);
  });

  it('returns empty subagentMessages map when no subagents', () => {
    const tree: SessionTree = {
      mainJsonlContent: mainContent,
      subagents: [],
    };

    const { mainMessages, subagentMessages } = normalizeSessionTree(tree, 'claude-code');
    expect(mainMessages.length).toBeGreaterThan(0);
    expect(subagentMessages.size).toBe(0);
  });
});

describe('Agent tool summarization', () => {
  it('summarizes Agent tool call with subagent_type and description', () => {
    const mainContent = readFileSync(join(fixturesDir, 'claude-subagent-sample.jsonl'), 'utf-8');
    const raw = parseJsonl(mainContent);
    const messages = normalize(raw, 'claude-code');

    // Find the assistant message with the Agent tool call
    const withAgent = messages.find(m =>
      m.toolCalls?.some(tc => tc.name === 'Agent')
    );
    expect(withAgent).toBeDefined();

    const agentCall = withAgent!.toolCalls!.find(tc => tc.name === 'Agent')!;
    expect(agentCall.input).toContain('[Subagent: Explore]');
    expect(agentCall.input).toContain('Explore config module');
  });

  it('falls back to "general" when subagent_type is missing', () => {
    const line = JSON.stringify({
      type: 'assistant', sessionId: 's1', uuid: 'u1', timestamp: '2026-01-01T00:00:00Z',
      message: {
        role: 'assistant', model: 'claude-sonnet-4-20250514',
        content: [{ type: 'tool_use', id: 'a1', name: 'Agent', input: { description: 'do stuff', prompt: 'task' } }],
      },
    });
    const raw = parseJsonl(line);
    const messages = normalize(raw, 'claude-code');
    const agentCall = messages[0]!.toolCalls![0]!;
    expect(agentCall.input).toContain('[Subagent: general]');
    expect(agentCall.input).toContain('do stuff');
  });
});

describe('normalizeTraeFromSpans first-occurrence reconstruction', () => {
  function loadFixture(): unknown[] {
    const raw = readFileSync(join(fixturesDir, 'trae-rewrite-and-wrap.json'), 'utf-8');
    return JSON.parse(raw) as unknown[];
  }

  it('uses tool span raw output, not the wrapped/truncated version from next model.input', () => {
    const spans = loadFixture();
    const jsonl = spans.map(s => JSON.stringify(s)).join('\n');
    const messages = normalize(parseJsonl(jsonl), 'trae');

    // The tool result on the assistant message should be the raw "a.txt\nb.txt",
    // not the truncated "[truncated 1 of 2 lines] a.txt" wrapped in <toolcall_status>.
    const firstAssistant = messages.find(m => m.role === 'assistant' && m.toolResults?.length);
    expect(firstAssistant).toBeDefined();
    expect(firstAssistant!.toolResults).toHaveLength(1);
    const tr = firstAssistant!.toolResults![0]!;
    expect(tr.content).toBe('a.txt\nb.txt');
    expect(tr.content).not.toContain('<toolcall_status>');
    expect(tr.content).not.toContain('truncated');
    expect(tr.toolName).toBe('LS');
    expect(tr.isError).toBe(false);
  });

  it('pairs tool spans to assistant tool_calls via tool_call_id_origin (not the rewritten ID)', () => {
    const spans = loadFixture();
    const jsonl = spans.map(s => JSON.stringify(s)).join('\n');
    const messages = normalize(parseJsonl(jsonl), 'trae');

    const firstAssistant = messages.find(m => m.role === 'assistant' && m.toolCalls?.length);
    expect(firstAssistant).toBeDefined();
    // The toolCalls on the merged assistant message should reflect the ORIGIN id from m0.output,
    // because that is the unmodified LLM output. The "rewritten" id is a Trae presentation artifact.
    expect(firstAssistant!.toolCalls![0]!.toolUseId).toBe('call_ORIGIN_A');
  });

  it('captures user messages from the earliest model span where they appear', () => {
    const spans = loadFixture();
    const jsonl = spans.map(s => JSON.stringify(s)).join('\n');
    const messages = normalize(parseJsonl(jsonl), 'trae');

    // Two user messages: "list files" (from m0.input) and "thanks" (first appears in m1.input)
    const userTurns = messages.filter(m => m.role === 'user');
    expect(userTurns.map(m => m.content)).toEqual(['list files', 'thanks']);
  });

  it('takes assistant content from each model span output, in order', () => {
    const spans = loadFixture();
    const jsonl = spans.map(s => JSON.stringify(s)).join('\n');
    const messages = normalize(parseJsonl(jsonl), 'trae');

    const assistants = messages.filter(m => m.role === 'assistant');
    expect(assistants).toHaveLength(2);
    expect(assistants[0]!.content).toBe('running ls');
    expect(assistants[1]!.content).toBe('you are welcome');
  });

  it('attributes per-turn token usage from each originating model span', () => {
    const spans = loadFixture();
    const jsonl = spans.map(s => JSON.stringify(s)).join('\n');
    const messages = normalize(parseJsonl(jsonl), 'trae');

    const assistants = messages.filter(m => m.role === 'assistant');
    expect(assistants[0]!.usage?.inputTokens).toBe(100);
    expect(assistants[0]!.usage?.outputTokens).toBe(20);
    expect(assistants[1]!.usage?.inputTokens).toBe(300);
    expect(assistants[1]!.usage?.outputTokens).toBe(30);
  });

  it('handles the real trae-fornax-sample.json fixture without losing turns', () => {
    const raw = readFileSync(join(fixturesDir, 'trae-fornax-sample.json'), 'utf-8');
    const spans = JSON.parse(raw) as unknown[];
    const jsonl = spans.map(s => JSON.stringify(s)).join('\n');
    const messages = normalize(parseJsonl(jsonl), 'trae');

    expect(messages.length).toBeGreaterThan(0);

    // The fixture has 2 model spans → 2 assistant turns expected.
    const assistants = messages.filter(m => m.role === 'assistant');
    expect(assistants).toHaveLength(2);

    // The fixture has 1 tool span (LS). Its raw output is the bare directory
    // listing. The new algorithm attaches it as a toolResult on the first
    // assistant with the RAW content — not the wrapped "<toolcall_status>"
    // version that appears inside model[1].input.messages.
    const firstAsst = assistants[0]!;
    expect(firstAsst.toolResults).toBeDefined();
    expect(firstAsst.toolResults!.length).toBeGreaterThanOrEqual(1);
    const lsResult = firstAsst.toolResults!.find(r => r.toolName === 'LS');
    expect(lsResult).toBeDefined();
    expect(lsResult!.content).not.toContain('<toolcall_status>');
    expect(lsResult!.content).toContain('e:\\Desktop\\yinxiang');
  });
});
