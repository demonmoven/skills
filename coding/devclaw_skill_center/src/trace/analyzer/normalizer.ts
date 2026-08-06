import type { RawJsonlLine, NormalizedMessage, SessionTree } from './types.js';
import { parseJsonl } from './parser.js';

type ToolFormat = 'claude-code' | 'opencode' | 'trae';

/**
 * Detect the JSONL format by inspecting the first few messages.
 * - Claude Code: has `sessionId` (camelCase) and nested `message` object
 * - OpenCode: has `info` + `parts` structure (real format), or `session_id` (test fixtures)
 */
export function detectFormat(rawMessages: RawJsonlLine[]): ToolFormat {
  for (const msg of rawMessages.slice(0, 10)) {
    // Claude Code: { type, sessionId, message: {...} }
    if ('sessionId' in msg && 'message' in msg) return 'claude-code';
    // OpenCode real format: { info: { role, ... }, parts: [...] }
    if ('info' in msg && 'parts' in msg) return 'opencode';
    // OpenCode test fixture format: { session_id, type: "message", role, content }
    if ('session_id' in msg) return 'opencode';
    if (msg['type'] === 'message' && 'role' in msg) return 'opencode';
    // Trae (via Fornax): spans with trace_id + span_type, or pre-converted JSONL with _trae_format marker
    if ('_trae_format' in msg) return 'trae';
    if ('trace_id' in msg && 'span_type' in msg) return 'trae';
  }
  return 'claude-code';
}

/**
 * Normalize JSONL messages into the unified NormalizedMessage format.
 * Automatically detects format or uses the provided toolName.
 *
 * Post-processing merges tool results back onto the assistant message
 * that initiated the tool call, ensuring a consistent structure across
 * all tool formats: toolCalls and toolResults live on the same message.
 */
export function normalize(
  rawMessages: RawJsonlLine[],
  toolName?: string,
): NormalizedMessage[] {
  const format = toolName as ToolFormat | undefined ?? detectFormat(rawMessages);
  let messages: NormalizedMessage[];
  switch (format) {
    case 'claude-code':
      messages = normalizeClaudeCode(rawMessages);
      break;
    case 'opencode':
      messages = normalizeOpenCode(rawMessages);
      break;
    case 'trae':
      messages = normalizeTrae(rawMessages);
      break;
    default:
      messages = normalizeClaudeCode(rawMessages);
  }
  return mergeToolResults(messages);
}

/**
 * Normalize a full session tree (main session + subagents).
 * The main session is normalized as usual; each subagent's messages
 * are normalized separately and tagged with `subagentId`.
 *
 * @returns mainMessages and a Map of subagent messages keyed by agentId
 */
export function normalizeSessionTree(
  tree: SessionTree,
  toolName?: string,
): { mainMessages: NormalizedMessage[]; subagentMessages: Map<string, NormalizedMessage[]> } {
  // 1. Normalize the main JSONL
  const mainRaw = parseJsonl(tree.mainJsonlContent);
  const mainMessages = normalize(mainRaw, toolName);

  // 2. Normalize each subagent and tag messages with subagentId
  const subagentMessages = new Map<string, NormalizedMessage[]>();
  for (const sub of tree.subagents) {
    const subRaw = parseJsonl(sub.jsonlContent);
    const subMsgs = normalize(subRaw, toolName);
    for (const msg of subMsgs) {
      msg.subagentId = sub.agentId;
    }
    subagentMessages.set(sub.agentId, subMsgs);
  }

  return { mainMessages, subagentMessages };
}

/**
 * Post-process: merge user messages that contain ONLY tool results (no text content)
 * back onto the preceding assistant message. This unifies the rendering so that
 * tool calls and their results always appear together on the assistant message,
 * regardless of the original tool format.
 *
 * User messages that have actual text content (user typed something) are kept as-is.
 */
function mergeToolResults(messages: NormalizedMessage[]): NormalizedMessage[] {
  const result: NormalizedMessage[] = [];

  for (const msg of messages) {
    // A user message that has toolResults but no meaningful text content
    // is a "pure tool result carrier" — merge it into the previous assistant message
    if (
      msg.role === 'user' &&
      msg.toolResults &&
      msg.toolResults.length > 0 &&
      !msg.content.trim()
    ) {
      // Find the last assistant message in result
      for (let i = result.length - 1; i >= 0; i--) {
        if (result[i]!.role === 'assistant') {
          if (!result[i]!.toolResults) result[i]!.toolResults = [];
          result[i]!.toolResults!.push(...msg.toolResults);
          break;
        }
      }
      // Don't add this empty user message to the output
      continue;
    }

    result.push(msg);
  }

  return result;
}

