import { useState } from 'react';
import type { NormalizedMessage } from '../types';

const PAGE_SIZE = 50;

const card: React.CSSProperties = {
  background: 'var(--surface)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  padding: 16,
};

interface SubagentConversation {
  agentId: string;
  agentType: string;
  description: string;
  messages: NormalizedMessage[];
}

interface Props {
  messages: NormalizedMessage[];
  subagentConversations?: Record<string, SubagentConversation>;
}

/**
 * Find the agentId for an Agent tool call by matching its toolUseId
 * to the corresponding tool result that carries the agentId.
 */
function findAgentIdForToolCall(
  toolUseId: string | undefined,
  toolResults: Array<{ toolName: string; agentId?: string }> | undefined,
): string | null {
  if (!toolUseId || !toolResults) return null;
  // toolResult.toolName is set to tool_use_id in the normalizer
  const result = toolResults.find(tr => tr.toolName === toolUseId);
  return result?.agentId ?? null;
}

export function ConversationTab({ messages, subagentConversations }: Props) {
  const [page, setPage] = useState(0);
  const totalPages = Math.max(1, Math.ceil(messages.length / PAGE_SIZE));
  const pageMessages = messages.slice(page * PAGE_SIZE, (page + 1) * PAGE_SIZE);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      {/* Pagination header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <div style={{ fontSize: 13, color: 'var(--text2)' }}>
          {messages.length} messages
        </div>
        {totalPages > 1 && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <button
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              disabled={page === 0}
              style={paginationBtn}
            >
              ← Prev
            </button>
            <span style={{ fontSize: 13, color: 'var(--text2)' }}>
              {page + 1} / {totalPages}
            </span>
            <button
              onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
              disabled={page >= totalPages - 1}
              style={paginationBtn}
            >
              Next →
            </button>
          </div>
        )}
      </div>

      {/* Messages timeline */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {pageMessages.map((msg, i) => (
          <MessageBubble key={page * PAGE_SIZE + i} msg={msg} subagentConversations={subagentConversations} />
        ))}
      </div>

      {/* Pagination footer */}
      {totalPages > 1 && (
        <div style={{ display: 'flex', justifyContent: 'center', gap: 8 }}>
          <button
            onClick={() => setPage((p) => Math.max(0, p - 1))}
            disabled={page === 0}
            style={paginationBtn}
          >
            ← Prev
          </button>
          <span style={{ fontSize: 13, color: 'var(--text2)', lineHeight: '32px' }}>
            {page + 1} / {totalPages}
          </span>
          <button
            onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
            disabled={page >= totalPages - 1}
            style={paginationBtn}
          >
            Next →
          </button>
        </div>
      )}
    </div>
  );
}

const paginationBtn: React.CSSProperties = {
  background: 'var(--surface)',
  border: '1px solid var(--border)',
  borderRadius: 6,
  padding: '4px 12px',
  color: 'var(--text)',
  cursor: 'pointer',
  fontSize: 13,
};

const CONTENT_COLLAPSE_THRESHOLD = 2000;
const THINKING_COLLAPSE_THRESHOLD = 1000;

/** Collapsible long text block: shows first N chars with an expand button */
function CollapsibleText({
  text,
  threshold,
  style,
}: {
  text: string;
  threshold: number;
  style?: React.CSSProperties;
}) {
  const [expanded, setExpanded] = useState(false);
  const isLong = text.length > threshold;

  return (
    <div style={style}>
      <div
        style={{
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
          lineHeight: 1.5,
          maxHeight: expanded ? 'none' : 400,
          overflow: expanded ? 'visible' : 'auto',
        }}
      >
        {isLong && !expanded ? text.slice(0, threshold) + '…' : text}
      </div>
      {isLong && (
        <button
          onClick={() => setExpanded(!expanded)}
          style={{
            marginTop: 6,
            background: 'var(--surface2)',
            border: '1px solid var(--border)',
            borderRadius: 4,
            padding: '3px 10px',
            color: 'var(--accent)',
            cursor: 'pointer',
            fontSize: 12,
          }}
        >
          {expanded
            ? '▲ Collapse'
            : `▼ Show all (${text.length.toLocaleString()} chars)`}
        </button>
      )}
    </div>
  );
}

