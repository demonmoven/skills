#!/usr/bin/env node
// mcp-config.js - Generate .mcp.json for Playwright MCP from CLI args
// Usage: node mcp-config.js --mode <mode> --viewport 1440x900 --output-dir ./fixloop-output/traces [options]

const args = process.argv.slice(2);

function getArg(name, defaultValue) {
  const idx = args.indexOf(`--${name}`);
  if (idx === -1 || idx + 1 >= args.length) return defaultValue;
  return args[idx + 1];
}

function hasFlag(name) {
  return args.includes(`--${name}`);
}

// --- Parse CLI args ---
const mode = getArg('mode', 'bundled');
const viewport = getArg('viewport', '1440x900');
const outputDir = getArg('output-dir', './fixloop-output/traces');
const storageState = getArg('storage-state', null);
const cdpEndpoint = getArg('cdp-endpoint', null);
const outFile = getArg('out', null);

const [vpWidth, vpHeight] = viewport.split('x').map(Number);

// --- Build playwright MCP args ---
const mcpArgs = ['--caps', 'vision'];
mcpArgs.push('--save-session');
mcpArgs.push('--output-dir', outputDir);
mcpArgs.push('--output-mode', 'file');
mcpArgs.push('--viewport-size', `${vpWidth},${vpHeight}`);

if (storageState) {
  mcpArgs.push('--storage-state', storageState);
}

switch (mode) {
  case 'system_chrome':
    mcpArgs.push('--browser', 'chrome');
    break;
  case 'cdp_connect':
    if (!cdpEndpoint) {
      console.error('Error: --cdp-endpoint is required for cdp_connect mode');
      process.exit(1);
    }
    mcpArgs.push('--cdp-endpoint', cdpEndpoint);
    break;
  case 'headless':
    mcpArgs.push('--headless');
    break;
  case 'bundled':
    // Default chromium, no extra flags needed
    break;
  default:
    console.error(`Error: Unknown browser mode "${mode}". Use: system_chrome, cdp_connect, bundled, headless`);
    process.exit(1);
}

// --- Build .mcp.json ---
const mcpConfig = {
  mcpServers: {
    playwright: {
      command: 'npx',
      args: ['@playwright/mcp', ...mcpArgs]
    },
    'trace-analyzer': {
      command: 'npx',
      args: ['playwright-trace-analyzer-mcp']
    }
  }
};

const output = JSON.stringify(mcpConfig, null, 2);

if (outFile) {
  const fs = require('fs');
  const path = require('path');
  fs.mkdirSync(path.dirname(outFile), { recursive: true });
  fs.writeFileSync(outFile, output + '\n', 'utf8');
  console.log(`MCP config written to ${outFile}`);
} else {
  console.log(output);
}