// ─── Claude Code Normalizer ────────────────────────────────────────

function normalizeClaudeCode(rawMessages: RawJsonlLine[]): NormalizedMessage[] {
  const results: NormalizedMessage[] = [];

  for (const raw of rawMessages) {
    const type = raw['type'] as string | undefined;
    const timestamp = (raw['timestamp'] as string) ?? '';

    if (type === 'user') {
      if (raw['isMeta']) continue;
      const message = raw['message'] as Record<string, unknown> | undefined;
      if (!message) continue;

      const content = message['content'];
      const normalized: NormalizedMessage = {
        role: 'user',
        timestamp,
        content: '',
      };

      if (typeof content === 'string') {
        normalized.content = content;
      } else if (Array.isArray(content)) {
        const textParts: string[] = [];
        const toolResults: NormalizedMessage['toolResults'] = [];

        for (const block of content) {
          if (typeof block !== 'object' || block === null) continue;
          const b = block as Record<string, unknown>;
          if (b['type'] === 'text') {
            textParts.push(b['text'] as string ?? '');
          } else if (b['type'] === 'tool_result') {
            const resultContent = b['content'];
            let contentStr = '';
            if (typeof resultContent === 'string') {
              contentStr = resultContent;
            } else if (Array.isArray(resultContent)) {
              contentStr = (resultContent as Array<Record<string, unknown>>)
                .map(rc => (rc['text'] as string) ?? '')
                .join(' ');
            }
            toolResults.push({
              toolName: (b['tool_use_id'] as string) ?? 'unknown',
              isError: (b['is_error'] as boolean) ?? false,
              content: contentStr,
            });
          }
        }

        // Extract agentId from toolUseResult (Claude Code attaches this on Agent tool results)
        const toolUseResult = raw['toolUseResult'] as Record<string, unknown> | undefined;
        if (toolUseResult?.['agentId'] && toolResults.length > 0) {
          const lastResult = toolResults[toolResults.length - 1]!;
          lastResult.agentId = toolUseResult['agentId'] as string;
        }

        normalized.content = textParts.join('\n');
        if (toolResults.length > 0) normalized.toolResults = toolResults;
      }

      results.push(normalized);
    } else if (type === 'assistant') {
      const message = raw['message'] as Record<string, unknown> | undefined;
      if (!message) continue;

      const model = message['model'] as string | undefined;
      const usage = message['usage'] as Record<string, number> | undefined;
      const content = message['content'];

      const normalized: NormalizedMessage = {
        role: 'assistant',
        timestamp,
        content: '',
        model,
      };

      if (usage) {
        normalized.usage = {
          inputTokens: usage['input_tokens'] ?? 0,
          outputTokens: usage['output_tokens'] ?? 0,
          cacheWriteTokens: usage['cache_creation_input_tokens'] ?? 0,
          cacheReadTokens: usage['cache_read_input_tokens'] ?? 0,
        };
      }

      if (typeof content === 'string') {
        normalized.content = content;
      } else if (Array.isArray(content)) {
        const textParts: string[] = [];
        const toolCalls: NormalizedMessage['toolCalls'] = [];

        for (const block of content) {
          if (typeof block !== 'object' || block === null) continue;
          const b = block as Record<string, unknown>;
          if (b['type'] === 'text') {
            textParts.push(b['text'] as string ?? '');
          } else if (b['type'] === 'tool_use') {
            const name = b['name'] as string ?? 'unknown';
            const input = b['input'] as Record<string, unknown> | undefined;
            const toolUseId = b['id'] as string | undefined;
            toolCalls.push({
              name,
              input: summarizeToolInput(name, input),
              ...(toolUseId ? { toolUseId } : {}),
            });
          } else if (b['type'] === 'thinking') {
            normalized.thinking = (b['text'] as string) ?? '';
          }
        }

        normalized.content = textParts.join('\n');
        if (toolCalls.length > 0) normalized.toolCalls = toolCalls;
      }

      results.push(normalized);
    } else if (type === 'system') {
      results.push({
        role: 'system',
        timestamp,
        content: typeof raw['message'] === 'string'
          ? raw['message']
          : JSON.stringify(raw['message'] ?? ''),
      });
    }
  }

  return results;
}

