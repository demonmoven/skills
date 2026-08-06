import { useState, useEffect, useCallback } from 'react';
import { useParams, useSearchParams, Link } from 'react-router-dom';
import { fetchJson, postJson } from '../api/client';
import type { SessionAnalysis, EvalResult } from '../types';
import { SummaryTab } from '../components/SummaryTab';
import { ConversationTab } from '../components/ConversationTab';
import { CostTab } from '../components/CostTab';
import { TrajectoryTab } from '../components/TrajectoryTab';

type TabKey = 'summary' | 'conversation' | 'cost' | 'trajectory';

const TABS: { key: TabKey; label: string }[] = [
  { key: 'summary', label: 'Summary' },
  { key: 'conversation', label: 'Conversation' },
  { key: 'cost', label: 'Cost Analysis' },
  { key: 'trajectory', label: 'Trajectory' },
];

const TOOL_COLORS: Record<string, string> = {
  'claude-code': '#1f3a5f',
  opencode: '#1a3a2a',
  trae: '#2d1f4e',
};

const TOOL_TEXT_COLORS: Record<string, string> = {
  'claude-code': '#58a6ff',
  opencode: '#3fb950',
  trae: '#a371f7',
};

interface TraceSlice {
  traceId: string;
  index: number;
  traceCount: number;
  analysis: SessionAnalysis;
}

/** User info attached to the analysis response by the server. */
interface UserInfo {
  userId?: string;
  nickname?: string;
  department?: string;
}

