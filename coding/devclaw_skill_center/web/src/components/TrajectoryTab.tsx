import {
  BarChart,
  Bar,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import type { SessionAnalysis } from '../types';

const COLORS = {
  blue: '#58a6ff',
  green: '#3fb950',
  orange: '#d29922',
  purple: '#bc8cff',
  red: '#f85149',
};

const card: React.CSSProperties = {
  background: 'var(--surface)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  padding: 16,
};

const tooltipStyle = {
  background: 'var(--surface)',
  border: '1px solid var(--border)',
  borderRadius: 6,
  color: 'var(--text)',
};

const thStyle: React.CSSProperties = {
  textAlign: 'left' as const,
  padding: '8px 12px',
  fontSize: 12,
  color: 'var(--text2)',
  borderBottom: '1px solid var(--border)',
  fontWeight: 600,
};

const tdStyle: React.CSSProperties = {
  padding: '8px 12px',
  fontSize: 13,
  color: 'var(--text)',
  borderBottom: '1px solid var(--border)',
};

interface Props {
  analysis: SessionAnalysis;
}

export function TrajectoryTab({ analysis }: Props) {
  const avgLatency =
    analysis.latencies.length > 0
      ? analysis.latencies.reduce((a, b) => a + b, 0) / analysis.latencies.length
      : 0;

  const progressMetrics = [
    {
      title: 'Efficiency Score',
      value: analysis.efficiencyScore,
      display: `${(analysis.efficiencyScore * 100).toFixed(0)}%`,
      color: COLORS.green,
    },
    {
      title: 'Error Rate',
      value: analysis.errorRate,
      display: `${(analysis.errorRate * 100).toFixed(1)}%`,
      color: COLORS.red,
    },
    {
      title: 'Thinking Ratio',
      value: analysis.thinkingRatio,
      display: `${(analysis.thinkingRatio * 100).toFixed(1)}%`,
      color: COLORS.purple,
    },
    {
      title: 'Tool Diversity',
      value: Math.min(analysis.toolDiversity / 20, 1), // normalize for bar
      display: String(analysis.toolDiversity),
      color: COLORS.blue,
    },
    {
      title: 'Retry Sequences',
      value: Math.min(analysis.retries.length / 10, 1),
      display: String(analysis.retries.length),
      color: COLORS.orange,
    },
    {
      title: 'Avg Response Latency',
      value: Math.min(avgLatency / 30000, 1), // normalize against 30s
      display: avgLatency > 0 ? `${(avgLatency / 1000).toFixed(1)}s` : 'N/A',
      color: COLORS.blue,
    },
  ];

  // Tool call frequency — horizontal bar chart
  const toolFreqData = Object.entries(analysis.toolCounter)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 15)
    .map(([name, count]) => ({ name, count }));

  // Latency line chart data
  const latencyData = analysis.latencies.map((l, i) => ({
    turn: `T${i + 1}`,
    latency: Number((l / 1000).toFixed(2)),
  }));

  // Tool error rates table data
  const toolErrors = Object.entries(analysis.toolErrorMap)
    .map(([name, data]) => ({
      name,
      calls: data.calls,
      errors: data.errors,
      rate: data.calls > 0 ? ((data.errors / data.calls) * 100).toFixed(1) : '0.0',
    }))
    .sort((a, b) => b.errors - a.errors);

  // Files touched data
  const filesTouched = Object.entries(analysis.filesEdited)
    .sort((a, b) => b[1] - a[1])
    .map(([file, edits]) => ({ file, edits }));

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
      {/* Progress bar metrics */}
      <div style={card}>
        <div style={{ fontWeight: 600, marginBottom: 16, color: 'var(--text)' }}>
          Session Metrics
        </div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
          {progressMetrics.map((m) => (
            <div key={m.title}>
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  marginBottom: 4,
                }}
              >
                <span style={{ fontSize: 13, color: 'var(--text)' }}>{m.title}</span>
                <span style={{ fontSize: 13, fontWeight: 600, color: m.color }}>
                  {m.display}
                </span>
              </div>
              <div
                style={{
                  height: 6,
                  borderRadius: 3,
                  background: 'var(--border)',
                  overflow: 'hidden',
                }}
              >
                <div
                  style={{
                    height: '100%',
                    width: `${Math.min(m.value * 100, 100)}%`,
                    background: m.color,
                    borderRadius: 3,
                    transition: 'width 0.3s ease',
                  }}
                />
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Tool Call Frequency — horizontal bar chart */}
      <div style={card}>
        <div style={{ fontWeight: 600, marginBottom: 12, color: 'var(--text)' }}>
          Tool Call Frequency
        </div>
        {toolFreqData.length > 0 ? (
          <ResponsiveContainer width="100%" height={Math.max(250, toolFreqData.length * 30)}>
            <BarChart data={toolFreqData} layout="vertical" margin={{ left: 120 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
              <XAxis type="number" stroke="var(--text2)" fontSize={12} />
              <YAxis
                type="category"
                dataKey="name"
                stroke="var(--text2)"
                fontSize={12}
                width={110}
              />
              <Tooltip contentStyle={tooltipStyle} />
              <Bar dataKey="count" fill={COLORS.blue} name="Calls" radius={[0, 4, 4, 0]} />
            </BarChart>
          </ResponsiveContainer>
        ) : (
          <div style={{ color: 'var(--text2)', textAlign: 'center', padding: 40 }}>
            No tool call data
          </div>
        )}
      </div>

      {/* Response Latency line chart */}
      <div style={card}>
        <div style={{ fontWeight: 600, marginBottom: 12, color: 'var(--text)' }}>
          Response Latency
        </div>
        {latencyData.length > 0 ? (
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={latencyData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
              <XAxis dataKey="turn" stroke="var(--text2)" fontSize={12} />
              <YAxis stroke="var(--text2)" fontSize={12} tickFormatter={(v: number) => `${v}s`} />
              <Tooltip contentStyle={tooltipStyle} formatter={(v: number) => `${v}s`} />
              <Line
                type="monotone"
                dataKey="latency"
                stroke={COLORS.orange}
                strokeWidth={2}
                dot={{ r: 3 }}
                name="Latency"
              />
            </LineChart>
          </ResponsiveContainer>
        ) : (
          <div style={{ color: 'var(--text2)', textAlign: 'center', padding: 40 }}>
            No latency data
          </div>
        )}
      </div>

      {/* Tool Error Rates table */}
      {toolErrors.length > 0 && (
        <div style={card}>
          <div style={{ fontWeight: 600, marginBottom: 12, color: 'var(--text)' }}>
            Tool Error Rates
          </div>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  <th style={thStyle}>Tool</th>
                  <th style={thStyle}>Calls</th>
                  <th style={thStyle}>Errors</th>
                  <th style={thStyle}>Error Rate</th>
                </tr>
              </thead>
              <tbody>
                {toolErrors.map((t) => (
                  <tr key={t.name}>
                    <td style={tdStyle}>{t.name}</td>
                    <td style={tdStyle}>{t.calls}</td>
                    <td style={{ ...tdStyle, color: t.errors > 0 ? COLORS.red : 'var(--text)' }}>
                      {t.errors}
                    </td>
                    <td style={{ ...tdStyle, color: Number(t.rate) > 0 ? COLORS.red : 'var(--text)' }}>
                      {t.rate}%
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Retry Sequences table */}
      {analysis.retries.length > 0 && (
        <div style={card}>
          <div style={{ fontWeight: 600, marginBottom: 12, color: 'var(--text)' }}>
            Retry Sequences
          </div>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  <th style={thStyle}>Tool</th>
                  <th style={thStyle}>Consecutive Retries</th>
                </tr>
              </thead>
              <tbody>
                {analysis.retries.map((r, i) => (
                  <tr key={i}>
                    <td style={tdStyle}>{r.tool}</td>
                    <td style={{ ...tdStyle, color: COLORS.orange }}>{r.count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Files Touched table */}
      {filesTouched.length > 0 && (
        <div style={card}>
          <div style={{ fontWeight: 600, marginBottom: 12, color: 'var(--text)' }}>
            Files Touched
          </div>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead>
                <tr>
                  <th style={thStyle}>File</th>
                  <th style={thStyle}>Edits</th>
                </tr>
              </thead>
              <tbody>
                {filesTouched.map((f) => (
                  <tr key={f.file}>
                    <td style={{ ...tdStyle, fontFamily: 'monospace', fontSize: 12 }}>
                      {f.file}
                    </td>
                    <td style={tdStyle}>{f.edits}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

    </div>
  );
}


