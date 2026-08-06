#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

// --- CLI Parsing ---
const args = process.argv.slice(2);
const sessionDir = args.find(a => !a.startsWith('--'));
const outFlag = args.indexOf('--out');
const outPath = outFlag !== -1 ? args[outFlag + 1] : null;

if (!sessionDir) {
  console.error('Usage: trace-analyzer.js <session-dir> [--out <timeline.json>]');
  process.exit(1);
}

if (!fs.existsSync(sessionDir)) {
  console.error(`Error: Session directory not found: ${sessionDir}`);
  process.exit(1);
}

// --- Session Parser ---
function readSessionFiles(dir) {
  const files = fs.readdirSync(dir)
    .filter(f => f.endsWith('.json'))
    .sort();
  const entries = [];
  for (const f of files) {
    try {
      const parsed = JSON.parse(fs.readFileSync(path.join(dir, f), 'utf8'));
      // Handle both single-object files and array-of-events files
      if (Array.isArray(parsed)) {
        entries.push(...parsed);
      } else {
        entries.push(parsed);
      }
    } catch { /* skip unparseable files */ }
  }
  return entries;
}

function hashSnapshot(snapshot) {
  if (!snapshot) return null;
  const raw = typeof snapshot === 'string' ? snapshot : JSON.stringify(snapshot);
  return crypto.createHash('sha256').update(raw).digest('hex').slice(0, 16);
}

function buildTimeline(rawEntries) {
  const events = [];
  let testCase = 'unknown';
  let startedAt = null;
  let endedAt = null;
  let failurePoint = null;

  for (let i = 0; i < rawEntries.length; i++) {
    const entry = rawEntries[i];
    const ts = entry.timestamp || entry.time || entry.ts || new Date().toISOString();
    if (i === 0) startedAt = ts;
    endedAt = ts;

    if (entry.test_case || entry.testCase || entry.name) {
      testCase = entry.test_case || entry.testCase || entry.name;
    }

    const actionName = entry.action || entry.method || entry.type || entry.command || 'unknown';
    const eventType = categorizeAction(actionName);

    const ev = {
      seq: i + 1,
      timestamp: ts,
      type: eventType,
      action: actionName,
      params: entry.params || entry.arguments || entry.args || {},
      duration_ms: entry.duration_ms || entry.duration || entry.elapsed || 0,
      a11y_snapshot_hash: hashSnapshot(entry.snapshot || entry.a11y_snapshot || entry.accessibility),
      network_events: (entry.network_events || entry.network || []).map(n => ({
        method: n.method || 'GET',
        url: n.url || '',
        status: n.status || 0,
        duration_ms: n.duration_ms || n.duration || 0,
      })),
      console_logs: (entry.console_logs || entry.console || entry.logs || []).map(l => ({
        level: l.level || 'log',
        text: l.text || l.message || String(l),
      })),
      screenshot_path: entry.screenshot_path || entry.screenshot || null,
    };

    // Detect failure
    if (entry.error || entry.status === 'failed' || entry.success === false) {
      if (!failurePoint) {
        failurePoint = {
          at_seq: ev.seq,
          expected: entry.expected || entry.assertion || '',
          actual: entry.actual || entry.error || entry.message || '',
        };
      }
    }

    events.push(ev);
  }

  const verdict = failurePoint ? 'FAILED' : (events.length > 0 ? 'PASSED' : 'BLOCKED');

  return {
    test_case: testCase,
    started_at: startedAt,
    ended_at: endedAt,
    verdict,
    events,
    failure_point: failurePoint,
    detected_patterns: [],
  };
}

function categorizeAction(action) {
  const nav = ['navigate', 'goto', 'browser_navigate', 'page.goto'];
  const assert = ['assert', 'expect', 'verify', 'check', 'browser_snapshot', 'snapshot'];
  const lower = action.toLowerCase();
  if (nav.some(n => lower.includes(n))) return 'navigation';
  if (assert.some(a => lower.includes(a))) return 'assertion';
  return 'action';
}

// --- Pattern Detectors ---
function detectHoverLoss(events) {
  for (let i = 0; i < events.length - 1; i++) {
    const curr = events[i];
    const next = events[i + 1];
    if (!next.action.toLowerCase().includes('click')) continue;
    if (curr.a11y_snapshot_hash && next.a11y_snapshot_hash && curr.a11y_snapshot_hash !== next.a11y_snapshot_hash) {
      const failed = next.params?.error || (events[i + 2]?.action?.toLowerCase().includes('fail'));
      if (failed) {
        return {
          pattern: 'hover_loss',
          description: `Element present at seq ${curr.seq} disappeared before click at seq ${next.seq}`,
          evidence: { before_seq: curr.seq, click_seq: next.seq, snapshot_before: curr.a11y_snapshot_hash, snapshot_after: next.a11y_snapshot_hash },
        };
      }
    }
  }
  return null;
}