export function SessionDetail() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const [searchParams] = useSearchParams();
  const objectKey = searchParams.get('objectKey') ?? '';

  // Multi-trace state: populated from API response traceSlices
  const [traceAnalyses, setTraceAnalyses] = useState<TraceSlice[]>([]);
  const [activeTraceIdx, setActiveTraceIdx] = useState(-1);
  const isMultiTrace = traceAnalyses.length > 1;

  const [tab, setTab] = useState<TabKey>('summary');
  const [analysis, setAnalysis] = useState<SessionAnalysis | null>(null);
  const [userInfo, setUserInfo] = useState<UserInfo>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [evaluating, setEvaluating] = useState(false);
  const [evalError, setEvalError] = useState<string | null>(null);
  const [selectedCli, setSelectedCli] = useState<'claude-code' | 'opencode'>('claude-code');

  const runEvaluate = useCallback(async () => {
    if (!sessionId || !objectKey) return;
    try {
      setEvaluating(true);
      setEvalError(null);

      const evalBody: Record<string, unknown> = {
        objectKey,
        cli: selectedCli,
      };
      if (isMultiTrace) {
        evalBody['traceContext'] = `This is a Trae thread with ${traceAnalyses.length} traces. The file contains all traces sequentially. Focus on the overall session quality.`;
      }

      const result = await postJson<EvalResult>(
        `/sessions/${sessionId}/evaluate`,
        evalBody,
      );
      setAnalysis((prev) => (prev ? { ...prev, evaluation: result } : prev));
    } catch (err) {
      setEvalError(err instanceof Error ? err.message : 'Evaluation failed');
    } finally {
      setEvaluating(false);
    }
  }, [sessionId, objectKey, selectedCli, isMultiTrace, traceAnalyses.length]);

  const loadAnalysis = useCallback(
    async () => {
      if (!sessionId) return;
      try {
        setLoading(true);
        setError(null);

        const path = `/sessions/${sessionId}/analysis?objectKey=${encodeURIComponent(objectKey)}`;
        const data = await fetchJson<SessionAnalysis & { traceSlices?: TraceSlice[] } & UserInfo>(path);
        setAnalysis(data);
        setUserInfo({ userId: data.userId, nickname: data.nickname, department: data.department });

        // Use traceSlices from API response (server splits by boundary markers)
        if (data.traceSlices && data.traceSlices.length > 1) {
          setTraceAnalyses(data.traceSlices);
          setActiveTraceIdx(data.traceSlices.length - 1);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load analysis');
      } finally {
        setLoading(false);
      }
    },
    [sessionId, objectKey],
  );

  useEffect(() => {
    loadAnalysis();
  }, [loadAnalysis]);

  // The currently displayed analysis: either a per-trace slice or the full analysis.
  // Evaluation is always stored on the top-level analysis; propagate it to slices.
  const baseDisplayed = isMultiTrace && activeTraceIdx >= 0 && traceAnalyses[activeTraceIdx]
    ? traceAnalyses[activeTraceIdx]!.analysis
    : analysis;
  const displayedAnalysis = baseDisplayed && analysis?.evaluation && !baseDisplayed.evaluation
    ? { ...baseDisplayed, evaluation: analysis.evaluation }
    : baseDisplayed;

  // Loading state
  if (loading) {
    return (
      <div style={container}>
        <div style={{ textAlign: 'center', padding: 80, color: 'var(--text2)' }}>
          <div style={spinner} />
          <div style={{ marginTop: 16 }}>Analyzing session…</div>
        </div>
      </div>
    );
  }

  // Error state
  if (error) {
    return (
      <div style={container}>
        <Link to="/" style={backLink}>← Back</Link>
        <div style={{ textAlign: 'center', padding: 60, background: 'var(--surface)', border: '1px solid var(--border)', borderRadius: 8, marginTop: 16 }}>
          <div style={{ fontSize: 40, marginBottom: 12 }}>⚠️</div>
          <div style={{ color: '#f85149', fontWeight: 600, marginBottom: 8 }}>{error}</div>
          <button onClick={() => loadAnalysis()} style={primaryBtn}>Retry</button>
        </div>
      </div>
    );
  }

  if (!displayedAnalysis) {
    return (
      <div style={container}>
        <Link to="/" style={backLink}>← Back</Link>
        <div style={{ textAlign: 'center', padding: 60, color: 'var(--text2)', marginTop: 16 }}>
          No analysis data available
        </div>
      </div>
    );
  }

  const toolColor = TOOL_COLORS[displayedAnalysis.toolName] ?? '#1f1f1f';
  const toolTextColor = TOOL_TEXT_COLORS[displayedAnalysis.toolName] ?? 'var(--text2)';

  return (
    <div style={container}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
        <Link to="/" style={backLink}>← Back</Link>
        <h1 style={{ fontSize: 18, fontWeight: 600, color: 'var(--text)', margin: 0 }}>
          {sessionId}
        </h1>
        <span style={{ ...toolBadge, background: toolColor, color: toolTextColor }}>
          {displayedAnalysis.toolName}
        </span>
        {userInfo.userId && (
          <span style={{ fontSize: 13, color: 'var(--text2)' }}>
            {userInfo.nickname ? `${userInfo.nickname} (${userInfo.userId})` : userInfo.userId}
          </span>
        )}
        {userInfo.department && (
          <span style={{ fontSize: 12, color: 'var(--text2)', opacity: 0.7 }}>
            {userInfo.department}
          </span>
        )}
        <span style={{ fontSize: 13, color: 'var(--text2)' }}>
          {displayedAnalysis.durationStr}
        </span>
        <div style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 8 }}>
          <select
            value={selectedCli}
            onChange={(e) => setSelectedCli(e.target.value as 'claude-code' | 'opencode')}
            disabled={evaluating}
            style={selectStyle}
          >
            <option value="claude-code">claude-code</option>
            <option value="opencode">opencode</option>
          </select>
          <button
            onClick={runEvaluate}
            disabled={evaluating}
            style={{ ...evalBtn, opacity: evaluating ? 0.6 : 1, cursor: evaluating ? 'not-allowed' : 'pointer' }}
          >
            {evaluating ? (<><span style={miniSpinner} /> Evaluating…</>) : displayedAnalysis?.evaluation ? 'Re-evaluate' : 'Evaluate'}
          </button>
          {evalError && (
            <span style={{ fontSize: 12, color: '#f85149', maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {evalError}
            </span>
          )}
        </div>
      </div>

      {/* Trace selector (multi-trace Trae threads) */}
      {isMultiTrace && (
        <div style={{ display: 'flex', gap: 0, borderBottom: '2px solid var(--border)', marginTop: 16, overflowX: 'auto' }}>
          {traceAnalyses.map((_t, idx) => (
            <button
              key={idx}
              onClick={() => setActiveTraceIdx(idx)}
              style={{
                padding: '8px 16px',
                fontSize: 13,
                fontWeight: activeTraceIdx === idx ? 600 : 400,
                color: activeTraceIdx === idx ? '#a371f7' : 'var(--text2)',
                background: activeTraceIdx === idx ? 'rgba(163,113,247,0.1)' : 'transparent',
                border: 'none',
                borderBottom: activeTraceIdx === idx ? '2px solid #a371f7' : '2px solid transparent',
                cursor: 'pointer',
                whiteSpace: 'nowrap',
                transition: 'color 0.15s, border-color 0.15s, background 0.15s',
              }}
            >
              Trace {idx + 1}{idx === traceAnalyses.length - 1 ? ' (latest)' : ''}
            </button>
          ))}
        </div>
      )}

      {/* Analysis tab bar */}
      <div style={{ display: 'flex', gap: 0, borderBottom: '1px solid var(--border)', marginTop: isMultiTrace ? 8 : 16 }}>
        {TABS.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            style={{
              padding: '10px 20px',
              fontSize: 14,
              fontWeight: tab === t.key ? 600 : 400,
              color: tab === t.key ? 'var(--accent)' : 'var(--text2)',
              background: 'transparent',
              border: 'none',
              borderBottom: tab === t.key ? '2px solid var(--accent)' : '2px solid transparent',
              cursor: 'pointer',
              transition: 'color 0.15s, border-color 0.15s',
            }}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div style={{ marginTop: 20 }}>
        {tab === 'summary' && <SummaryTab analysis={displayedAnalysis} />}
        {tab === 'conversation' && <ConversationTab key={displayedAnalysis.sessionId} messages={displayedAnalysis.conversation} subagentConversations={displayedAnalysis.subagentConversations} />}
        {tab === 'cost' && <CostTab analysis={displayedAnalysis} />}
        {tab === 'trajectory' && <TrajectoryTab analysis={displayedAnalysis} />}
      </div>
    </div>
  );
}

/* ---- Shared styles ---- */
const container: React.CSSProperties = { padding: 24, maxWidth: 1200, margin: '0 auto' };
const backLink: React.CSSProperties = { fontSize: 14, color: 'var(--accent)', textDecoration: 'none' };
const toolBadge: React.CSSProperties = { display: 'inline-block', fontSize: 11, fontWeight: 600, padding: '2px 8px', borderRadius: 4, background: '#1f3a5f', color: '#58a6ff' };
const primaryBtn: React.CSSProperties = { padding: '6px 16px', fontSize: 13, fontWeight: 600, color: '#fff', background: 'var(--accent)', border: 'none', borderRadius: 6, cursor: 'pointer' };
const evalBtn: React.CSSProperties = { padding: '6px 16px', fontSize: 13, fontWeight: 600, color: '#fff', background: '#8957e5', border: 'none', borderRadius: 6, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 6 };
const selectStyle: React.CSSProperties = { padding: '5px 8px', fontSize: 13, color: 'var(--text)', background: 'var(--surface)', border: '1px solid var(--border)', borderRadius: 6, cursor: 'pointer' };
const miniSpinner: React.CSSProperties = { display: 'inline-block', width: 12, height: 12, border: '2px solid rgba(255,255,255,0.3)', borderTopColor: '#fff', borderRadius: '50%', animation: 'spin 0.8s linear infinite' };
const spinner: React.CSSProperties = { width: 32, height: 32, border: '3px solid var(--border)', borderTopColor: 'var(--accent)', borderRadius: '50%', margin: '0 auto', animation: 'spin 0.8s linear infinite' };

if (typeof document !== 'undefined' && !document.getElementById('xtrace-spinner-kf')) {
  const style = document.createElement('style');
  style.id = 'xtrace-spinner-kf';
  style.textContent = '@keyframes spin { to { transform: rotate(360deg); } }';
  document.head.appendChild(style);
}
