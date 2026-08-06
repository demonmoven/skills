#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const root = path.resolve(__dirname, "..");
const pkg = require("../package.json");
const version = pkg.version;
const registry =
  process.env.BNPM_REGISTRY ||
  process.env.npm_config_registry ||
  (pkg.publishConfig && pkg.publishConfig.registry) ||
  "https://bnpm.byted.org";

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

function fail(message, details = []) {
  console.error(message);
  for (const detail of details) {
    console.error(`  ${detail}`);
  }
  process.exit(1);
}

function commandExists(command) {
  const lookup = process.platform === "win32" ? "where" : "which";
  const result = spawnSync(lookup, [command], { stdio: "ignore" });
  return result.status === 0;
}

function readChecksums() {
  const checksumsPath = path.join(root, "checksums.txt");
  if (!fs.existsSync(checksumsPath)) {
    fail("checksums.txt is missing. Run npm run release:bnpm before publishing npm.");
  }

  const entries = new Map();
  const content = fs.readFileSync(checksumsPath, "utf8");
  for (const line of content.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    const idx = trimmed.indexOf("  ");
    if (idx === -1) continue;
    entries.set(trimmed.slice(idx + 2), trimmed.slice(0, idx));
  }
  return entries;
}

function expectedArchives() {
  const archives = [];
  for (const osName of pkg.os || []) {
    const platform = PLATFORM_MAP[osName];
    if (!platform) continue;

    for (const cpuName of pkg.cpu || []) {
      const arch = ARCH_MAP[cpuName];
      if (!arch) continue;

      const ext = osName === "win32" ? ".zip" : ".tar.gz";
      archives.push(`gloop-${version}-${platform}-${arch}${ext}`);
    }
  }
  return archives.sort();
}

function ensureChecksums(archives) {
  const checksums = readChecksums();
  const missing = archives.filter((archive) => !checksums.has(archive));
  if (missing.length) {
    fail("checksums.txt is missing required archive entries:", missing);
  }
}

function localBinaryPath(archive) {
  return path.join(root, "dist", archive);
}

function ensureLocalBinaries(archives) {
  const missing = archives
    .map((archive) => localBinaryPath(archive))
    .filter((archivePath) => !fs.existsSync(archivePath));
  if (missing.length) {
    fail("Local binary artifacts are missing:", missing);
  }
}

function hasLocalBinaries(archives) {
  return archives.every((archive) => fs.existsSync(localBinaryPath(archive)));
}

function binaryUrl(archive) {
  return `${registry.replace(/\/+$/, "")}/-/binary/gloop/v${version}/${archive}`;
}

function curl(args) {
  return spawnSync("curl", args, {
    cwd: root,
    env: process.env,
    stdio: ["ignore", "pipe", "pipe"],
    encoding: "utf8",
  });
}

function remoteBinaryExists(url) {
  const commonArgs = [
    "--fail",
    "--location",
    "--silent",
    "--show-error",
    "--connect-timeout",
    "10",
    "--max-time",
    "30",
    "--max-redirs",
    "3",
  ];

  const head = curl([...commonArgs, "--head", url]);
  if (head.status === 0) return true;

  const nullPath = process.platform === "win32" ? "NUL" : "/dev/null";
  const get = curl([...commonArgs, "--range", "0-0", "--output", nullPath, url]);
  return get.status === 0;
}

function ensureRemoteBinaries(archives) {
  if (!commandExists("curl")) {
    fail("Missing required command: curl");
  }

  const missing = archives.map(binaryUrl).filter((url) => !remoteBinaryExists(url));
  if (missing.length) {
    fail("Remote binary artifacts are missing or unreachable:", missing);
  }
}

function main() {
  const archives = expectedArchives();
  if (!archives.length) {
    fail("No expected release archives could be derived from package.json os/cpu.");
  }
  ensureChecksums(archives);
  if (hasLocalBinaries(archives)) {
    ensureLocalBinaries(archives);
    console.log(`npm release guard passed for ${pkg.name}@${version} (local bundled binaries)`);
    return;
  }
  ensureRemoteBinaries(archives);
  console.log(`npm release guard passed for ${pkg.name}@${version}`);
}

main();
