import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { fetchJson, postJson } from '../api/client';
import type { SessionListItem, SessionListResponse, SessionAnalysis } from '../types';

const PAGE_SIZE = 20;

const TOOL_COLORS: Record<string, string> = {
  'claude-code': '#58a6ff',
  opencode: '#3fb950',
  trae: '#a371f7',
};

/* ------------------------------------------------------------------ */
/*  Styles                                                             */
/* ------------------------------------------------------------------ */

const s = {
  page: {
    maxWidth: 960,
    margin: '0 auto',
    padding: '32px 24px',
  } satisfies React.CSSProperties,

  title: {
    fontSize: 28,
    fontWeight: 700,
    marginBottom: 24,
    color: 'var(--text)',
  } satisfies React.CSSProperties,

  filterBar: {
    display: 'flex',
    gap: 12,
    flexWrap: 'wrap' as const,
    marginBottom: 24,
  } satisfies React.CSSProperties,

  select: {
    padding: '8px 12px',
    background: 'var(--surface)',
    color: 'var(--text)',
    border: '1px solid var(--border)',
    borderRadius: 6,
    fontSize: 14,
    outline: 'none',
    minWidth: 150,
  } satisfies React.CSSProperties,

  input: {
    padding: '8px 12px',
    background: 'var(--surface)',
    color: 'var(--text)',
    border: '1px solid var(--border)',
    borderRadius: 6,
    fontSize: 14,
    outline: 'none',
    minWidth: 140,
  } satisfies React.CSSProperties,

  card: {
    display: 'flex',
    alignItems: 'center',
    gap: 16,
    padding: '16px 20px',
    background: 'var(--surface)',
    border: '1px solid var(--border)',
    borderRadius: 8,
    marginBottom: 10,
    cursor: 'pointer',
    transition: 'border-color 0.15s, background 0.15s',
  } satisfies React.CSSProperties,

  cardHover: {
    borderColor: 'var(--accent)',
    background: 'var(--surface2)',
  } satisfies React.CSSProperties,

  sessionId: {
    fontFamily: 'monospace',
    fontSize: 14,
    color: 'var(--accent)',
    minWidth: 140,
  } satisfies React.CSSProperties,

  badge: (color: string) =>
    ({
      display: 'inline-block',
      padding: '2px 10px',
      borderRadius: 12,
      fontSize: 12,
      fontWeight: 600,
      color: '#fff',
      background: color,
      whiteSpace: 'nowrap',
    }) satisfies React.CSSProperties,

  meta: {
    fontSize: 13,
    color: 'var(--text2)',
    marginLeft: 'auto',
    textAlign: 'right' as const,
    whiteSpace: 'nowrap' as const,
  } satisfies React.CSSProperties,

  pager: {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    gap: 16,
    marginTop: 24,
  } satisfies React.CSSProperties,

  pageBtn: (disabled: boolean) =>
    ({
      padding: '8px 18px',
      background: disabled ? 'var(--surface2)' : 'var(--surface)',
      color: disabled ? 'var(--text2)' : 'var(--text)',
      border: '1px solid var(--border)',
      borderRadius: 6,
      fontSize: 14,
      cursor: disabled ? 'not-allowed' : 'pointer',
      opacity: disabled ? 0.5 : 1,
    }) satisfies React.CSSProperties,

  center: {
    textAlign: 'center' as const,
    padding: '64px 0',
    color: 'var(--text2)',
    fontSize: 15,
  } satisfies React.CSSProperties,

  error: {
    textAlign: 'center' as const,
    padding: '64px 0',
    color: 'var(--red)',
    fontSize: 15,
  } satisfies React.CSSProperties,
} as const;

/* ------------------------------------------------------------------ */
/*  Component                                                          */
/* ------------------------------------------------------------------ */