// ─── OpenCode Normalizer ───────────────────────────────────────────

/**
 * Real OpenCode JSONL format:
 *   { info: { role, time: { created }, tokens: { input, output, cache: { write, read } }, cost, modelID, ... },
 *     parts: [ { type: "text", text }, { type: "tool", tool, state: { status, input, output, title }, callID }, ... ] }
 */
function normalizeOpenCode(rawMessages: RawJsonlLine[]): NormalizedMessage[] {
  const results: NormalizedMessage[] = [];

  for (const raw of rawMessages) {
    // Real OpenCode format: { info, parts }
    if ('info' in raw && 'parts' in raw) {
      const info = raw['info'] as Record<string, unknown>;
      const parts = raw['parts'] as Array<Record<string, unknown>> ?? [];
      const role = info['role'] as string;

      if (role !== 'user' && role !== 'assistant') continue;

      // Timestamp from info.time.created (Unix ms)
      const time = info['time'] as Record<string, unknown> | undefined;
      const createdMs = time?.['created'] as number | undefined;
      const timestamp = createdMs ? new Date(createdMs).toISOString() : '';

      // Model
      const model = (info['modelID'] as string) ?? (info['model'] as Record<string, unknown>)?.['modelID'] as string ?? undefined;

      const normalized: NormalizedMessage = {
        role: role as 'user' | 'assistant',
        timestamp,
        content: '',
        model,
      };

      // Extract tokens/cost from info (assistant messages)
      const tokens = info['tokens'] as Record<string, unknown> | undefined;
      const cost = info['cost'] as number | undefined;
      if (tokens) {
        const cache = tokens['cache'] as Record<string, number> | undefined;
        normalized.usage = {
          inputTokens: (tokens['input'] as number) ?? 0,
          outputTokens: (tokens['output'] as number) ?? 0,
          cacheWriteTokens: cache?.['write'] ?? 0,
          cacheReadTokens: cache?.['read'] ?? 0,
        };
      }
      // If we have cost but no tokens, still mark usage as present with cost info
      if (cost !== undefined && cost > 0 && !normalized.usage) {
        normalized.usage = {
          inputTokens: 0,
          outputTokens: 0,
          cacheWriteTokens: 0,
          cacheReadTokens: 0,
        };
      }

      // Process parts
      const textParts: string[] = [];
      const toolCalls: NormalizedMessage['toolCalls'] = [];
      const toolResults: NormalizedMessage['toolResults'] = [];

      for (const part of parts) {
        const partType = part['type'] as string;

        if (partType === 'text') {
          const text = (part['text'] as string) ?? '';
          if (text.trim()) textParts.push(text);
        } else if (partType === 'tool') {
          const toolName = (part['tool'] as string) ?? 'unknown';
          const state = part['state'] as Record<string, unknown> | undefined;
          const status = state?.['status'] as string | undefined;
          const input = state?.['input'] as Record<string, unknown> | undefined;
          const output = (state?.['output'] as string) ?? '';
          const title = state?.['title'] as string | undefined;

          // Build a descriptive input string: title + full input details
          let inputStr = title ?? summarizeToolInput(toolName, input);
          if (input && Object.keys(input).length > 0) {
            // Append the full input params for expandability
            const fullInput = JSON.stringify(input, null, 2);
            if (fullInput.length > inputStr.length + 10) {
              inputStr = inputStr + '\n' + fullInput;
            }
          }

          toolCalls.push({
            name: toolName,
            input: inputStr,
          });

          // Record tool result with output (for both completed and error)
          if (status === 'completed' || status === 'error') {
            const isError = status === 'error';
            toolResults.push({
              toolName,
              isError,
              content: output,
            });
          }
        }
      }

      normalized.content = textParts.join('\n');
      if (toolCalls.length > 0) normalized.toolCalls = toolCalls;
      // OpenCode: tool results live in the same assistant message as tool calls
      if (toolResults.length > 0) normalized.toolResults = toolResults;

      results.push(normalized);
      continue;
    }

    // Fallback: test fixture format { session_id, type: "message"|"tool_use", role, content }
    const type = raw['type'] as string | undefined;
    const timestamp = (raw['timestamp'] as string) ?? (raw['created_at'] as string) ?? '';

    if (type === 'message') {
      const role = raw['role'] as string;
      if (role === 'user' || role === 'assistant') {
        const normalized: NormalizedMessage = {
          role: role as 'user' | 'assistant',
          timestamp,
          content: (raw['content'] as string) ?? '',
          model: raw['model'] as string | undefined,
        };
        results.push(normalized);
      } else if (role === 'system') {
        results.push({ role: 'system', timestamp, content: (raw['content'] as string) ?? '' });
      }
    } else if (type === 'tool_use') {
      const toolName = (raw['tool'] as string) ?? (raw['name'] as string) ?? 'unknown';
      const input = (raw['input'] as Record<string, unknown>) ?? {};
      for (let i = results.length - 1; i >= 0; i--) {
        if (results[i]!.role === 'assistant') {
          if (!results[i]!.toolCalls) results[i]!.toolCalls = [];
          results[i]!.toolCalls!.push({ name: toolName, input: summarizeToolInput(toolName, input) });
          break;
        }
      }
    } else if (type === 'tool_result') {
      const toolName = (raw['tool'] as string) ?? (raw['name'] as string) ?? 'unknown';
      const isError = (raw['is_error'] as boolean) ?? (raw['error'] as boolean) ?? false;
      const content = (raw['content'] as string) ?? (raw['output'] as string) ?? '';
      let attached = false;
      for (let i = results.length - 1; i >= 0; i--) {
        if (results[i]!.role === 'user') {
          if (!results[i]!.toolResults) results[i]!.toolResults = [];
          results[i]!.toolResults!.push({ toolName, isError, content });
          attached = true;
          break;
        }
      }
      if (!attached) {
        results.push({ role: 'user', timestamp, content: '', toolResults: [{ toolName, isError, content }] });
      }
    }
  }

  return results;
}

