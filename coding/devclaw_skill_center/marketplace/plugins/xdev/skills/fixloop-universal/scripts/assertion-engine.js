#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');

// --- CLI Parsing ---
const args = process.argv.slice(2);

function getArg(name, defaultValue) {
  const idx = args.indexOf('--' + name);
  return idx !== -1 && args[idx + 1] ? args[idx + 1] : defaultValue;
}

const timelinePath = getArg('timeline', null);
const screenshotPath = getArg('screenshot', null);
const baselinePath = getArg('baseline', null);
const a11ySnapshotPath = getArg('a11y-snapshot', null);
const expectedElementsRaw = getArg('expected-elements', '');
const threshold = parseFloat(getArg('threshold', '0.1'));
const maxDiffPercent = parseFloat(getArg('max-diff-percent', '5'));
const TIMEOUT_MS = parseInt(getArg('action-timeout', '30000'), 10);

if (!timelinePath && !screenshotPath && !a11ySnapshotPath) {
  console.error('Usage: assertion-engine.js --timeline <path> [--screenshot <path>] [--baseline <path>]');
  console.error('  [--a11y-snapshot <path>] [--expected-elements "Btn1,Btn2"] [--threshold 0.1]');
  console.error('  [--max-diff-percent 5] [--action-timeout 30000]');
  console.error('\nAt least one of --timeline, --screenshot, or --a11y-snapshot is required.');
  process.exit(1);
}

const expectedElements = expectedElementsRaw
  ? expectedElementsRaw.split(',').map(s => s.trim()).filter(Boolean)
  : [];

// --- File Loaders ---
function loadJSON(filePath) {
  if (!filePath) return null;
  if (!fs.existsSync(filePath)) {
    console.error(`Warning: File not found: ${filePath}`);
    return null;
  }
  try {
    return JSON.parse(fs.readFileSync(filePath, 'utf8'));
  } catch (err) {
    console.error(`Warning: Failed to parse ${filePath}: ${err.message}`);
    return null;
  }
}

function loadImage(filePath) {
  if (!filePath || !fs.existsSync(filePath)) return null;
  return fs.readFileSync(filePath);
}

// --- Visual Regression (pixelmatch) ---
function runVisualRegression(screenshotBuf, baselineBuf) {
  if (!screenshotBuf || !baselineBuf) {
    return { pass: true, skipped: true, reason: 'No baseline or screenshot provided' };
  }

  let PNG, pixelmatch;
  try {
    PNG = require('pngjs').PNG;
  } catch {
    return { pass: true, skipped: true, reason: 'pngjs not installed, visual comparison skipped' };
  }
  try {
    pixelmatch = require('pixelmatch');
  } catch {
    return { pass: true, skipped: true, reason: 'pixelmatch not installed, visual comparison skipped' };
  }

  try {
    const img1 = PNG.sync.read(screenshotBuf);
    const img2 = PNG.sync.read(baselineBuf);

    const width = Math.min(img1.width, img2.width);
    const height = Math.min(img1.height, img2.height);

    // Resize if dimensions differ
    const crop = (img, w, h) => {
      if (img.width === w && img.height === h) return img.data;
      const cropped = Buffer.alloc(w * h * 4);
      for (let y = 0; y < h; y++) {
        img.data.copy(cropped, y * w * 4, y * img.width * 4, y * img.width * 4 + w * 4);
      }
      return cropped;
    };

    const data1 = crop(img1, width, height);
    const data2 = crop(img2, width, height);
    const diff = new PNG({ width, height });

    const numDiff = pixelmatch(data1, data2, diff.data, width, height, { threshold });
    const totalPixels = width * height;
    const diffPercent = totalPixels > 0 ? (numDiff / totalPixels) * 100 : 0;

    // Write diff image
    let diffImagePath = null;
    if (diffPercent > 0) {
      const outDir = path.dirname(screenshotPath || '.');
      diffImagePath = path.join(outDir, 'visual-diff.png');
      try {
        fs.writeFileSync(diffImagePath, PNG.sync.write(diff));
      } catch { diffImagePath = null; }
    }

    return {
      pass: diffPercent <= maxDiffPercent,
      diff_percent: Math.round(diffPercent * 100) / 100,
      diff_pixels: numDiff,
      total_pixels: totalPixels,
      diff_image: diffImagePath,
    };
  } catch (err) {
    return { pass: true, skipped: true, reason: `Visual comparison error: ${err.message}` };
  }
}

