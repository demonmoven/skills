#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const root = path.resolve(__dirname, "..");
const pkg = require("../package.json");
const registry = process.env.BNPM_REGISTRY || "https://bnpm.byted.org";
const version = pkg.version;
const tag = `v${version}`;
const distDir = path.join(root, "dist");
const rootChecksums = path.join(root, "checksums.txt");
const distChecksums = path.join(distDir, "checksums.txt");

const args = new Set(process.argv.slice(2));
const dryRun = args.has("--dry-run");
const allowUntagged = args.has("--allow-untagged");
const snapshotMode = dryRun || allowUntagged;
const skipPublish = snapshotMode || args.has("--skip-publish");
const skipBinaryCheck = args.has("--skip-binary-check");

function run(command, commandArgs, options = {}) {
  const result = spawnSync(command, commandArgs, {
    cwd: root,
    env: process.env,
    stdio: "inherit",
    ...options,
  });
  if (result.status !== 0) {
    process.exit(result.status || 1);
  }
}

function read(command, commandArgs) {
  const result = spawnSync(command, commandArgs, {
    cwd: root,
    env: process.env,
    stdio: ["ignore", "pipe", "pipe"],
    encoding: "utf8",
  });
  return {
    ok: result.status === 0,
    stdout: result.stdout.trim(),
    stderr: result.stderr.trim(),
  };
}

function runChecked(command, commandArgs) {
  const result = spawnSync(command, commandArgs, {
    cwd: root,
    env: process.env,
    stdio: ["ignore", "pipe", "pipe"],
    encoding: "utf8",
  });
  return {
    ok: result.status === 0,
    stdout: result.stdout.trim(),
    stderr: result.stderr.trim(),
    status: result.status,
  };
}

function requireCommand(command, hint) {
  const result = read(process.platform === "win32" ? "where" : "which", [command]);
  if (!result.ok || !result.stdout) {
    console.error(`Missing required command: ${command}`);
    if (hint) console.error(hint);
    process.exit(1);
  }
}

function ensureTaggedRelease() {
  const currentTag = read("git", ["describe", "--tags", "--exact-match", "HEAD"]);
  if (currentTag.ok && currentTag.stdout === tag) {
    return;
  }

  if (allowUntagged || dryRun) {
    console.warn(`[WARN] HEAD is not tagged ${tag}; continuing because this is an untagged/dry run.`);
    return;
  }

  console.error(`HEAD must be tagged ${tag} before release.`);
  console.error(`Create and push the tag first, or rerun with --allow-untagged for local packaging only.`);
  process.exit(1);
}

function ensureNpmAuth() {
  if (skipPublish) return;
  const result = read("npm", ["whoami", "--registry", registry]);
  if (!result.ok) {
    console.error(`Not logged in to bnpm registry: ${registry}`);
    console.error("Run: npx @bytedance-dev/bnpm@latest login --auth-type=sso");
    process.exit(1);
  }
}

function buildReleaseArtifacts() {
  const releaseArgs = ["release", "--clean", "--skip=publish"];
  if (snapshotMode) releaseArgs.push("--snapshot");
  run("goreleaser", releaseArgs);
}

function syncChecksums() {
  if (!fs.existsSync(distChecksums)) {
    console.error(`Missing ${distChecksums}; GoReleaser did not generate checksums.`);
    process.exit(1);
  }
  fs.copyFileSync(distChecksums, rootChecksums);
}

function readChecksums() {
  const entries = new Map();
  const content = fs.readFileSync(rootChecksums, "utf8");
  for (const line of content.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    const idx = trimmed.indexOf("  ");
    if (idx === -1) continue;
    entries.set(trimmed.slice(idx + 2), trimmed.slice(0, idx));
  }
  return entries;
}

function listArchives() {
  if (!fs.existsSync(distDir)) return [];
  return fs.readdirSync(distDir)
    .filter((name) => /^gloop-.+\.(tar\.gz|zip)$/.test(name))
    .sort();
}

function expectedArchives() {
  const platforms = {
    darwin: "darwin",
    linux: "linux",
    win32: "windows",
  };
  const archs = {
    x64: "amd64",
    arm64: "arm64",
  };

  const archives = [];
  for (const osName of pkg.os || []) {
    const platform = platforms[osName];
    if (!platform) continue;
    for (const cpuName of pkg.cpu || []) {
      const arch = archs[cpuName];
      if (!arch) continue;
      const ext = osName === "win32" ? ".zip" : ".tar.gz";
      archives.push(`gloop-${version}-${platform}-${arch}${ext}`);
    }
  }
  return archives.sort();
}