// ─── Trae Normalizer (via Fornax trace) ───────────────────────────

/**
 * Normalize Trae trace data that has been pre-converted to JSONL.
 *
 * The import endpoint converts Fornax spans to JSONL before storage.
 * Two input formats are supported:
 *
 * 1. **Pre-converted JSONL** (from import endpoint): each line is a
 *    NormalizedMessage-like object with `_trae_format: true` marker.
 *    These are passed through directly after stripping the marker.
 *
 * 2. **Raw Fornax spans** (trace_id + span_type): the function picks the
 *    last `type === "model"` span, reconstructs the full conversation from
 *    its `input.messages` + `output.choices[0].message`, and appends any
 *    trailing tool span results that occurred after the last LLM call.
 */
function normalizeTrae(rawMessages: RawJsonlLine[]): NormalizedMessage[] {
  // Case 1: pre-converted JSONL from import endpoint
  if (rawMessages.length > 0 && rawMessages[0]!['_trae_format'] === true) {
    return rawMessages
      .filter(raw => !raw['_trae_trace_boundary']) // skip boundary markers between traces
      .map(raw => {
        // Ensure toolCalls[].input is always a string (may be object from serialization)
        const toolCalls = raw['toolCalls'] as NormalizedMessage['toolCalls'];
        if (toolCalls) {
          for (const tc of toolCalls) {
            if (typeof tc.input !== 'string') {
              tc.input = JSON.stringify(tc.input);
            }
          }
        }
        return {
          role: (raw['role'] as string) as 'user' | 'assistant' | 'system',
          timestamp: (raw['timestamp'] as string) ?? '',
          content: (raw['content'] as string) ?? '',
          toolCalls,
          toolResults: raw['toolResults'] as NormalizedMessage['toolResults'],
          thinking: raw['thinking'] as string | undefined,
          model: raw['model'] as string | undefined,
          usage: raw['usage'] as NormalizedMessage['usage'],
        };
      });
  }

  // Case 2: raw Fornax spans — reconstruct conversation from the last model span
  return normalizeTraeFromSpans(rawMessages);
}

/** Parse a JSON-encoded string field from a Fornax span; return {} on failure. */
function safeParseJson(value: unknown): Record<string, unknown> {
  if (typeof value !== 'string' || value.length === 0) return {};
  try {
    return JSON.parse(value) as Record<string, unknown>;
  } catch {
    return {};
  }
}

