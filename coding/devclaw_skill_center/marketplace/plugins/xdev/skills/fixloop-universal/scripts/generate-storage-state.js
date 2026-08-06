#!/usr/bin/env node
// generate-storage-state.js - Generate Playwright-compatible storage-state.json
// Usage:
//   node generate-storage-state.js --strategy none
//   node generate-storage-state.js --strategy cookie_inject --cookies '[...]'
//   node generate-storage-state.js --strategy local_storage --items '[...]' --origin http://localhost:8080
//   node generate-storage-state.js --strategy form_login
//   Options: --out <path> (default: ./fixloop-output/.storage-state.json)

const fs = require('fs');
const path = require('path');

const args = process.argv.slice(2);

function getArg(name, defaultValue) {
  const idx = args.indexOf(`--${name}`);
  if (idx === -1 || idx + 1 >= args.length) return defaultValue;
  return args[idx + 1];
}

// --- Parse CLI args ---
const strategy = getArg('strategy', 'none');
const outFile = getArg('out', './fixloop-output/.storage-state.json');

// --- Build storage state ---
const storageState = {
  cookies: [],
  origins: []
};

switch (strategy) {
  case 'none':
  case 'form_login':
    // Empty storage state — form_login is handled at runtime by the test harness
    break;

  case 'cookie_inject': {
    const cookiesRaw = getArg('cookies', null);
    if (!cookiesRaw) {
      console.error('Error: --cookies is required for cookie_inject strategy');
      console.error('  Format: --cookies \'[{"name":"x","value":"y","domain":"d","path":"/"}]\'');
      process.exit(1);
    }

    let cookies;
    try {
      cookies = JSON.parse(cookiesRaw);
    } catch (e) {
      console.error(`Error: Invalid JSON for --cookies: ${e.message}`);
      process.exit(1);
    }

    if (!Array.isArray(cookies)) {
      console.error('Error: --cookies must be a JSON array');
      process.exit(1);
    }

    // Normalize cookies to Playwright format
    storageState.cookies = cookies.map((c) => ({
      name: c.name,
      value: c.value,
      domain: c.domain || 'localhost',
      path: c.path || '/',
      expires: c.expires || -1,
      httpOnly: c.httpOnly || false,
      secure: c.secure || false,
      sameSite: c.sameSite || 'Lax'
    }));
    break;
  }

  case 'local_storage': {
    const itemsRaw = getArg('items', null);
    const origin = getArg('origin', null);

    if (!itemsRaw) {
      console.error('Error: --items is required for local_storage strategy');
      console.error('  Format: --items \'[{"key":"x","value":"y"}]\'');
      process.exit(1);
    }
    if (!origin) {
      console.error('Error: --origin is required for local_storage strategy');
      console.error('  Example: --origin http://localhost:8080');
      process.exit(1);
    }

    let items;
    try {
      items = JSON.parse(itemsRaw);
    } catch (e) {
      console.error(`Error: Invalid JSON for --items: ${e.message}`);
      process.exit(1);
    }

    if (!Array.isArray(items)) {
      console.error('Error: --items must be a JSON array');
      process.exit(1);
    }

    storageState.origins = [{
      origin: origin,
      localStorage: items.map((item) => ({
        name: item.key || item.name,
        value: item.value
      }))
    }];
    break;
  }

  case 'custom':
    // Custom auth is handled at runtime by the test harness — empty storage state
    break;

  default:
    console.warn(`Warning: Unknown strategy "${strategy}". Generating empty storage state.`);
    break;
}

// --- Write output ---
const output = JSON.stringify(storageState, null, 2);

fs.mkdirSync(path.dirname(outFile), { recursive: true });
fs.writeFileSync(outFile, output + '\n', 'utf8');
console.log(`Storage state written to ${outFile} (strategy: ${strategy})`);
