/** A single span from a Fornax trace (as returned by `fornax-cli trace get`) */
export interface FornaxSpan {
  trace_id: string;
  span_id: string;
  parent_id?: string;
  span_name: string;
  span_type: string;
  /** Top-level type: "model" for LLM calls, "unknwon" (sic) for others */
  type: string;
  duration: string;            // milliseconds as string
  started_at: string;          // epoch milliseconds as string
  status: string;
  status_code: number;
  input: string;               // JSON-encoded string
  output: string;              // JSON-encoded string
  service_name: string;
  logid: string;
  custom_tags: Record<string, string>;
  system_tags: Record<string, string>;
}

/** Complete trace data from Fornax (array of spans sharing the same trace_id) */
export interface FornaxTraceData {
  trace_id: string;
  spans: FornaxSpan[];
}

/** Options for fetching a trace via fornax-cli */
export interface FornaxFetchOptions {
  ak?: string;
  sk?: string;
  region?: string;
  endpoint?: string;
  /** CLI timeout, e.g. "30s", "1m". Default: "60s" */
  timeout?: string;
  /** Time window start, ISO 8601. Example: "2026-04-09T11:00:00+08:00" */
  since?: string;
  /** Time window end, ISO 8601. Example: "2026-04-09T12:00:00+08:00" */
  until?: string;
}