/**
 * Convert raw Fornax span array into NormalizedMessage[] using a first-occurrence
 * reconstruction strategy.
 *
 * Trae uses rolling context where each model call's input.messages includes the
 * full (potentially rewritten/truncated) history. Instead of reading everything
 * from the last model span's input, this walker pulls each piece of content from
 * its earliest unmodified source:
 *   - Each assistant turn comes from its own model span's output (LLM-original).
 *   - Each tool result comes from the tool span's raw output field, paired to the
 *     assistant tool_call via custom_tags.tool_call_id_origin — bypassing Trae's
 *     tool_call_id rewriting when it injects tool results into the next input.
 *   - System and leading user messages come from the earliest model span's input.
 *   - Inter-turn user messages come from the trailing-user position of the next
 *     model span's input.messages.
 */
function normalizeTraeFromSpans(spans: RawJsonlLine[]): NormalizedMessage[] {
  // Sort all spans chronologically. We will walk model spans in order, taking
  // each piece of content from its earliest unmodified source.
  const modelSpans = spans
    .filter(s => s['type'] === 'model')
    .sort((a, b) => parseInt(String(a['started_at'] ?? '0'), 10) - parseInt(String(b['started_at'] ?? '0'), 10));
  if (modelSpans.length === 0) return [];

  const toolSpans = spans
    .filter(s => s['span_type'] === 'tool')
    .sort((a, b) => parseInt(String(a['started_at'] ?? '0'), 10) - parseInt(String(b['started_at'] ?? '0'), 10));

  // Index tool spans by their ORIGIN tool_call_id (matches the assistant
  // tool_calls[].id in the model span's *output*). Trae rewrites IDs when
  // injecting tool results into the next model.input, but the tool span
  // preserves both: tool_call_id_origin and tool_call_id_formatted.
  const toolByOriginId = new Map<string, RawJsonlLine>();
  for (const t of toolSpans) {
    const tags = t['custom_tags'] as Record<string, string> | undefined;
    const originId = tags?.['tool_call_id_origin'] ?? tags?.['tool_call_id'];
    if (originId) toolByOriginId.set(originId, t);
  }

  const results: NormalizedMessage[] = [];

  // Seed: take system + leading user messages from the earliest model span's
  // input.messages. These have not yet had a chance to be rewritten.
  const firstInput = safeParseJson(modelSpans[0]!['input']);
  const firstMessages = (firstInput['messages'] as Array<Record<string, unknown>>) ?? [];
  for (const m of firstMessages) {
    if (m['role'] === 'assistant') break;
    const normalized = convertOpenAIMessage(m);
    if (normalized) results.push(normalized);
  }

  // Walk model spans. For each Mi:
  //   - Append Mi.output as the i-th assistant turn (uncompressed).
  //   - Pair tool spans whose started_at falls in [Mi, Mi+1) by tool_call_id_origin
  //     to Mi.output.tool_calls[].id; emit each as a tool message using the tool
  //     span's RAW output.
  //   - From Mi+1.input.messages, scan for any user message that did not appear
  //     in Mi.input.messages — emit it as a new user turn.
  for (let i = 0; i < modelSpans.length; i++) {
    const mi = modelSpans[i]!;
    const miStarted = parseInt(String(mi['started_at'] ?? '0'), 10);
    const miNextStarted = i + 1 < modelSpans.length
      ? parseInt(String(modelSpans[i + 1]!['started_at'] ?? '0'), 10)
      : Number.POSITIVE_INFINITY;
    const miCustom = mi['custom_tags'] as Record<string, string> | undefined;
    const miModelName = miCustom?.['model_name'] ?? 'unknown';
    const miTimestamp = miStarted > 0 ? new Date(miStarted).toISOString() : '';
    const miInputTokens = parseInt(miCustom?.['input_tokens'] ?? '0', 10);
    const miOutputTokens = parseInt(miCustom?.['output_tokens'] ?? '0', 10);

    const miOutput = safeParseJson(mi['output']);
    const choices = (miOutput['choices'] as Array<Record<string, unknown>>) ?? [];
    const outputMsg = choices.length > 0
      ? (choices[0]!['message'] as Record<string, unknown>) ?? ({} as Record<string, unknown>)
      : ({} as Record<string, unknown>);
    const assistant = convertOpenAIOutputMessage(outputMsg, miTimestamp, miModelName, miInputTokens, miOutputTokens);
    if (assistant) results.push(assistant);

    // Pair tool spans in this turn's window to the assistant tool_calls.
    const callIdsInOrder = ((outputMsg['tool_calls'] as Array<Record<string, unknown>>) ?? [])
      .map(tc => tc['id'] as string)
      .filter(Boolean);
    const consumedTools = new Set<string>();
    for (const callId of callIdsInOrder) {
      const t = toolByOriginId.get(callId);
      if (!t) continue;
      const ts = parseInt(String(t['started_at'] ?? '0'), 10);
      if (ts < miStarted || ts >= miNextStarted) continue;
      consumedTools.add(callId);
      const tags = t['custom_tags'] as Record<string, string> | undefined;
      const toolName = tags?.['tool_name'] ?? (t['span_name'] as string) ?? 'unknown';
      const rawOutput = String(t['output'] ?? '');
      const isError = (t['status'] as string) === 'error' || (t['status_code'] as number) !== 0;
      // Attach to the assistant message we just appended, mirroring the merge
      // behavior of mergeToolResults so the tool result lives on the same message.
      if (assistant) {
        if (!assistant.toolResults) assistant.toolResults = [];
        assistant.toolResults.push({ toolName, isError, content: rawOutput });
      }
    }

    // Orphan tool spans in this window (no matching tool_call) — still emit so
    // the timeline is complete.
    for (const [originId, t] of toolByOriginId) {
      if (consumedTools.has(originId)) continue;
      const ts = parseInt(String(t['started_at'] ?? '0'), 10);
      if (ts < miStarted || ts >= miNextStarted) continue;
      const tags = t['custom_tags'] as Record<string, string> | undefined;
      const toolName = tags?.['tool_name'] ?? (t['span_name'] as string) ?? 'unknown';
      const rawOutput = String(t['output'] ?? '');
      const isError = (t['status'] as string) === 'error' || (t['status_code'] as number) !== 0;
      if (assistant) {
        if (!assistant.toolResults) assistant.toolResults = [];
        assistant.toolResults.push({ toolName, isError, content: rawOutput });
      }
    }

    // Detect new user messages introduced between Mi and Mi+1.
    if (i + 1 < modelSpans.length) {
      const nextInput = safeParseJson(modelSpans[i + 1]!['input']);
      const nextMessages = (nextInput['messages'] as Array<Record<string, unknown>>) ?? [];
      // Walk from the end; collect contiguous trailing user messages. Trae appends
      // new user input to the end of the rolling history, so trailing-user is the
      // canonical "new user message between rounds" position.
      const trailingUsers: Array<Record<string, unknown>> = [];
      for (let j = nextMessages.length - 1; j >= 0; j--) {
        if (nextMessages[j]!['role'] === 'user') {
          trailingUsers.unshift(nextMessages[j]!);
        } else {
          break;
        }
      }
      for (const u of trailingUsers) {
        const normalized = convertOpenAIMessage(u);
        if (normalized) results.push(normalized);
      }
    }
  }

  return results;
}

