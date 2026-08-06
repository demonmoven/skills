#!/usr/bin/env node
'use strict';

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');

// --- CLI Parsing ---
const args = process.argv.slice(2);
const snapshotPath = args.find(a => !a.startsWith('--'));

function getArg(name, defaultValue) {
  const idx = args.indexOf('--' + name);
  return idx !== -1 && args[idx + 1] ? args[idx + 1] : defaultValue;
}

const url = getArg('url', '');
const visitedPath = getArg('visited', null);
const budget = parseInt(getArg('budget', '10'), 10);
const strategy = getArg('strategy', 'a11y_driven');

if (!snapshotPath) {
  console.error('Usage: state-explorer.js <a11y-snapshot.json> --url URL [--visited visited-states.json] [--budget N] [--strategy a11y_driven|boundary|chaos]');
  process.exit(1);
}

if (!url) {
  console.error('Error: --url is required (the dev server port is dynamic, so the orchestrator must provide it)');
  process.exit(1);
}

if (!fs.existsSync(snapshotPath)) {
  console.error(`Error: Snapshot file not found: ${snapshotPath}`);
  process.exit(1);
}

const INTERACTIVE_ROLES = new Set([
  'button', 'link', 'textbox', 'searchbox', 'checkbox', 'radio',
  'combobox', 'menuitem', 'tab', 'slider', 'switch', 'spinbutton',
]);

// --- A11y Tree Walker ---
function extractInteractiveElements(node, results) {
  if (!node) return results;

  const role = (node.role || '').toLowerCase();
  if (INTERACTIVE_ROLES.has(role)) {
    results.push({
      ref: node.ref || node.id || null,
      role: role,
      name: node.name || node.label || node.text || '',
      description: node.description || node.value || '',
    });
  }

  const children = node.children || node.childNodes || node.nodes || [];
  if (Array.isArray(children)) {
    for (const child of children) {
      extractInteractiveElements(child, results);
    }
  }

  return results;
}

function computeStateHash(elements, pageUrl) {
  const pairs = elements
    .map(e => `${e.role}:${e.name}`)
    .sort();
  const pathname = extractPathname(pageUrl);
  const raw = pairs.join(',') + '||' + pathname;
  return crypto.createHash('sha256').update(raw).digest('hex').slice(0, 16);
}

function extractPathname(urlStr) {
  try {
    return new URL(urlStr).pathname;
  } catch {
    return urlStr;
  }
}

// --- Visited States ---
function loadVisited(filePath) {
  if (!filePath) return { states: {}, transitions: {} };
  try {
    if (fs.existsSync(filePath)) {
      return JSON.parse(fs.readFileSync(filePath, 'utf8'));
    }
  } catch { /* corrupted file, start fresh */ }
  return { states: {}, transitions: {} };
}

function saveVisited(filePath, visited) {
  if (!filePath) return;
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, JSON.stringify(visited, null, 2));
}

// --- Strategy: a11y_driven ---
function planA11yDriven(elements, stateHash, visited) {
  const plan = [];
  const transitions = visited.transitions || {};

  for (const el of elements) {
    const transitionKey = `${stateHash}:${el.ref || el.role + ':' + el.name}`;
    const wasTested = transitions[transitionKey];

    const action = inferAction(el.role);
    plan.push({
      action,
      element: { role: el.role, name: el.name, ref: el.ref },
      reason: wasTested ? `Re-test (last: ${wasTested.last_tested})` : 'Untested element in current state',
      priority: wasTested ? 10 : 1,
      _transition_key: transitionKey,
    });
  }

  plan.sort((a, b) => a.priority - b.priority);
  return plan;
}

function inferAction(role) {
  switch (role) {
    case 'textbox':
    case 'searchbox':
    case 'spinbutton':
      return 'fill';
    case 'checkbox':
    case 'radio':
    case 'switch':
      return 'toggle';
    case 'combobox':
      return 'select';
    case 'slider':
      return 'drag';
    case 'tab':
    case 'menuitem':
    case 'button':
    case 'link':
    default:
      return 'click';
  }
}

