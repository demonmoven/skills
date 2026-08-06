export type ToolName = 'claude-code' | 'opencode';

export interface AdapterMeta {
  toolName: ToolName;
  sessionId: string;
  toolVersion?: string;
  startedAt?: string;
}

export interface Adapter {
  name: string;
  /** Check if the given file path belongs to this tool */
  detect(filePath: string): boolean;
  /** Extract session-level metadata from the JSONL file */
  extractMeta(filePath: string): Promise<AdapterMeta>;
}
