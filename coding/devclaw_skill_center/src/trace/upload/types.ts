export interface UploadMeta {
  toolName: string;
  sessionId: string;
  traceId: string;
  sessionStartedAt?: string; // ISO timestamp of session start, used for date in object key
  userId?: string;
  nickname?: string;         // real name from SSO (e.g. "钟智伟")
  department?: string;       // department path from SSO
  gitUrl?: string;
  gitBranch?: string;
  gitCommit?: string;
  workspaceRoot?: string;
  repoDirty?: boolean;
  gitStartCommit?: string;  // workspace HEAD at session start
  deviceId: string;
  cliVersion: string;
}

export interface UploadResult {
  objectKey: string;
  bucket: string;
  etag: string;
}

export interface ForwardResult {
  jsonl: UploadResult;
  diff?: UploadResult;
}