function ensureArchivesMatchNpmVersion(archives) {
  const expected = expectedArchives();
  const actual = new Set(archives);
  const missing = expected.filter((archive) => !actual.has(archive));
  if (missing.length) {
    console.error("Release archives do not match package version expectations:");
    for (const archive of missing) {
      console.error(`  missing ${archive}`);
    }
    console.error("\nRefusing to publish npm because install URLs/checksums would not line up.");
    process.exit(1);
  }
}

function ensureChecksumsCoverArchives(archives) {
  const checksums = readChecksums();
  const missing = archives.filter((archive) => !checksums.has(archive));
  if (missing.length) {
    console.error("checksums.txt is missing entries for:");
    for (const archive of missing) {
      console.error(`  ${archive}`);
    }
    process.exit(1);
  }
}

function binaryUrl(archive) {
  return `${registry.replace(/\/+$/, "")}/-/binary/gloop/v${version}/${archive}`;
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

  const head = runChecked("curl", [...commonArgs, "--head", url]);
  if (head.ok) return true;

  // Some object stores do not support HEAD. Fetch one byte as a fallback.
  const get = runChecked("curl", [...commonArgs, "--range", "0-0", "--output", osNullPath(), url]);
  return get.ok;
}

function osNullPath() {
  return process.platform === "win32" ? "NUL" : "/dev/null";
}

function ensureRemoteBinariesUploaded(archives) {
  const missing = [];
  for (const archive of archives) {
    const url = binaryUrl(archive);
    if (!remoteBinaryExists(url)) {
      missing.push(url);
    }
  }

  if (!missing.length) return;

  console.error("\nRemote binary artifacts are missing or unreachable:");
  for (const url of missing) {
    console.error(`  ${url}`);
  }
  console.error("\nUpload these files before publishing npm, then rerun with --skip-binary-check.");
  process.exit(1);
}

function hasBundledArchives(archives) {
  return archives.every((archive) => fs.existsSync(path.join(distDir, archive)));
}

function printBinaryUploadInstructions(archives) {
  console.log("\nBinary artifacts are ready:");
  for (const archive of archives) {
    console.log(`  dist/${archive}`);
  }

  console.log("\nUpload these files to the bnpm binary path before publishing npm:");
  for (const archive of archives) {
    console.log(`  ${binaryUrl(archive)}`);
  }

  console.log("\nIf your binary files are hosted elsewhere, install users can set:");
  console.log("  GLOOP_BINARY_BASE_URL=https://your-host/gloop");
}

function printSnapshotArtifacts(archives) {
  console.log("\nSnapshot artifacts are ready:");
  for (const archive of archives) {
    console.log(`  dist/${archive}`);
  }
  console.log("\nThese are snapshot-only artifacts and must NOT be uploaded for npm install.");
  console.log("Create the release tag and rerun without --dry-run to generate publishable archive names.");
}

function publishNpm() {
  if (skipPublish) {
    run("npm", ["pack", "--dry-run", "--ignore-scripts"]);
    return;
  }
  run("npm", ["publish", "--registry", registry]);
}

function main() {
  requireCommand("npm", "Install Node.js/npm first.");
  requireCommand("git", "Install git first.");
  requireCommand("go", "Install Go before building gloop native binaries.");
  requireCommand("goreleaser", "Install GoReleaser: https://goreleaser.com/install/");
  requireCommand("curl", "Install curl before verifying uploaded binary artifacts.");

  ensureTaggedRelease();
  ensureNpmAuth();

  run("node", ["--check", "scripts/npm-install.js"]);
  run("node", ["--check", "scripts/npm-run-bin.js"]);
  buildReleaseArtifacts();

  const archives = listArchives();
  if (!archives.length) {
    console.error("No gloop release archives were generated under dist/.");
    process.exit(1);
  }

  if (snapshotMode) {
    printSnapshotArtifacts(archives);
    publishNpm();
    return;
  }

  syncChecksums();
  ensureArchivesMatchNpmVersion(archives);
  ensureChecksumsCoverArchives(archives);
  if (hasBundledArchives(expectedArchives())) {
    console.log("\nLocal binary artifacts are bundled in the npm package; publishing npm directly.");
    publishNpm();
    return;
  }

  if (!skipBinaryCheck) {
    printBinaryUploadInstructions(archives);
    console.log("\nRelease paused before npm publish.");
    console.log("Upload the binary artifacts, then rerun with --skip-binary-check to publish npm.");
    console.log("For a dry run, use: npm run release:bnpm -- --dry-run");
    process.exit(0);
  }

  ensureRemoteBinariesUploaded(archives);
  publishNpm();
}

main();