/** Convert an OpenAI-format message from input.messages to NormalizedMessage */
function convertOpenAIMessage(
  msg: Record<string, unknown>,
): NormalizedMessage | null {
  const role = msg['role'] as string;

  if (role === 'system') {
    return {
      role: 'system',
      timestamp: '',
      content: String(msg['content'] ?? ''),
    };
  }

  if (role === 'user') {
    // Trae user messages may use `parts` array instead of `content` string
    const parts = msg['parts'] as Array<Record<string, unknown>> | undefined;
    const content = msg['content'];
    let text = '';

    if (parts && Array.isArray(parts)) {
      text = parts
        .filter(p => p['type'] === 'text')
        .map(p => String(p['text'] ?? ''))
        .join('\n');
    } else if (typeof content === 'string') {
      text = content;
    } else if (Array.isArray(content)) {
      // OpenAI multi-part content
      text = (content as Array<Record<string, unknown>>)
        .filter(p => p['type'] === 'text')
        .map(p => String(p['text'] ?? ''))
        .join('\n');
    }

    return {
      role: 'user',
      timestamp: '',
      content: text,
    };
  }

  if (role === 'assistant') {
    const content = String(msg['content'] ?? '');
    const thinking = msg['reasoning_content'] as string | undefined;
    const toolCalls = msg['tool_calls'] as Array<Record<string, unknown>> | undefined;

    const normalized: NormalizedMessage = {
      role: 'assistant',
      timestamp: '',
      content,
    };

    if (thinking) normalized.thinking = thinking;

    if (toolCalls && toolCalls.length > 0) {
      normalized.toolCalls = toolCalls.map(tc => {
        const fn = tc['function'] as Record<string, unknown> | undefined;
        const name = String(fn?.['name'] ?? 'unknown');
        let input: Record<string, unknown> | undefined;
        try {
          const args = fn?.['arguments'];
          input = typeof args === 'string' ? JSON.parse(args) : (args as Record<string, unknown>);
        } catch { /* use raw string */ }
        return {
          name,
          input: summarizeToolInput(name, input),
          toolUseId: tc['id'] as string | undefined,
        };
      });
    }

    return normalized;
  }

  if (role === 'tool') {
    // Tool result message — attach to the preceding assistant message
    const content = String(msg['content'] ?? '');
    const toolCallId = msg['tool_call_id'] as string | undefined;
    return {
      role: 'user',
      timestamp: '',
      content: '',
      toolResults: [{
        toolName: toolCallId ?? 'unknown',
        isError: false,
        content,
      }],
    };
  }

  return null;
}