function detectAnimationTiming(events) {
  for (let i = 1; i < events.length; i++) {
    const ev = events[i];
    if (!ev.action.toLowerCase().includes('click')) continue;
    const prev = events[i - 1];
    if (prev.duration_ms > 0 && prev.duration_ms < 300 && (ev.params?.error || ev.action.includes('fail'))) {
      return {
        pattern: 'animation_timing',
        description: `Element appeared ${prev.duration_ms}ms before failed click at seq ${ev.seq}`,
        evidence: { appear_seq: prev.seq, click_seq: ev.seq, appear_duration_ms: prev.duration_ms },
      };
    }
  }
  return null;
}

function detectRenderRace(events) {
  const hashes = events.map(e => e.a11y_snapshot_hash).filter(Boolean);
  for (let i = 0; i < hashes.length - 2; i++) {
    if (hashes[i] === hashes[i + 2] && hashes[i] !== hashes[i + 1]) {
      return {
        pattern: 'render_race',
        description: `Snapshot oscillation: seq ${i + 1} = seq ${i + 3}, different from seq ${i + 2}`,
        evidence: { seq_a: i + 1, seq_b: i + 2, seq_c: i + 3, hash_a: hashes[i], hash_b: hashes[i + 1] },
      };
    }
  }
  return null;
}

function detectNetworkDependency(events, failurePoint) {
  if (!failurePoint) return null;
  const failSeq = failurePoint.at_seq;
  for (let i = failSeq - 2; i >= 0 && i >= failSeq - 4; i--) {
    const ev = events[i];
    if (!ev) continue;
    const badReq = ev.network_events.find(n => n.status >= 400 || n.status === 0);
    if (badReq) {
      return {
        pattern: 'network_dependency',
        description: `Network error (${badReq.status} ${badReq.url}) preceded failure at seq ${failSeq}`,
        evidence: { network_seq: ev.seq, fail_seq: failSeq, request: badReq },
      };
    }
  }
  return null;
}

function detectFocusSteal(events) {
  for (let i = 0; i < events.length; i++) {
    const ev = events[i];
    const isType = ev.action.toLowerCase().includes('type') || ev.action.toLowerCase().includes('fill') || ev.action.toLowerCase().includes('press');
    if (!isType) continue;
    if (ev.params?.error && (ev.params.error.includes('focus') || ev.params.error.includes('detached'))) {
      return {
        pattern: 'focus_steal',
        description: `Type action at seq ${ev.seq} failed due to focus/detach issue`,
        evidence: { seq: ev.seq, error: ev.params.error },
      };
    }
    if (i > 0 && events[i - 1].a11y_snapshot_hash && ev.a11y_snapshot_hash && events[i - 1].a11y_snapshot_hash !== ev.a11y_snapshot_hash) {
      if (ev.params?.error) {
        return {
          pattern: 'focus_steal',
          description: `Focused element changed between seq ${i} and seq ${ev.seq}, type action failed`,
          evidence: { prev_seq: events[i - 1].seq, type_seq: ev.seq, snapshot_before: events[i - 1].a11y_snapshot_hash, snapshot_after: ev.a11y_snapshot_hash },
        };
      }
    }
  }
  return null;
}

function detectZIndexOcclusion(events) {
  for (const ev of events) {
    const errStr = JSON.stringify(ev.params?.error || '').toLowerCase();
    if (errStr.includes('not visible') || errStr.includes('covered') || errStr.includes('intercepted') || errStr.includes('obscured')) {
      return {
        pattern: 'z_index_occlusion',
        description: `Element exists but click blocked at seq ${ev.seq}: covered or not visible`,
        evidence: { seq: ev.seq, error: ev.params?.error },
      };
    }
  }
  return null;
}

function runPatternDetectors(timeline) {
  const detectors = [
    detectHoverLoss,
    detectAnimationTiming,
    detectRenderRace,
    (events) => detectNetworkDependency(events, timeline.failure_point),
    detectFocusSteal,
    detectZIndexOcclusion,
  ];
  const patterns = [];
  for (const detect of detectors) {
    const result = detect(timeline.events);
    if (result) patterns.push(result);
  }
  return patterns;
}

// --- Main ---
// 职责分工：
//   本脚本：解析 --save-session JSON → timeline.json + 6 种失败模式检测
//   trace-analyzer MCP server (@metoto/playwright-trace-analyzer-mcp)：
//     作为独立 MCP 服务器运行，提供 trace.zip 深度分析（过滤、网络分析、截图关联）
//     由诊断 Agent 通过 MCP 工具直接调用，不在本脚本中集成
function main() {
  const rawEntries = readSessionFiles(sessionDir);

  if (rawEntries.length === 0) {
    console.error('Warning: No session data found in ' + sessionDir);
  }

  const timeline = buildTimeline(rawEntries);
  timeline.detected_patterns = runPatternDetectors(timeline);

  const outputFile = outPath || path.join(sessionDir, 'timeline.json');
  fs.mkdirSync(path.dirname(outputFile), { recursive: true });
  fs.writeFileSync(outputFile, JSON.stringify(timeline, null, 2));
  console.log(JSON.stringify({ status: 'ok', output: outputFile, verdict: timeline.verdict, patterns: timeline.detected_patterns.length }));
}

main();
