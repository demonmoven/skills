import esbuild from 'esbuild';
import { builtinModules } from 'module';
import { execSync } from 'child_process';
import fs from 'fs';
import path from 'path';

const pkg = JSON.parse(fs.readFileSync('./package.json', 'utf8'));
const deps = Object.keys(pkg.dependencies || {});

const sharedExternal = [
  ...builtinModules,
  ...builtinModules.map((m) => `node:${m}`),
  ...deps,
  'fsevents',
];

const banner = {
  js: [
    `import { fileURLToPath as __esm_fileURLToPath } from 'url';`,
    `import { dirname as __esm_dirname } from 'path';`,
    `const __filename = __esm_fileURLToPath(import.meta.url);`,
    `const __dirname = __esm_dirname(__filename);`,
  ].join('\n'),
};

// 1. Main CLI bundle (xdev)
await esbuild.build({
  entryPoints: ['src/index.ts'],
  bundle: true,
  format: 'esm',
  platform: 'node',
  target: 'node18',
  outfile: 'dist/index.js',
  banner,
  external: sharedExternal,
  define: {
    // Inject the package.json version as a compile-time constant so
    // src/index.ts can use it via `declare const XDEV_VERSION: string`.
    // Avoids the previous .version('0.1.0') hardcoded string drifting from
    // package.json on every release.
    XDEV_VERSION: JSON.stringify(pkg.version),
  },
});

const cliOut = 'dist/index.js';
const cliContent = fs.readFileSync(cliOut, 'utf8');
if (!cliContent.startsWith('#!/')) {
  fs.writeFileSync(cliOut, `#!/usr/bin/env node\n${cliContent}`);
}
fs.chmodSync(cliOut, '755');
console.log(path.resolve(cliOut));

// 2. OpenCode plugin entry (consumed via package.json#exports as ./trace/opencode)
//    OpenCode loads this at runtime by importing
//    "@byted/xdex/trace/opencode" so it must be a separate
//    bundle, not part of the CLI.
await esbuild.build({
  entryPoints: ['src/trace/plugins/opencode.ts'],
  bundle: true,
  format: 'esm',
  platform: 'node',
  target: 'node18',
  outfile: 'dist/trace/plugins/opencode.js',
  external: sharedExternal,
});
console.log(path.resolve('dist/trace/plugins/opencode.js'));

// 3. lark-cli MCP server entry — deployed standalone to ~/.xdev/lark-mcp-server/
//    by `xdev lark-mcp mac setup` and spawned as a child process by Claude
//    Desktop based on claude_desktop_config.json. Must be:
//
//      - fully self-contained (no node_modules dependency on the user's machine)
//      - ESM with shebang (so node can run it directly)
//      - external only to node builtins (NOT to xdev's deps, since the
//        deployed location won't have access to xdev's node_modules)
//
//    Currently the server uses zero npm deps, so the bundle ends up containing
//    only the source itself + a tiny ESM banner. If a future version adds
//    @modelcontextprotocol/sdk, esbuild will inline it from xdev's
//    node_modules/.
const mcpExternal = [
  ...builtinModules,
  ...builtinModules.map((m) => `node:${m}`),
];

await esbuild.build({
  entryPoints: ['src/lark-mcp/server.ts'],
  bundle: true,
  format: 'esm',
  platform: 'node',
  target: 'node18',
  outfile: 'dist/lark-mcp-server.mjs',
  banner,
  external: mcpExternal,
});

const mcpOut = 'dist/lark-mcp-server.mjs';
const mcpContent = fs.readFileSync(mcpOut, 'utf8');
if (!mcpContent.startsWith('#!/')) {
  fs.writeFileSync(mcpOut, `#!/usr/bin/env node\n${mcpContent}`);
}
fs.chmodSync(mcpOut, '755');
console.log(path.resolve(mcpOut));

// 4. Build the trace Web Dashboard frontend (Vite + React).
//    Skipped if web/node_modules is missing (developers who only need the CLI
//    can omit the web toolchain by not running `npm install` in web/).
if (fs.existsSync('web/node_modules')) {
  execSync('npm run build --prefix web', { stdio: 'inherit' });
  console.log(path.resolve('web/dist'));
} else {
  console.warn('[build] Skipping web/ Vite build (web/node_modules not found — run `cd web && npm install` first)');
}

try {
  execSync('npm link', { stdio: 'inherit' });
  console.log('✓ xdev command linked locally (npm package: @byted/xdex)');
} catch (e) {
  console.error('✗ npm link failed:', e.message);
  process.exit(1);
}
