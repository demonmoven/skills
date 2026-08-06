import {
  PieChart,
  Pie,
  Cell,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import type { SessionAnalysis, HarnessAnalysis, UserCorrection, Inefficiency, SubagentStats } from '../types';

const COLORS = ['#58a6ff', '#3fb950', '#d29922', '#bc8cff', '#f85149'];
const C = { blue: '#58a6ff', green: '#3fb950', orange: '#d29922', purple: '#bc8cff', red: '#f85149' };

const card: React.CSSProperties = {
  background: 'var(--surface)',
  border: '1px solid var(--border)',
  borderRadius: 8,
  padding: 16,
};

const label: React.CSSProperties = {
  fontSize: 12,
  color: 'var(--text2)',
  marginBottom: 4,
};

const value: React.CSSProperties = {
  fontSize: 24,
  fontWeight: 600,
  color: 'var(--text)',
};

interface Props {
  analysis: SessionAnalysis;
}

export function SummaryTab({ analysis }: Props) {
  const totalTurns = analysis.userMsgCount + analysis.assistantMsgCount;
  const cacheHitRate = analysis.hasUsageData
    ? (
        (analysis.totalCacheRead /
          Math.max(analysis.totalInput + analysis.totalCacheRead, 1)) *
        100
      ).toFixed(1)
    : null;

  const modelsStr = Object.keys(analysis.modelsUsed).join(', ') || 'N/A';

  const metrics = [
    { title: 'Duration', val: analysis.durationStr },
    {
      title: 'Total Cost',
      val: analysis.hasUsageData ? `$${analysis.totalCost.toFixed(4)}` : 'N/A',
      sub: !analysis.hasUsageData ? 'No usage data available' : undefined,
    },
    { title: 'Turns', val: String(totalTurns) },
    { title: 'Tool Calls', val: String(analysis.totalToolCalls) },
    {
      title: 'Error Rate',
      val: `${(analysis.errorRate * 100).toFixed(1)}%`,
    },
    {
      title: 'Efficiency Score',
      val: `${(analysis.efficiencyScore * 100).toFixed(0)}%`,
    },
    {
      title: 'Cache Hit Rate',
      val: cacheHitRate !== null ? `${cacheHitRate}%` : 'N/A',
      sub: cacheHitRate === null ? 'No usage data available' : undefined,
    },
    { title: 'Models', val: modelsStr, small: true },
    {
      title: 'LLM Eval Score',
      val: analysis.evaluation
        ? analysis.evaluation.score.toFixed(2)
        : 'N/A',
      sub: analysis.evaluation
        ? `by ${analysis.evaluation.evaluatedBy}`
        : 'Click Evaluate to run',
      scoreColor: analysis.evaluation
        ? analysis.evaluation.score >= 0.7
          ? '#3fb950'
          : analysis.evaluation.score >= 0.4
            ? '#d29922'
            : '#f85149'
        : undefined,
    },
  ];

  // Tool usage distribution pie data
  const toolData = Object.entries(analysis.toolCounter)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 10)
    .map(([name, count]) => ({ name, value: count }));

  // Token distribution pie data
  const tokenData = analysis.hasUsageData
    ? [
        { name: 'Input', value: analysis.totalInput },
        { name: 'Output', value: analysis.totalOutput },
        { name: 'Cache Write', value: analysis.totalCacheWrite },
        { name: 'Cache Read', value: analysis.totalCacheRead },
      ].filter((d) => d.value > 0)
    : [];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
      {/* Metric cards grid */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(4, 1fr)',
          gap: 12,
        }}
      >
        {metrics.map((m) => (
          <div key={m.title} style={card}>
            <div style={label}>{m.title}</div>
            <div
              style={{
                ...value,
                fontSize: m.small ? 14 : 24,
                wordBreak: 'break-all',
                color: m.scoreColor ?? 'var(--text)',
              }}
            >
              {m.val}
            </div>
            {m.sub && (
              <div style={{ fontSize: 11, color: 'var(--text2)', marginTop: 4 }}>
                {m.sub}
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Charts row */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
        {/* Tool Usage Distribution */}
        <div style={card}>
          <div style={{ fontWeight: 600, marginBottom: 12, color: 'var(--text)' }}>
            Tool Usage Distribution
          </div>
          {toolData.length > 0 ? (
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={toolData}
                  dataKey="value"
                  nameKey="name"
                  cx="50%"
                  cy="50%"
                  outerRadius={100}
                  label={({ name, percent }) =>
                    `${name} ${(percent * 100).toFixed(0)}%`
                  }
                >
                  {toolData.map((_, i) => (
                    <Cell key={i} fill={COLORS[i % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{
                    background: 'var(--surface)',
                    border: '1px solid var(--border)',
                    borderRadius: 6,
                    color: 'var(--text)',
                  }}
                />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          ) : (
            <div style={{ color: 'var(--text2)', padding: 40, textAlign: 'center' }}>
              No tool usage data
            </div>
          )}
        </div>

        {/* Token Distribution */}
        <div style={card}>
          <div style={{ fontWeight: 600, marginBottom: 12, color: 'var(--text)' }}>
            Token Distribution
          </div>
          {tokenData.length > 0 ? (
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={tokenData}
                  dataKey="value"
                  nameKey="name"
                  cx="50%"
                  cy="50%"
                  outerRadius={100}
                  label={({ name, percent }) =>
                    `${name} ${(percent * 100).toFixed(0)}%`
                  }
                >
                  {tokenData.map((_, i) => (
                    <Cell key={i} fill={COLORS[i % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip
                  contentStyle={{
                    background: 'var(--surface)',
                    border: '1px solid var(--border)',
                    borderRadius: 6,
                    color: 'var(--text)',
                  }}
                  formatter={(v: number) => v.toLocaleString()}
                />
                <Legend />
              </PieChart>
            </ResponsiveContainer>
          ) : (
            <div style={{ color: 'var(--text2)', padding: 40, textAlign: 'center' }}>
              No token usage data available
            </div>
          )}
        </div>
      </div>

      {/* Subagent Statistics */}
      {analysis.subagentStats && analysis.subagentStats.totalSubagents > 0 && (
        <SubagentSection stats={analysis.subagentStats} />
      )}

      {/* LLM-as-Judge Evaluation */}
      {analysis.evaluation && <EvalSection evaluation={analysis.evaluation} />}

      {/* Harness Artifact Analysis */}
      {analysis.evaluation?.harnessAnalysis && (
        <HarnessSection
          harness={analysis.evaluation.harnessAnalysis}
          stats={analysis.evaluation.harnessStats}
        />
      )}
    </div>
  );
}

// ─── LLM Evaluation Section ────────────────────────────────────────

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

function EvalSection({ evaluation }: { evaluation: NonNullable<SessionAnalysis['evaluation']> }) {
  const scoreColor =
    evaluation.score >= 0.7 ? C.green : evaluation.score >= 0.4 ? C.orange : C.red;

  const dimensions = [
    { label: 'Task Completion', value: evaluation.taskCompletion, color: C.blue },
    { label: 'User Alignment', value: evaluation.userAlignment, color: C.green },
    { label: 'Efficiency', value: evaluation.efficiency, color: C.green },
    { label: 'Error Handling', value: evaluation.errorHandling, color: C.orange },
    { label: 'Decision Quality', value: evaluation.decisionQuality, color: C.blue },
    { label: 'Communication', value: evaluation.communication, color: C.purple },
    { label: 'Code Quality', value: evaluation.codeQuality, color: C.blue },
    { label: 'Harness Usage', value: evaluation.harnessUsage, color: C.purple },
  ];

  return (
    <div style={card}>
      <div style={{ fontWeight: 600, marginBottom: 16, color: 'var(--text)' }}>
        LLM-as-Judge Evaluation
      </div>

      <div style={{ display: 'flex', gap: 24, flexWrap: 'wrap' }}>
        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', minWidth: 120 }}>
          <div style={{ fontSize: 48, fontWeight: 700, color: scoreColor, lineHeight: 1 }}>
            {evaluation.score.toFixed(2)}
          </div>
          <div style={{ fontSize: 12, color: 'var(--text2)', marginTop: 4 }}>Overall Score</div>
          <div style={{ fontSize: 11, color: 'var(--text2)', marginTop: 2 }}>by {evaluation.evaluatedBy}</div>
        </div>

        <div style={{ flex: 1, minWidth: 300, display: 'flex', flexDirection: 'column', gap: 10 }}>
          {dimensions.map((d) => (
            <div key={d.label}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 3 }}>
                <span style={{ fontSize: 13, color: 'var(--text)' }}>{d.label}</span>
                <span style={{ fontSize: 13, fontWeight: 600, color: d.color }}>{(d.value * 100).toFixed(0)}%</span>
              </div>
              <div style={{ height: 6, borderRadius: 3, background: 'var(--border)', overflow: 'hidden' }}>
                <div style={{ height: '100%', width: `${d.value * 100}%`, background: d.color, borderRadius: 3, transition: 'width 0.3s ease' }} />
              </div>
            </div>
          ))}
        </div>
      </div>

      <div style={{ marginTop: 16, fontSize: 14, lineHeight: 1.7, color: 'var(--text)', whiteSpace: 'pre-wrap', padding: 16, background: 'var(--bg)', borderRadius: 6, border: '1px solid var(--border)' }}>
        {evaluation.comment}
      </div>

      {/* User Corrections */}
      {evaluation.userCorrections && evaluation.userCorrections.length > 0 && (
        <UserCorrectionsSection corrections={evaluation.userCorrections} />
      )}

      {/* Inefficiencies */}
      {evaluation.inefficiencies && evaluation.inefficiencies.length > 0 && (
        <InefficienciesSection inefficiencies={evaluation.inefficiencies} />
      )}

      <div style={{ fontSize: 11, color: 'var(--text2)', marginTop: 8 }}>
        Evaluated by {evaluation.evaluatedBy} at {new Date(evaluation.evaluatedAt).toLocaleString()}
      </div>
    </div>
  );
}

// ─── User Corrections Section ──────────────────────────────────────

const severityColor: Record<string, string> = {
  critical: '#f85149',
  high: '#d29922',
  medium: '#8b949e',
};

function UserCorrectionsSection({ corrections }: { corrections: UserCorrection[] }) {
  return (
    <div style={{ marginTop: 16 }}>
      <div style={{ fontSize: 14, fontWeight: 600, color: C.red, marginBottom: 8 }}>
        User Corrections ({corrections.length})
      </div>
      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={thStyle}>Turn</th>
              <th style={thStyle}>Severity</th>
              <th style={thStyle}>Assistant Behavior</th>
              <th style={thStyle}>User Expected</th>
              <th style={thStyle}>Pattern</th>
            </tr>
          </thead>
          <tbody>
            {corrections.map((c, i) => (
              <tr key={i}>
                <td style={{ ...tdStyle, fontFamily: 'monospace', fontSize: 12 }}>{c.turn}</td>
                <td style={{ ...tdStyle, fontWeight: 600, color: severityColor[c.severity] ?? '#8b949e' }}>{c.severity}</td>
                <td style={{ ...tdStyle, fontSize: 12, maxWidth: 200 }}>{c.behavior}</td>
                <td style={{ ...tdStyle, fontSize: 12, maxWidth: 200 }}>{c.expectation}</td>
                <td style={{ ...tdStyle, fontSize: 12, maxWidth: 200 }}>{c.gap}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ─── Inefficiencies Section ───────────────────────────────────────

function InefficienciesSection({ inefficiencies }: { inefficiencies: Inefficiency[] }) {
  const totalWasted = inefficiencies.reduce((sum, item) => sum + item.wastedTurns, 0);
  return (
    <div style={{ marginTop: 16 }}>
      <div style={{ fontSize: 14, fontWeight: 600, color: C.orange, marginBottom: 8 }}>
        Inefficiency Patterns ({inefficiencies.length}, ~{totalWasted} wasted turns)
      </div>
      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr>
              <th style={thStyle}>Pattern</th>
              <th style={thStyle}>Wasted</th>
              <th style={thStyle}>Description</th>
              <th style={thStyle}>Suggestion</th>
            </tr>
          </thead>
          <tbody>
            {inefficiencies.map((item, i) => (
              <tr key={i}>
                <td style={{ ...tdStyle, fontWeight: 600, fontSize: 12, whiteSpace: 'nowrap' }}>{item.pattern}</td>
                <td style={{ ...tdStyle, textAlign: 'center', color: C.orange }}>{item.wastedTurns}</td>
                <td style={{ ...tdStyle, fontSize: 12, maxWidth: 250 }}>{item.description}</td>
                <td style={{ ...tdStyle, fontSize: 12, maxWidth: 250 }}>{item.suggestion}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ─── Harness Analysis Section ──────────────────────────────────────

function HarnessSection({ harness, stats }: {
  harness: HarnessAnalysis;
  stats?: { totalArtifacts: number; usedArtifacts: number; unusedArtifacts: number; repoScanned: boolean };
}) {
  const mechItems = [
    { label: 'Auto-injected (e.g. CLAUDE.md)', text: harness.mechanismEffectiveness.autoInjected },
    { label: 'Hard constraints (e.g. pre-commit)', text: harness.mechanismEffectiveness.hardConstraints },
    { label: 'Explicit invocation (e.g. skills)', text: harness.mechanismEffectiveness.explicitInvocation },
    { label: 'Passive documentation', text: harness.mechanismEffectiveness.passiveDocumentation },
  ];

  return (
    <div style={card}>
      <div style={{ fontWeight: 600, marginBottom: 16, color: 'var(--text)' }}>Harness Artifact Analysis</div>

      <div style={{ display: 'flex', gap: 16, marginBottom: 20, flexWrap: 'wrap' }}>
        <StatBadge label="Used" value={stats?.usedArtifacts ?? harness.usedArtifacts.length} color={C.green} />
        <StatBadge label="Unused" value={stats?.unusedArtifacts ?? harness.unusedArtifacts.length} color={C.red} />
        <StatBadge label="Total" value={stats?.totalArtifacts ?? (harness.usedArtifacts.length + harness.unusedArtifacts.length)} color={C.blue} />
        {stats?.repoScanned && (
          <div style={{ alignSelf: 'center', fontSize: 11, color: 'var(--text2)' }}>based on repo scan</div>
        )}
      </div>

      {harness.usedArtifacts.length > 0 && (
        <div style={{ marginBottom: 20 }}>
          <div style={{ fontSize: 14, fontWeight: 600, color: C.green, marginBottom: 8 }}>Artifacts That Worked</div>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead><tr><th style={thStyle}>Path</th><th style={thStyle}>Reads</th><th style={thStyle}>Mechanism</th><th style={thStyle}>Evidence</th></tr></thead>
              <tbody>
                {harness.usedArtifacts.map((a, i) => (
                  <tr key={i}>
                    <td style={{ ...tdStyle, fontFamily: 'monospace', fontSize: 12 }}>{a.path}</td>
                    <td style={tdStyle}>{a.readCount}</td>
                    <td style={{ ...tdStyle, fontSize: 12 }}>{a.mechanism}</td>
                    <td style={{ ...tdStyle, fontSize: 12, maxWidth: 300 }}>{a.evidence}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {harness.unusedArtifacts.length > 0 && (
        <div style={{ marginBottom: 20 }}>
          <div style={{ fontSize: 14, fontWeight: 600, color: C.red, marginBottom: 8 }}>Should Have Used</div>
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
              <thead><tr><th style={thStyle}>Path</th><th style={thStyle}>Potential Value If Used</th></tr></thead>
              <tbody>
                {harness.unusedArtifacts.map((a, i) => (
                  <tr key={i}>
                    <td style={{ ...tdStyle, fontFamily: 'monospace', fontSize: 12 }}>{a.path}</td>
                    <td style={{ ...tdStyle, fontSize: 12 }}>{a.intendedPurpose}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      <div style={{ marginBottom: 20 }}>
        <div style={{ fontSize: 14, fontWeight: 600, color: 'var(--text)', marginBottom: 8 }}>Mechanism Effectiveness</div>
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {mechItems.filter(m => m.text).map((m, i) => (
            <div key={i} style={{ padding: 12, background: 'var(--bg)', borderRadius: 6, border: '1px solid var(--border)' }}>
              <div style={{ fontSize: 12, fontWeight: 600, color: 'var(--text2)', marginBottom: 4 }}>{m.label}</div>
              <div style={{ fontSize: 13, color: 'var(--text)', lineHeight: 1.5 }}>{m.text}</div>
            </div>
          ))}
        </div>
      </div>

      {harness.summary && (
        <div style={{ fontSize: 14, lineHeight: 1.7, color: 'var(--text)', whiteSpace: 'pre-wrap', padding: 16, background: 'var(--bg)', borderRadius: 6, border: '1px solid var(--border)' }}>
          {harness.summary}
        </div>
      )}
    </div>
  );
}

function StatBadge({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div style={{ textAlign: 'center', minWidth: 70 }}>
      <div style={{ fontSize: 24, fontWeight: 700, color }}>{value}</div>
      <div style={{ fontSize: 11, color: 'var(--text2)' }}>{label}</div>
    </div>
  );
}

// ─── Subagent Section ─────────────────────────────────────────────

const AGENT_TYPE_COLORS: Record<string, string> = {
  'general-purpose': '#58a6ff',
  'Explore': '#3fb950',
  'Plan': '#d29922',
  'code-reviewer': '#bc8cff',
};

function SubagentSection({ stats }: { stats: SubagentStats }) {
  const typeData = Object.entries(stats.agentTypeDistribution).map(([name, value]) => ({
    name,
    value,
  }));

  return (
    <div style={card}>
      <div style={{ fontWeight: 600, marginBottom: 12, color: 'var(--text)', fontSize: 16 }}>
        Subagent Statistics
      </div>

      {/* Summary metrics */}
      <div style={{ display: 'flex', gap: 24, marginBottom: 16 }}>
        <div>
          <div style={label}>Subagents</div>
          <div style={{ fontSize: 20, fontWeight: 600, color: C.blue }}>{stats.totalSubagents}</div>
        </div>
        <div>
          <div style={label}>Total Tokens</div>
          <div style={{ fontSize: 20, fontWeight: 600, color: 'var(--text)' }}>
            {stats.totalSubagentTokens.toLocaleString()}
          </div>
        </div>
        <div>
          <div style={label}>Total Cost</div>
          <div style={{ fontSize: 20, fontWeight: 600, color: C.green }}>
            ${stats.totalSubagentCost.toFixed(4)}
          </div>
        </div>
        <div>
          <div style={label}>Error Rate</div>
          <div style={{ fontSize: 20, fontWeight: 600, color: stats.subagentErrorRate > 10 ? C.red : 'var(--text)' }}>
            {stats.subagentErrorRate}%
          </div>
        </div>
      </div>

      {/* Agent type distribution + subagent table */}
      <div style={{ display: 'grid', gridTemplateColumns: typeData.length > 0 ? '200px 1fr' : '1fr', gap: 16 }}>
        {/* Type distribution pie */}
        {typeData.length > 0 && (
          <div>
            <div style={{ ...label, marginBottom: 8 }}>Agent Types</div>
            <ResponsiveContainer width="100%" height={160}>
              <PieChart>
                <Pie data={typeData} dataKey="value" nameKey="name" cx="50%" cy="50%" outerRadius={60}
                  label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                >
                  {typeData.map((entry, i) => (
                    <Cell key={entry.name} fill={AGENT_TYPE_COLORS[entry.name] ?? COLORS[i % COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
        )}

        {/* Subagent table */}
        <div style={{ overflow: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
            <thead>
              <tr>
                <th style={thStyle}>Name</th>
                <th style={thStyle}>Type</th>
                <th style={thStyle}>Description</th>
                <th style={{ ...thStyle, textAlign: 'right' }}>Messages</th>
                <th style={{ ...thStyle, textAlign: 'right' }}>Tokens</th>
                <th style={{ ...thStyle, textAlign: 'right' }}>Cost</th>
              </tr>
            </thead>
            <tbody>
              {stats.subagents.map((sub) => (
                <tr key={sub.agentId}>
                  <td style={tdStyle}>
                    <code style={{ fontSize: 12, color: C.blue }}>{sub.slug}</code>
                  </td>
                  <td style={tdStyle}>
                    <span style={{
                      background: AGENT_TYPE_COLORS[sub.agentType] ?? '#666',
                      color: '#fff',
                      padding: '2px 6px',
                      borderRadius: 4,
                      fontSize: 11,
                    }}>
                      {sub.agentType}
                    </span>
                  </td>
                  <td style={{ ...tdStyle, maxWidth: 300, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {sub.description}
                  </td>
                  <td style={{ ...tdStyle, textAlign: 'right' }}>{sub.messageCount}</td>
                  <td style={{ ...tdStyle, textAlign: 'right' }}>{sub.totalTokens.toLocaleString()}</td>
                  <td style={{ ...tdStyle, textAlign: 'right' }}>${sub.totalCost.toFixed(4)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
