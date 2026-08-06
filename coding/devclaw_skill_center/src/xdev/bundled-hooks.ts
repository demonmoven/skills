// Use cat (not cat /dev/stdin) to read stdin — /dev/stdin may be unavailable when
// the parent process (xdev) consumed it via readline before spawning claude.
// Use printf '%s' instead of echo to avoid expanding \n escape sequences in JSON.
// Trailing & backgrounds the trace command so the hook returns immediately.
// No async:true — sync hooks get reliable stdin; we background the slow part ourselves.
export const TRACE_SESSION_START_CMD =
  `INPUT=$(cat); SESSION_ID=$(printf '%s' "$INPUT" | jq -r '.session_id'); xdev trace session-start --session-id "$SESSION_ID" &`;

export const TRACE_FORWARD_CMD =
  `INPUT=$(cat); TRANSCRIPT=$(printf '%s' "$INPUT" | jq -r '.transcript_path'); xdev trace forward --file "$TRANSCRIPT" &`;

export const CODE_STAT_CMD =
  `TEA_APP_ID=1220 TEA_CHANNEL=cn TEA_APP_NAME_FOR_BITS=caribou timeout 8s ab-agent-collect-event || true`;

export interface ClaudeStyleHookEntry {
  type: 'command';
  command: string;
  async?: boolean;
  timeout?: number;
}

export interface ClaudeStyleHookGroup {
  matcher?: string;
  hooks: ClaudeStyleHookEntry[];
}

export type ClaudeStyleHookMap = Record<string, ClaudeStyleHookGroup[]>;

export interface CocoHookMatcher {
  event: string;
  tool?: string;
}

export interface CocoHook {
  type: 'command';
  command: string;
  timeout?: string;
  matchers: CocoHookMatcher[];
}

export function getCCBundledHookGroups(): ClaudeStyleHookMap {
  return {
    SessionStart: [
      {
        matcher: '',
        hooks: [{ type: 'command', command: TRACE_SESSION_START_CMD }],
      },
    ],
    // Separate groups with different matchers so mergeClaudeStyleHooks won't merge them.
    // Each group gets its own stdin pipe — same-group hooks share stdin, so npx would
    // consume it before the trace hook can read it if they were in the same group.
    Stop: [
      {
        matcher: '*',
        hooks: [{ type: 'command', command: CODE_STAT_CMD }],
      },
      {
        matcher: '',
        hooks: [{ type: 'command', command: TRACE_FORWARD_CMD }],
      },
    ],
    PostToolUse: [
      {
        matcher: 'Write|Edit|MultiEdit',
        hooks: [{ type: 'command', command: CODE_STAT_CMD }],
      },
    ],
  };
}

export function getCodexBundledHookGroups(): ClaudeStyleHookMap {
  return {
    SessionStart: [
      {
        matcher: 'startup|resume',
        hooks: [{ type: 'command', command: TRACE_SESSION_START_CMD, timeout: 30 }],
      },
    ],
    Stop: [
      {
        matcher: '*',
        hooks: [{ type: 'command', command: CODE_STAT_CMD }],
      },
      {
        hooks: [{ type: 'command', command: TRACE_FORWARD_CMD, timeout: 60 }],
      },
    ],
    PostToolUse: [
      {
        matcher: 'Write|Edit|MultiEdit',
        hooks: [{ type: 'command', command: CODE_STAT_CMD }],
      },
    ],
  };
}

export function getCocoBundledHooks(): CocoHook[] {
  return [
    {
      type: 'command',
      command: TRACE_SESSION_START_CMD,
      timeout: '30s',
      matchers: [{ event: 'session_start' }],
    },
    {
      type: 'command',
      command: TRACE_FORWARD_CMD,
      timeout: '60s',
      matchers: [{ event: 'stop' }],
    },
  ];
}