// --- Strategy: boundary ---
function planBoundary(elements) {
  const plan = [];
  let priority = 1;

  for (const el of elements) {
    const role = el.role;

    if (['textbox', 'searchbox', 'spinbutton'].includes(role)) {
      plan.push({
        action: 'fill',
        element: { role: el.role, name: el.name, ref: el.ref },
        value: '',
        reason: 'Boundary: empty input',
        priority: priority++,
      });
      plan.push({
        action: 'fill',
        element: { role: el.role, name: el.name, ref: el.ref },
        value: 'x'.repeat(1000),
        reason: 'Boundary: very long input (1000 chars)',
        priority: priority++,
      });
      plan.push({
        action: 'fill',
        element: { role: el.role, name: el.name, ref: el.ref },
        value: '<script>alert(1)</script>',
        reason: 'Boundary: XSS payload',
        priority: priority++,
      });
    } else if (['button', 'link', 'tab', 'menuitem'].includes(role)) {
      plan.push({
        action: 'double_click',
        element: { role: el.role, name: el.name, ref: el.ref },
        reason: 'Boundary: rapid double-click',
        priority: priority++,
      });
    } else if (['checkbox', 'radio', 'switch'].includes(role)) {
      plan.push({
        action: 'rapid_toggle',
        element: { role: el.role, name: el.name, ref: el.ref },
        count: 5,
        reason: 'Boundary: rapid toggle 5x',
        priority: priority++,
      });
    }
  }

  return plan;
}

// --- Strategy: chaos ---
function planChaos() {
  const gremlinsPath = require.resolve('gremlins.js/dist/gremlins.min.js');
  // The explorer agent should read this file and inject via browser_console_execute
  return [{
    action: 'inject_gremlins',
    element: null,
    reason: 'Chaos: inject gremlins.js for random interaction fuzzing',
    priority: 1,
    gremlins_local_path: gremlinsPath,
    instructions: 'Read the local gremlins.min.js file and inject its contents via browser_console_execute, then call gremlins.createHorde().unleash(). Observe for crashes, unhandled errors, and layout breakage for 30 seconds.',
  }];
}

// --- Main ---
function main() {
  let snapshot;
  try {
    snapshot = JSON.parse(fs.readFileSync(snapshotPath, 'utf8'));
  } catch (err) {
    console.error(`Error: Failed to parse snapshot: ${err.message}`);
    process.exit(1);
  }

  const elements = extractInteractiveElements(snapshot, []);
  const stateHash = computeStateHash(elements, url);
  const visited = loadVisited(visitedPath);

  // Record this state visit
  if (!visited.states[stateHash]) {
    visited.states[stateHash] = { first_seen: new Date().toISOString(), url, element_count: elements.length };
  }
  visited.states[stateHash].last_seen = new Date().toISOString();

  let rawPlan;
  switch (strategy) {
    case 'boundary':
      rawPlan = planBoundary(elements);
      break;
    case 'chaos':
      rawPlan = planChaos();
      break;
    case 'a11y_driven':
    default:
      rawPlan = planA11yDriven(elements, stateHash, visited);
      break;
  }

  const plan = rawPlan.slice(0, budget);
  const untestedCount = strategy === 'a11y_driven'
    ? plan.filter(p => p.priority === 1).length
    : elements.length;

  // Mark planned transitions as pending in visited
  if (strategy === 'a11y_driven') {
    for (const item of plan) {
      if (item._transition_key && !visited.transitions[item._transition_key]) {
        visited.transitions[item._transition_key] = { status: 'pending', planned_at: new Date().toISOString() };
      }
    }
  }

  // Clean internal keys from output
  const cleanPlan = plan.map(p => {
    const { _transition_key, ...rest } = p;
    return rest;
  });

  const output = {
    state_hash: stateHash,
    url,
    total_interactive: elements.length,
    untested_count: untestedCount,
    strategy,
    plan: cleanPlan,
  };

  saveVisited(visitedPath, visited);

  const outDir = path.dirname(snapshotPath);
  const outFile = path.join(outDir, 'exploration-plan.json');
  fs.writeFileSync(outFile, JSON.stringify(output, null, 2));
  console.log(JSON.stringify({ status: 'ok', output: outFile, state_hash: stateHash, total: elements.length, planned: cleanPlan.length }));
}

main();