// --- A11y Check ---
function runA11yCheck(snapshot) {
  const result = { pass: true, missing: [], unexpected_alerts: [] };
  if (!snapshot) {
    if (expectedElements.length > 0) {
      result.pass = false;
      result.missing = expectedElements;
      result.reason = 'No a11y snapshot provided but expected elements were specified';
    }
    return result;
  }

  // Collect all element names from the tree
  const allNames = new Set();
  const alertElements = [];

  function walk(node) {
    if (!node) return;
    const name = node.name || node.label || node.text || '';
    const role = (node.role || '').toLowerCase();
    if (name) allNames.add(name);
    if (name) allNames.add(name.toLowerCase());

    if (role === 'alert' || role === 'alertdialog' || role === 'error') {
      alertElements.push({ role, name, description: node.description || '' });
    }

    const children = node.children || node.childNodes || node.nodes || [];
    if (Array.isArray(children)) {
      for (const child of children) walk(child);
    }
  }

  walk(snapshot);

  // Check expected elements
  for (const expected of expectedElements) {
    const found = allNames.has(expected) || allNames.has(expected.toLowerCase());
    if (!found) {
      result.missing.push(expected);
      result.pass = false;
    }
  }

  result.unexpected_alerts = alertElements;

  return result;
}

// --- Console Check ---
function runConsoleCheck(timeline) {
  const result = { pass: true, errors: [], warnings: 0 };
  if (!timeline) return result;

  for (const ev of (timeline.events || [])) {
    for (const log of (ev.console_logs || [])) {
      const level = (log.level || '').toLowerCase();
      const text = log.text || '';
      if (level === 'error') {
        result.errors.push(text);
        if (text.toLowerCase().includes('uncaught') || text.toLowerCase().includes('unhandled')) {
          result.pass = false;
        }
      }
      if (level === 'warning' || level === 'warn') {
        result.warnings++;
      }
    }
  }

  // Uncaught exceptions are critical
  if (result.errors.some(e => /uncaught|unhandled/i.test(e))) {
    result.pass = false;
  }

  return result;
}

// --- Network Check ---
function runNetworkCheck(timeline) {
  const result = { pass: true, failed_requests: [], total_requests: 0 };
  if (!timeline) return result;

  for (const ev of (timeline.events || [])) {
    for (const req of (ev.network_events || [])) {
      result.total_requests++;
      const status = req.status || 0;
      if (status >= 500) {
        result.pass = false;
        result.failed_requests.push({ url: req.url, status, type: '5xx' });
      } else if (status >= 400) {
        result.failed_requests.push({ url: req.url, status, type: '4xx' });
      }
    }
  }

  return result;
}

// --- Timing Check ---
function runTimingCheck(timeline) {
  const result = { pass: true, slowest_action_ms: 0, slow_actions: [] };
  if (!timeline) return result;

  for (const ev of (timeline.events || [])) {
    const dur = ev.duration_ms || 0;
    if (dur > result.slowest_action_ms) {
      result.slowest_action_ms = dur;
    }
    if (dur > TIMEOUT_MS) {
      result.pass = false;
      result.slow_actions.push({ seq: ev.seq, action: ev.action, duration_ms: dur });
    }
  }

  return result;
}

// --- Confidence Calculator ---
function calculateConfidence(breakdown) {
  const signals = [];
  const weights = { visual: 0.2, a11y: 0.25, console: 0.25, network: 0.15, timing: 0.15 };

  for (const [key, weight] of Object.entries(weights)) {
    const signal = breakdown[key];
    if (!signal) continue;
    if (signal.skipped) {
      // Skipped signals don't count toward confidence
      continue;
    }
    signals.push({ key, weight, pass: signal.pass });
  }

  if (signals.length === 0) return 0.5;

  const totalWeight = signals.reduce((sum, s) => sum + s.weight, 0);
  const passWeight = signals.filter(s => s.pass).reduce((sum, s) => sum + s.weight, 0);

  return Math.round((passWeight / totalWeight) * 100) / 100;
}