/** Convert the output.choices[0].message to a NormalizedMessage */
function convertOpenAIOutputMessage(
  msg: Record<string, unknown>,
  timestamp: string,
  modelName: string,
  inputTokens: number,
  outputTokens: number,
): NormalizedMessage | null {
  const role = msg['role'] as string;
  if (role !== 'assistant') return null;

  const content = String(msg['content'] ?? '');
  const thinking = msg['reasoning_content'] as string | undefined;
  const toolCalls = msg['tool_calls'] as Array<Record<string, unknown>> | undefined;

  const normalized: NormalizedMessage = {
    role: 'assistant',
    timestamp,
    content,
    model: modelName,
    usage: {
      inputTokens,
      outputTokens,
      cacheWriteTokens: 0,
      cacheReadTokens: 0,
    },
  };

  if (thinking) normalized.thinking = thinking;

  if (toolCalls && toolCalls.length > 0) {
    normalized.toolCalls = toolCalls.map(tc => {
      const fn = tc['function'] as Record<string, unknown> | undefined;
      const name = String(fn?.['name'] ?? 'unknown');
      let input: Record<string, unknown> | undefined;
      try {
        const args = fn?.['arguments'];
        input = typeof args === 'string' ? JSON.parse(args) : (args as Record<string, unknown>);
      } catch { /* use raw */ }
      return {
        name,
        input: summarizeToolInput(name, input),
        toolUseId: tc['id'] as string | undefined,
      };
    });
  }

  return normalized;
}

// ─── Helpers ───────────────────────────────────────────────────────

function summarizeToolInput(toolName: string, input: Record<string, unknown> | undefined): string {
  if (!input) return '';
  switch (toolName) {
    case 'Bash':
    case 'bash':
      return (input['command'] as string) ?? '';
    case 'Edit':
    case 'Write':
    case 'edit':
    case 'write': {
      const fp = (input['file_path'] as string) ?? (input['filePath'] as string) ?? (input['path'] as string) ?? '';
      return fp;
    }
    case 'Read':
    case 'read':
      return (input['file_path'] as string) ?? (input['filePath'] as string) ?? (input['path'] as string) ?? '';
    case 'Glob':
    case 'Grep':
    case 'glob':
    case 'grep':
      return `${input['pattern'] ?? ''} in ${input['path'] ?? '.'}`;
    case 'Agent':
      return `[Subagent: ${(input['subagent_type'] as string) ?? (input['subagentType'] as string) ?? 'general'}] ${(input['description'] as string) ?? ''}`;
    // Trae tool names
    case 'RunCommand':
    case 'run_command':
      return (input['command'] as string) ?? '';
    case 'SearchCodebase':
    case 'search_codebase':
    case 'search':
      return (input['query'] as string) ?? (input['keyword'] as string) ?? '';
    case 'LS':
    case 'list_dir':
      return (input['path'] as string) ?? '';
    case 'TodoWrite':
    case 'todo_write':
      return typeof input['todos'] === 'string' ? input['todos'] : JSON.stringify(input['todos'] ?? input);
    case 'WebSearch':
    case 'web_search':
      return (input['query'] as string) ?? '';
    default:
      return JSON.stringify(input);
  }
}
