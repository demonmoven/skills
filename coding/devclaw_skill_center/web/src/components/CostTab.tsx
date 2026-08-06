import {
  LineChart,
  Line,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
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

const labelStyle: React.CSSProperties = {
  fontSize: 12,
  color: 'var(--text2)',
  marginBottom: 4,
};

const valueStyle: React.CSSProperties = {
  fontSize: 24,
  fontWeight: 600,
  color: 'var(--text)',
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

export function CostTab({ analysis }: Props) {
  if (!analysis.hasUsageData) {
    return (
      <div
        style={{
          ...card,
          textAlign: 'center',
          padding: 48,
          color: 'var(--text2)',
        }}
      >
        <div style={{ fontSize: 40, marginBottom: 12 }}>📊</div>
        <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 8, color: 'var(--text)' }}>
          No Usage Data
        </div>
        <div>This tool does not provide token usage data</div>
      </div>
    );
  }

  const metrics = [
    { title: 'Total Cost', val: `$${analysis.totalCost.toFixed(4)}` },
    { title: 'Input Tokens', val: analysis.totalInput.toLocaleString() },
    { title: 'Output Tokens', val: analysis.totalOutput.toLocaleString() },
    { title: 'Cache Read', val: analysis.totalCacheRead.toLocaleString() },
  ];

  // Cumulative cost chart data
  const cumulativeData = analysis.cumulativeCosts.map((c) => ({
    turn: `T${c.turn}`,
    cost: Number(c.cost.toFixed(6)),
  }));

  // Per-turn stacked bar data
  const perTurnData = analysis.perTurnData.map((t) => ({
    turn: `T${t.turn}`,
    input: t.inputTokens,
    output: t.outputTokens,
    cacheWrite: t.cacheWrite,
    cacheRead: t.cacheRead,
  }));

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
      {/* Metric cards */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 12 }}>
        {metrics.map((m) => (
          <div key={m.title} style={card}>
            <div style={labelStyle}>{m.title}</div>
            <div style={valueStyle}>{m.val}</div>
          </div>
        ))}
      </div>

      {/* Cumulative Cost line chart */}
      <div style={card}>
        <div style={{ fontWeight: 600, marginBottom: 12, color: 'var(--text)' }}>
          Cumulative Cost
        </div>
        {cumulativeData.length > 0 ? (
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={cumulativeData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
              <XAxis dataKey="turn" stroke="var(--text2)" fontSize={12} />
              <YAxis stroke="var(--text2)" fontSize={12} tickFormatter={(v: number) => `$${v}`} />
              <Tooltip contentStyle={tooltipStyle} formatter={(v: number) => `$${v.toFixed(6)}`} />
              <Line
                type="monotone"
                dataKey="cost"
                stroke={COLORS.blue}
                strokeWidth={2}
                dot={{ r: 3 }}
                name="Cumulative Cost"
              />
            </LineChart>
          </ResponsiveContainer>
        ) : (
          <div style={{ color: 'var(--text2)', textAlign: 'center', padding: 40 }}>
            No cost data
          </div>
        )}
      </div>

      {/* Per-Turn Token Usage stacked bar chart */}
      <div style={card}>
        <div style={{ fontWeight: 600, marginBottom: 12, color: 'var(--text)' }}>
          Per-Turn Token Usage
        </div>
        {perTurnData.length > 0 ? (
          <ResponsiveContainer width="100%" height={300}>
            <BarChart data={perTurnData}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
              <XAxis dataKey="turn" stroke="var(--text2)" fontSize={12} />
              <YAxis stroke="var(--text2)" fontSize={12} />
              <Tooltip contentStyle={tooltipStyle} formatter={(v: number) => v.toLocaleString()} />
              <Legend />
              <Bar dataKey="input" stackId="a" fill={COLORS.blue} name="Input" />
              <Bar dataKey="output" stackId="a" fill={COLORS.green} name="Output" />
              <Bar dataKey="cacheWrite" stackId="a" fill={COLORS.orange} name="Cache Write" />
              <Bar dataKey="cacheRead" stackId="a" fill={COLORS.purple} name="Cache Read" />
            </BarChart>
          </ResponsiveContainer>
        ) : (
          <div style={{ color: 'var(--text2)', textAlign: 'center', padding: 40 }}>
            No per-turn data
          </div>
        )}
      </div>

      {/* Per-Turn details table */}
      <div style={card}>
        <div style={{ fontWeight: 600, marginBottom: 12, color: 'var(--text)' }}>
          Per-Turn Details
        </div>
        <div style={{ overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr>
                <th style={thStyle}>Turn</th>
                <th style={thStyle}>Model</th>
                <th style={thStyle}>Input</th>
                <th style={thStyle}>Output</th>
                <th style={thStyle}>Cache W</th>
                <th style={thStyle}>Cache R</th>
                <th style={thStyle}>Cost</th>
              </tr>
            </thead>
            <tbody>
              {analysis.perTurnData.map((t) => (
                <tr key={t.turn}>
                  <td style={tdStyle}>{t.turn}</td>
                  <td style={{ ...tdStyle, fontSize: 12, color: 'var(--text2)' }}>
                    {t.model || '-'}
                  </td>
                  <td style={tdStyle}>{t.inputTokens.toLocaleString()}</td>
                  <td style={tdStyle}>{t.outputTokens.toLocaleString()}</td>
                  <td style={tdStyle}>{t.cacheWrite.toLocaleString()}</td>
                  <td style={tdStyle}>{t.cacheRead.toLocaleString()}</td>
                  <td style={tdStyle}>${t.cost.toFixed(6)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