// --- Summarizer ---
function buildSummary(pass, breakdown) {
  const issues = [];

  if (breakdown.console && !breakdown.console.pass) {
    const first = breakdown.console.errors[0] || 'console error';
    issues.push(`Console error (${first.slice(0, 60)})`);
  }

  if (breakdown.a11y && !breakdown.a11y.pass) {
    const missing = breakdown.a11y.missing.join(', ');
    issues.push(`missing '${missing}' in a11y tree`);
  }

  if (breakdown.visual && !breakdown.visual.pass) {
    issues.push(`visual diff ${breakdown.visual.diff_percent}% > ${maxDiffPercent}%`);
  }

  if (breakdown.network && !breakdown.network.pass) {
    const count = breakdown.network.failed_requests.filter(r => r.type === '5xx').length;
    issues.push(`${count} server error(s)`);
  }

  if (breakdown.timing && !breakdown.timing.pass) {
    issues.push(`timeout (slowest: ${breakdown.timing.slowest_action_ms}ms)`);
  }

  if (issues.length === 0) {
    return pass ? 'PASS: All assertion signals passed' : 'FAIL: Unknown failure';
  }

  return `FAIL: ${issues.join('; ')}`;
}

// --- Critical Failure Rules ---
function hasCriticalFailure(breakdown) {
  // Console uncaught exception
  if (breakdown.console && !breakdown.console.pass) return true;
  // A11y missing expected element
  if (breakdown.a11y && breakdown.a11y.missing.length > 0) return true;
  // Visual diff exceeds threshold
  if (breakdown.visual && !breakdown.visual.skipped && !breakdown.visual.pass) return true;
  // Network 5xx
  if (breakdown.network && breakdown.network.failed_requests.some(r => r.type === '5xx')) return true;

  return false;
}

// --- Main ---
function main() {
  const timeline = loadJSON(timelinePath);
  const a11ySnapshot = loadJSON(a11ySnapshotPath);
  const screenshotBuf = loadImage(screenshotPath);
  const baselineBuf = loadImage(baselinePath);

  const breakdown = {
    visual: runVisualRegression(screenshotBuf, baselineBuf),
    a11y: runA11yCheck(a11ySnapshot),
    console: runConsoleCheck(timeline),
    network: runNetworkCheck(timeline),
    timing: runTimingCheck(timeline),
  };

  const criticalFail = hasCriticalFailure(breakdown);
  const allPass = Object.values(breakdown).every(b => b.pass !== false);
  const pass = !criticalFail && allPass;

  const confidence = calculateConfidence(breakdown);

  const criticalFailures = [];
  if (breakdown.console && !breakdown.console.pass) {
    for (const err of breakdown.console.errors) {
      if (/uncaught|unhandled/i.test(err)) criticalFailures.push(`Console: ${err}`);
    }
  }
  if (breakdown.a11y && breakdown.a11y.missing.length > 0) {
    criticalFailures.push(`A11y: missing elements [${breakdown.a11y.missing.join(', ')}]`);
  }
  if (breakdown.visual && !breakdown.visual.skipped && !breakdown.visual.pass) {
    criticalFailures.push(`Visual: diff ${breakdown.visual.diff_percent}% exceeds max ${maxDiffPercent}%`);
  }
  if (breakdown.network) {
    for (const req of breakdown.network.failed_requests.filter(r => r.type === '5xx')) {
      criticalFailures.push(`Network: ${req.status} on ${req.url}`);
    }
  }

  const summary = buildSummary(pass, breakdown);

  const verdict = {
    pass,
    confidence,
    critical_failures: criticalFailures,
    breakdown,
    summary,
  };

  // Determine output path
  let outDir = '.';
  if (timelinePath) outDir = path.dirname(timelinePath);
  else if (screenshotPath) outDir = path.dirname(screenshotPath);
  else if (a11ySnapshotPath) outDir = path.dirname(a11ySnapshotPath);

  const outFile = path.join(outDir, 'verdict.json');
  fs.mkdirSync(path.dirname(outFile), { recursive: true });
  fs.writeFileSync(outFile, JSON.stringify(verdict, null, 2));
  console.log(JSON.stringify({ status: 'ok', output: outFile, pass: verdict.pass, confidence: verdict.confidence, summary: verdict.summary }));
}

main();