function MessageBubble({ msg, subagentConversations }: {
  msg: NormalizedMessage;
  subagentConversations?: Record<string, SubagentConversation>;
}) {
  const [thinkingOpen, setThinkingOpen] = useState(false);
  const isUser = msg.role === 'user';
  const isAssistant = msg.role === 'assistant';

  const roleBadge: React.CSSProperties = {
    display: 'inline-block',
    fontSize: 11,
    fontWeight: 600,
    padding: '2px 8px',
    borderRadius: 4,
    background: isUser ? '#1f3a5f' : isAssistant ? '#2d1f5e' : '#2d3a2d',
    color: isUser ? '#58a6ff' : isAssistant ? '#bc8cff' : '#3fb950',
    marginBottom: 6,
  };

  const align: React.CSSProperties = isAssistant
    ? { alignItems: 'flex-end' }
    : { alignItems: 'flex-start' };

  const ts = msg.timestamp
    ? new Date(msg.timestamp).toLocaleTimeString()
    : '';

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        ...align,
      }}
    >
      <div
        style={{
          ...card,
          maxWidth: '85%',
          minWidth: 200,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
          <span style={roleBadge}>{msg.role}</span>
          {msg.model && (
            <span style={{ fontSize: 11, color: 'var(--text2)' }}>{msg.model}</span>
          )}
          {ts && (
            <span style={{ fontSize: 11, color: 'var(--text2)', marginLeft: 'auto' }}>
              {ts}
            </span>
          )}
        </div>

        {/* Content — collapsible if long */}
        {msg.content && (
          <CollapsibleText
            text={msg.content}
            threshold={CONTENT_COLLAPSE_THRESHOLD}
            style={{ fontSize: 13, color: 'var(--text)' }}
          />
        )}

        {/* Thinking — collapsible, full text available */}
        {msg.thinking && (
          <div style={{ marginTop: 8 }}>
            <button
              onClick={() => setThinkingOpen(!thinkingOpen)}
              style={{
                background: '#2d1f4e',
                border: '1px solid #4a3080',
                borderRadius: 4,
                padding: '3px 8px',
                color: '#bc8cff',
                cursor: 'pointer',
                fontSize: 12,
              }}
            >
              {thinkingOpen ? '▼' : '▶'} Thinking ({msg.thinking.length.toLocaleString()} chars)
            </button>
            {thinkingOpen && (
              <CollapsibleText
                text={msg.thinking}
                threshold={THINKING_COLLAPSE_THRESHOLD}
                style={{
                  marginTop: 6,
                  background: '#1a1028',
                  border: '1px solid #4a3080',
                  borderRadius: 6,
                  padding: 10,
                  fontSize: 12,
                  color: '#bc8cff',
                }}
              />
            )}
          </div>
        )}

        {/* Tool Calls — collapsible, with inline subagent conversations */}
        {msg.toolCalls && msg.toolCalls.length > 0 && (
          <div style={{ marginTop: 8, display: 'flex', flexDirection: 'column', gap: 4 }}>
            {msg.toolCalls.map((tc, i) => {
              let subConv: SubagentConversation | null = null;
              if (tc.name === 'Agent' && subagentConversations) {
                const agentId = findAgentIdForToolCall(tc.toolUseId, msg.toolResults);
                if (agentId && subagentConversations[agentId]) {
                  subConv = subagentConversations[agentId];
                }
              }
              return (
                <div key={i}>
                  <ToolCallBlock name={tc.name} input={tc.input} />
                  {subConv && <SubagentConversationBlock conv={subConv} />}
                </div>
              );
            })}
          </div>
        )}

        {/* Tool Results — collapsible */}
        {msg.toolResults && msg.toolResults.length > 0 && (
          <div style={{ marginTop: 8, display: 'flex', flexDirection: 'column', gap: 4 }}>
            {msg.toolResults.map((tr, i) => (
              <ToolResultBlock key={i} toolName={tr.toolName} isError={tr.isError} content={tr.content} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

const TOOL_SUMMARY_LEN = 120;

function ToolCallBlock({ name, input }: { name: string; input: string }) {
  const [expanded, setExpanded] = useState(false);
  const isLong = input.length > TOOL_SUMMARY_LEN;
  const summary = isLong ? input.slice(0, TOOL_SUMMARY_LEN) + '…' : input;

  return (
    <div
      style={{
        background: '#161b22',
        border: '1px solid var(--border)',
        borderRadius: 4,
        padding: '6px 10px',
        fontSize: 12,
        fontFamily: 'monospace',
      }}
    >
      <div
        style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: isLong ? 'pointer' : 'default' }}
        onClick={() => isLong && setExpanded(!expanded)}
      >
        <span style={{ color: name === 'Agent' ? '#d29922' : '#58a6ff', fontWeight: 600, flexShrink: 0 }}>
          {name === 'Agent' ? '⑂ Agent' : name}
        </span>
        {!expanded && (
          <span style={{ color: 'var(--text2)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {summary}
          </span>
        )}
        {isLong && (
          <span style={{ color: 'var(--accent)', flexShrink: 0, marginLeft: 'auto', fontSize: 11 }}>
            {expanded ? '▲' : '▼'}
          </span>
        )}
      </div>
      {expanded && (
        <div
          style={{
            marginTop: 6,
            padding: '6px 8px',
            background: '#0d1117',
            borderRadius: 4,
            color: 'var(--text)',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
            lineHeight: 1.5,
            maxHeight: 500,
            overflow: 'auto',
          }}
        >
          {input}
        </div>
      )}
    </div>
  );
}

function ToolResultBlock({ toolName, isError, content }: { toolName: string; isError: boolean; content: string }) {
  const [expanded, setExpanded] = useState(false);
  const hasContent = content.length > 0;
  const statusColor = isError ? '#f85149' : '#3fb950';
  const statusBg = isError ? '#3d1418' : '#1a2e1a';

  return (
    <div
      style={{
        background: statusBg,
        border: `1px solid ${isError ? '#f8514933' : '#3fb95033'}`,
        borderRadius: 4,
        padding: '6px 10px',
        fontSize: 12,
      }}
    >
      <div
        style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: hasContent ? 'pointer' : 'default' }}
        onClick={() => hasContent && setExpanded(!expanded)}
      >
        <span style={{ color: statusColor, fontWeight: 600 }}>
          {isError ? '✗' : '✓'} {toolName}
        </span>
        {hasContent && !expanded && (
          <span style={{ color: 'var(--text2)', fontSize: 11, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {content.slice(0, 80)}{content.length > 80 ? '…' : ''}
          </span>
        )}
        {hasContent && (
          <span style={{ color: statusColor, flexShrink: 0, marginLeft: 'auto', fontSize: 11 }}>
            {expanded ? '▲' : `▼ ${content.length.toLocaleString()} chars`}
          </span>
        )}
      </div>
      {expanded && (
        <div
          style={{
            marginTop: 6,
            padding: '6px 8px',
            background: '#0d1117',
            borderRadius: 4,
            color: 'var(--text)',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
            lineHeight: 1.5,
            maxHeight: 500,
            overflow: 'auto',
            fontFamily: 'monospace',
            fontSize: 11,
          }}
        >
          {content}
        </div>
      )}
    </div>
  );
}

// ─── Subagent Conversation Block ──────────────────────────────────

function SubagentConversationBlock({ conv }: { conv: SubagentConversation }) {
  const [expanded, setExpanded] = useState(false);
  const msgCount = conv.messages.length;

  return (
    <div
      style={{
        marginTop: 4,
        border: '1px solid #d2992233',
        borderRadius: 6,
        overflow: 'hidden',
      }}
    >
      <div
        onClick={() => setExpanded(!expanded)}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          padding: '8px 12px',
          background: '#1a1500',
          cursor: 'pointer',
          fontSize: 12,
        }}
      >
        <span style={{ color: '#d29922', fontWeight: 600 }}>
          {expanded ? '▼' : '▶'} Subagent: {conv.agentType}
        </span>
        <span style={{ color: 'var(--text2)', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {conv.description}
        </span>
        <span style={{ color: 'var(--text2)', flexShrink: 0 }}>
          {msgCount} messages
        </span>
      </div>
      {expanded && (
        <div
          style={{
            padding: '8px 12px',
            background: '#12100a',
            borderTop: '1px solid #d2992222',
            display: 'flex',
            flexDirection: 'column',
            gap: 8,
          }}
        >
          {conv.messages.map((msg, i) => (
            <SubagentMessageBubble key={i} msg={msg} />
          ))}
        </div>
      )}
    </div>
  );
}

function SubagentMessageBubble({ msg }: { msg: NormalizedMessage }) {
  const isUser = msg.role === 'user';
  const isAssistant = msg.role === 'assistant';

  const roleBadge: React.CSSProperties = {
    display: 'inline-block',
    fontSize: 10,
    fontWeight: 600,
    padding: '1px 6px',
    borderRadius: 3,
    background: isUser ? '#1f3a5f' : isAssistant ? '#2d1f5e' : '#2d3a2d',
    color: isUser ? '#58a6ff' : isAssistant ? '#bc8cff' : '#3fb950',
  };

  return (
    <div
      style={{
        background: '#161b22',
        border: '1px solid var(--border)',
        borderRadius: 6,
        padding: '8px 10px',
        fontSize: 12,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
        <span style={roleBadge}>{msg.role}</span>
        {msg.model && <span style={{ fontSize: 10, color: 'var(--text2)' }}>{msg.model}</span>}
      </div>

      {msg.content && (
        <CollapsibleText text={msg.content} threshold={800} style={{ color: 'var(--text)', lineHeight: 1.4 }} />
      )}

      {msg.toolCalls && msg.toolCalls.length > 0 && (
        <div style={{ marginTop: 4, display: 'flex', flexDirection: 'column', gap: 3 }}>
          {msg.toolCalls.map((tc, i) => (
            <ToolCallBlock key={i} name={tc.name} input={tc.input} />
          ))}
        </div>
      )}

      {msg.toolResults && msg.toolResults.length > 0 && (
        <div style={{ marginTop: 4, display: 'flex', flexDirection: 'column', gap: 3 }}>
          {msg.toolResults.map((tr, i) => (
            <ToolResultBlock key={i} toolName={tr.toolName} isError={tr.isError} content={tr.content} />
          ))}
        </div>
      )}
    </div>
  );
}