export function SessionList() {
  const navigate = useNavigate();

  /* filter state */
  const [toolName, setToolName] = useState('');
  const [userId, setUserId] = useState('');
  const [sessionIdFilter, setSessionIdFilter] = useState('');
  const [dateFrom, setDateFrom] = useState('');
  const [dateTo, setDateTo] = useState('');
  const [department, setDepartment] = useState('');

  /* data state */
  const [page, setPage] = useState(1);
  const [sessions, setSessions] = useState<SessionListItem[]>([]);
  const [total, setTotal] = useState(0);
  const [departments, setDepartments] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  /* hover tracking */
  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null);

  /* Trae import state */
  const [showImport, setShowImport] = useState(false);
  const [importThreadId, setImportThreadId] = useState('');
  const [importAk, setImportAk] = useState('');
  const [importSk, setImportSk] = useState('');
  const [importRegion, setImportRegion] = useState('');
  const [importRepoUrl, setImportRepoUrl] = useState('');
  const [importCommitSha, setImportCommitSha] = useState('');
  const [importSince, setImportSince] = useState('');
  const [importUntil, setImportUntil] = useState('');
  const [importing, setImporting] = useState(false);
  const [importError, setImportError] = useState('');

  interface TraeImportResult {
    threadId: string;
    objectKey: string;
    traceCount: number;
    analysis?: SessionAnalysis;
  }

  /**
   * Normalize a datetime string to ISO 8601 with timezone.
   * Accepts: "2026/4/9 11:32:24", "2026-04-09 11:32:24", "2026-04-09T11:32:24+08:00", etc.
   * Returns ISO 8601 string with +08:00 timezone (assumes China time if no timezone given).
   */
  const normalizeTime = (input: string): string => {
    const s = input.trim();
    if (!s) return '';
    // Already ISO 8601 with timezone
    if (/T.*[+-]\d{2}:\d{2}$/.test(s)) return s;
    // Replace / with - for date part
    let normalized = s.replace(/\//g, '-');
    // Pad month/day: "2026-4-9" -> "2026-04-09"
    normalized = normalized.replace(/^(\d{4})-(\d{1,2})-(\d{1,2})/, (_m, y, mo, d) =>
      `${y}-${String(mo).padStart(2, '0')}-${String(d).padStart(2, '0')}`
    );
    // Replace space with T if not already
    if (!normalized.includes('T')) {
      normalized = normalized.replace(' ', 'T');
    }
    // Append timezone if missing
    if (!/[+-]\d{2}:\d{2}$/.test(normalized) && !normalized.endsWith('Z')) {
      normalized += '+08:00';
    }
    return normalized;
  };

  const handleImport = async () => {
    if (!importThreadId.trim()) return;
    setImporting(true);
    setImportError('');
    try {
      const body: Record<string, unknown> = { threadId: importThreadId.trim() };
      if (importRepoUrl.trim()) body['repoUrl'] = importRepoUrl.trim();
      if (importCommitSha.trim()) body['commitSha'] = importCommitSha.trim();
      const fornaxConfig: Record<string, string> = {};
      if (importAk.trim()) fornaxConfig['ak'] = importAk.trim();
      if (importSk.trim()) fornaxConfig['sk'] = importSk.trim();
      if (importRegion.trim()) fornaxConfig['region'] = importRegion.trim();
      if (importSince.trim()) fornaxConfig['since'] = normalizeTime(importSince);
      if (importUntil.trim()) fornaxConfig['until'] = normalizeTime(importUntil);
      if (Object.keys(fornaxConfig).length > 0) body['fornaxConfig'] = fornaxConfig;

      const result = await postJson<TraeImportResult>('/trae/import', body);

      navigate(
        `/sessions/${result.threadId}?objectKey=${encodeURIComponent(result.objectKey)}`,
      );
    } catch (err: unknown) {
      setImportError(err instanceof Error ? err.message : String(err));
    } finally {
      setImporting(false);
    }
  };

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  /* ---- fetch ---- */
  const loadSessions = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const params = new URLSearchParams();
      params.set('page', String(page));
      params.set('pageSize', String(PAGE_SIZE));
      if (toolName) params.set('toolName', toolName);
      if (userId.trim()) params.set('userId', userId.trim());
      if (sessionIdFilter.trim()) params.set('sessionId', sessionIdFilter.trim());
      if (dateFrom) params.set('dateFrom', dateFrom);
      if (dateTo) params.set('dateTo', dateTo);
      if (department) params.set('department', department);

      const data = await fetchJson<SessionListResponse>(
        `/sessions?${params.toString()}`,
      );
      setSessions(data.sessions);
      setTotal(data.total);
      if (data.departments) setDepartments(data.departments);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [page, toolName, userId, sessionIdFilter, dateFrom, dateTo, department]);

  useEffect(() => {
    void loadSessions();
  }, [loadSessions]);

  /* reset to page 1 when filters change */
  const handleToolChange = (v: string) => {
    setToolName(v);
    setPage(1);
  };
  const handleUserChange = (v: string) => {
    setUserId(v);
    setPage(1);
  };
  const handleSessionIdChange = (v: string) => {
    setSessionIdFilter(v);
    setPage(1);
  };
  const handleDateFromChange = (v: string) => {
    setDateFrom(v);
    setPage(1);
  };
  const handleDateToChange = (v: string) => {
    setDateTo(v);
    setPage(1);
  };
  const handleDepartmentChange = (v: string) => {
    setDepartment(v);
    setPage(1);
  };

  /* ---- navigate ---- */
  const goToSession = (item: SessionListItem) => {
    navigate(
      `/sessions/${item.sessionId}?objectKey=${encodeURIComponent(item.objectKey)}`,
    );
  };

  /* ---- render ---- */
  return (
    <div style={s.page}>
      <h1 style={s.title}>xtrace Dashboard</h1>

      {/* Filter bar */}
      <div style={s.filterBar}>
        <select
          style={s.select}
          value={toolName}
          onChange={(e) => handleToolChange(e.target.value)}
        >
          <option value="">All Tools</option>
          <option value="claude-code">claude-code</option>
          <option value="opencode">opencode</option>
          <option value="trae">trae</option>
        </select>

        <select
          style={{ ...s.select, minWidth: 200 }}
          value={department}
          onChange={(e) => handleDepartmentChange(e.target.value)}
        >
          <option value="">All Departments</option>
          {departments.map((dept) => (
            <option key={dept} value={dept}>
              {dept}
            </option>
          ))}
        </select>

        <input
          style={s.input}
          type="text"
          placeholder="User ID"
          value={userId}
          onChange={(e) => handleUserChange(e.target.value)}
        />

        <input
          style={{ ...s.input, minWidth: 260 }}
          type="text"
          placeholder="Session ID / Trace ID"
          value={sessionIdFilter}
          onChange={(e) => handleSessionIdChange(e.target.value)}
        />

        <input
          style={s.input}
          type="date"
          value={dateFrom}
          onChange={(e) => handleDateFromChange(e.target.value)}
        />

        <input
          style={s.input}
          type="date"
          value={dateTo}
          onChange={(e) => handleDateToChange(e.target.value)}
        />
      </div>

      {/* Import Trae Trace */}
      <div style={{ marginBottom: 20 }}>
        <button
          style={{
            padding: '8px 16px',
            background: '#a371f7',
            color: '#fff',
            border: 'none',
            borderRadius: 6,
            fontSize: 14,
            fontWeight: 600,
            cursor: 'pointer',
          }}
          onClick={() => setShowImport(!showImport)}
        >
          {showImport ? 'Cancel' : 'Import Trae Thread'}
        </button>
      </div>

      {showImport && (
        <div
          style={{
            padding: 20,
            background: 'var(--surface)',
            border: '1px solid var(--border)',
            borderRadius: 8,
            marginBottom: 24,
          }}
        >
          <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 12, color: 'var(--text)' }}>
            Import from Fornax
          </div>
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', marginBottom: 12 }}>
            <input
              style={{ ...s.input, flex: 1, minWidth: 240 }}
              type="text"
              placeholder="Thread ID (required)"
              value={importThreadId}
              onChange={(e) => setImportThreadId(e.target.value)}
            />
            <input
              style={{ ...s.input, minWidth: 200 }}
              type="text"
              placeholder="Repo URL (optional)"
              value={importRepoUrl}
              onChange={(e) => setImportRepoUrl(e.target.value)}
            />
            <input
              style={{ ...s.input, minWidth: 120 }}
              type="text"
              placeholder="Commit SHA (optional)"
              value={importCommitSha}
              onChange={(e) => setImportCommitSha(e.target.value)}
            />
            <input
              style={{ ...s.input, minWidth: 120 }}
              type="text"
              placeholder="AK (optional)"
              value={importAk}
              onChange={(e) => setImportAk(e.target.value)}
            />
            <input
              style={{ ...s.input, minWidth: 120 }}
              type="password"
              placeholder="SK (optional)"
              value={importSk}
              onChange={(e) => setImportSk(e.target.value)}
            />
            <input
              style={{ ...s.input, minWidth: 80 }}
              type="text"
              placeholder="Region"
              value={importRegion}
              onChange={(e) => setImportRegion(e.target.value)}
            />
          </div>
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', marginBottom: 12 }}>
            <input
              style={{ ...s.input, minWidth: 200 }}
              type="text"
              placeholder="Since (required, e.g. 2026-04-09T11:00:00+08:00)"
              value={importSince}
              onChange={(e) => setImportSince(e.target.value)}
            />
            <input
              style={{ ...s.input, minWidth: 200 }}
              type="text"
              placeholder="Until (optional, auto-detects if empty)"
              value={importUntil}
              onChange={(e) => setImportUntil(e.target.value)}
            />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <button
              style={{
                padding: '8px 20px',
                background: importing ? 'var(--surface2)' : '#a371f7',
                color: '#fff',
                border: 'none',
                borderRadius: 6,
                fontSize: 14,
                fontWeight: 600,
                cursor: importing ? 'not-allowed' : 'pointer',
                opacity: importing ? 0.6 : 1,
              }}
              disabled={importing || !importThreadId.trim() || !importSince.trim()}
              onClick={handleImport}
            >
              {importing ? 'Importing...' : 'Import'}
            </button>
            {importError && (
              <span style={{ color: 'var(--red)', fontSize: 13 }}>{importError}</span>
            )}
          </div>
          <div style={{ fontSize: 12, color: 'var(--text2)', marginTop: 8 }}>
            Thread ID is the conversation_id from Trae. Since is required (session start time). Until is optional — auto-slides in 1h windows until all traces are found.
          </div>
        </div>
      )}

      {/* Loading */}
      {loading && <div style={s.center}>Loading...</div>}

      {/* Error */}
      {!loading && error && <div style={s.error}>{error}</div>}

      {/* Empty */}
      {!loading && !error && sessions.length === 0 && (
        <div style={s.center}>No sessions found</div>
      )}

      {/* Session cards */}
      {!loading &&
        !error &&
        sessions.map((item, idx) => (
          <div
            key={item.sessionId}
            style={{
              ...s.card,
              ...(hoveredIdx === idx ? s.cardHover : {}),
            }}
            onClick={() => goToSession(item)}
            onMouseEnter={() => setHoveredIdx(idx)}
            onMouseLeave={() => setHoveredIdx(null)}
          >
            <span style={s.sessionId}>
              {item.sessionId.slice(0, 16)}
            </span>

            <span
              style={s.badge(TOOL_COLORS[item.toolName] ?? 'var(--text2)')}
            >
              {item.toolName}
            </span>

            <span style={{ fontSize: 13, color: 'var(--text2)' }}>
              {item.nickname ? `${item.nickname} (${item.userId})` : item.userId}
            </span>

            {item.department && (
              <span style={{ fontSize: 12, color: 'var(--text2)', opacity: 0.7 }}>
                {item.department}
              </span>
            )}

            <span style={s.meta}>{item.date}</span>
          </div>
        ))}

      {/* Pagination */}
      {!loading && !error && total > 0 && (
        <div style={s.pager}>
          <button
            style={s.pageBtn(page <= 1)}
            disabled={page <= 1}
            onClick={() => setPage((p) => Math.max(1, p - 1))}
          >
            Previous
          </button>

          <span style={{ fontSize: 14, color: 'var(--text2)' }}>
            Page {page} / {totalPages}
          </span>

          <button
            style={s.pageBtn(page >= totalPages)}
            disabled={page >= totalPages}
            onClick={() => setPage((p) => p + 1)}
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
}
