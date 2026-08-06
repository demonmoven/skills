#!/usr/bin/env node

const crypto = require("node:crypto");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { execFileSync } = require("node:child_process");

const pkg = require("../package.json");
const VERSION = pkg.version;
const NAME = "gloop";
const DEFAULT_REGISTRY = "https://bnpm.byted.org";

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

const platform = PLATFORM_MAP[process.platform];
const arch = ARCH_MAP[process.arch];
const isWindows = process.platform === "win32";
const archiveExt = isWindows ? ".zip" : ".tar.gz";
const archiveName = `${NAME}-${VERSION}-${platform}-${arch}${archiveExt}`;

function getShimDir() {
  const home = os.homedir();
  return path.join(home, `.${NAME}`, "bin");
}

function getBinDir() {
  return path.join(getShimDir(), "versions", `v${VERSION}`);
}

function getVersionMarkerPath(binDir) {
  return path.join(binDir, `.${NAME}-version`);
}

function getNativeBinaryPath(binDir, name = NAME) {
  return path.join(binDir, name + (isWindows ? ".exe" : ""));
}

function isInstalled(binDir) {
  try {
    if (fs.readFileSync(getVersionMarkerPath(binDir), "utf8").trim() !== VERSION) {
      return false;
    }
    return fs.existsSync(getNativeBinaryPath(binDir, "gloop")) && fs.existsSync(getNativeBinaryPath(binDir, "gloopd"));
  } catch {
    return false;
  }
}

function writeVersionMarker(binDir, version) {
  fs.writeFileSync(getVersionMarkerPath(binDir), version + "\n", "utf8");
}

const binDir = getBinDir();

const ALLOWED_HOSTS = new Set([
  "bnpm.byted.org",
  "bnpm.bytedance.net",
  "sf-unpkg-src.bytedance.net",
  "unpkg.bytedance.net",
]);

function joinUrl(base, suffix) {
  return base.replace(/\/+$/, "") + suffix;
}

function isHttpsUrl(raw) {
  try {
    const parsed = new URL(raw);
    return parsed.protocol === "https:" && !!parsed.hostname;
  } catch (_) {
    return false;
  }
}

function addAllowedHost(raw) {
  ALLOWED_HOSTS.add(new URL(raw).hostname);
}

function resolveDownloadUrls(env) {
  const urls = [];
  const explicitBase = (env.GLOOP_BINARY_BASE_URL || "").trim();
  if (explicitBase && isHttpsUrl(explicitBase)) {
    addAllowedHost(explicitBase);
    urls.push(joinUrl(explicitBase, `/v${VERSION}/${archiveName}`));
  }

  const binaryPath = `/-/binary/gloop/v${VERSION}/${archiveName}`;
  const registry = (env.npm_config_registry || DEFAULT_REGISTRY).trim();
  if (registry && isHttpsUrl(registry)) {
    addAllowedHost(registry);
    urls.push(joinUrl(registry, binaryPath));
  }

  const fallback = joinUrl(DEFAULT_REGISTRY, binaryPath);
  if (!urls.includes(fallback)) {
    urls.push(fallback);
  }
  return urls;
}

function assertAllowedHost(url) {
  const { hostname } = new URL(url);
  if (!ALLOWED_HOSTS.has(hostname)) {
    throw new Error(`Download host not allowed: ${hostname}`);
  }
}

function download(url, destPath) {
  assertAllowedHost(url);
  execFileSync(
    "curl",
    [
      "--fail",
      "--location",
      "--silent",
      "--show-error",
      "--connect-timeout",
      "10",
      "--max-time",
      "120",
      "--max-redirs",
      "3",
      "--output",
      destPath,
      url,
    ],
    { stdio: ["ignore", "ignore", "pipe"] }
  );
}

function bundledArchivePath() {
  const archivePath = path.join(__dirname, "..", "dist", archiveName);
  return fs.existsSync(archivePath) ? archivePath : "";
}

function getExpectedChecksum(name) {
  const checksumsPath = path.join(__dirname, "..", "checksums.txt");
  if (!fs.existsSync(checksumsPath)) {
    throw new Error("checksums.txt not found; refusing to install without checksum verification");
  }

  const content = fs.readFileSync(checksumsPath, "utf8");
  for (const line of content.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    const idx = trimmed.indexOf("  ");
    if (idx === -1) continue;
    const hash = trimmed.slice(0, idx);
    const filename = trimmed.slice(idx + 2);
    if (filename === name) return hash;
  }
  throw new Error(`Checksum entry not found for ${name}`);
}

function verifyChecksum(archivePath, expectedHash) {
  if (expectedHash === null) return;
  const hash = crypto.createHash("sha256");
  const fd = fs.openSync(archivePath, "r");
  try {
    const buf = Buffer.alloc(64 * 1024);
    let bytesRead;
    while ((bytesRead = fs.readSync(fd, buf, 0, buf.length, null)) > 0) {
      hash.update(buf.subarray(0, bytesRead));
    }
  } finally {
    fs.closeSync(fd);
  }

  const actual = hash.digest("hex");
  if (actual.toLowerCase() !== expectedHash.toLowerCase()) {
    throw new Error(
      `[SECURITY] Checksum mismatch for ${path.basename(archivePath)}: expected ${expectedHash} but got ${actual}`
    );
  }
}

function extractArchive(archivePath, destDir) {
  if (isWindows) {
    const command =
      "$ErrorActionPreference='Stop';" +
      "Expand-Archive -LiteralPath $env:GLOOP_ARCHIVE -DestinationPath $env:GLOOP_DEST -Force";
    execFileSync("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", command], {
      stdio: "inherit",
      env: { ...process.env, GLOOP_ARCHIVE: archivePath, GLOOP_DEST: destDir },
    });
    return;
  }
  execFileSync("tar", ["-xzf", archivePath, "-C", destDir], { stdio: "ignore" });
}

function writeShim(name) {
  const shimDir = getShimDir();
  const versionDir = getBinDir();
  fs.mkdirSync(shimDir, { recursive: true });

  if (isWindows) {
    const shimPath = path.join(shimDir, `${name}.cmd`);
    const target = path.join(versionDir, `${name}.exe`);
    try { fs.unlinkSync(shimPath); } catch (_) {}
    fs.writeFileSync(
      shimPath,
      ["@echo off", `setlocal`, `"${target}" %*`, "exit /b %ERRORLEVEL%", ""].join("\r\n"),
      "utf8"
    );
    return;
  }

  const shimPath = path.join(shimDir, name);
  const target = path.join(versionDir, name);
  try { fs.unlinkSync(shimPath); } catch (_) {}
  fs.writeFileSync(
    shimPath,
    ["#!/bin/sh", 'DIR="$(CDPATH= cd "$(dirname "$0")" && pwd)"', `exec "$DIR/versions/v${VERSION}/${name}" "$@"`, ""].join("\n"),
    "utf8"
  );
  fs.chmodSync(shimPath, 0o755);
  // Keep target referenced so future refactors do not silently decouple the shim path.
  if (!fs.existsSync(target)) {
    throw new Error(`Expected installed binary missing for shim: ${target}`);
  }
}

function writeShims() {
  writeShim("gloop");
  writeShim("gloopd");
}

function shimsExist() {
  const shimDir = getShimDir();
  try {
    for (const name of ["gloop", "gloopd"]) {
      const shimPath = path.join(shimDir, name + (isWindows ? ".cmd" : ""));
      if (!fs.existsSync(shimPath)) return false;
      // Verify shim points to current version
      if (!isWindows) {
        const content = fs.readFileSync(shimPath, "utf8");
        if (!content.includes(`versions/v${VERSION}/`)) return false;
      }
    }
    return true;
  } catch (_) {
    return false;
  }
}

function sleep(ms) {
  Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, ms);
}

function acquireInstallLock() {
  const lockDir = path.join(path.dirname(binDir), `.install-v${VERSION}.lock`);
  const deadline = Date.now() + 30_000;
  while (true) {
    try {
      fs.mkdirSync(lockDir);
      fs.writeFileSync(path.join(lockDir, "pid"), String(process.pid), "utf8");
      return () => fs.rmSync(lockDir, { recursive: true, force: true });
    } catch (error) {
      if (error.code !== "EEXIST") throw error;
      try {
        const stat = fs.statSync(lockDir);
        if (Date.now() - stat.mtimeMs > 5 * 60_000) {
          fs.rmSync(lockDir, { recursive: true, force: true });
          continue;
        }
      } catch (_) {
        continue;
      }
      if (Date.now() > deadline) {
        throw new Error(`Timed out waiting for install lock: ${lockDir}`);
      }
      sleep(100);
    }
  }
}

function install() {
  if (!platform || !arch) {
    throw new Error(`Unsupported platform: ${process.platform}-${process.arch}`);
  }

  fs.mkdirSync(path.dirname(binDir), { recursive: true });
  const releaseLock = acquireInstallLock();
  try {
    if (isInstalled(binDir)) {
      if (!shimsExist()) writeShims();
      return;
    }

    const stagingDir = `${binDir}.tmp-${process.pid}`;
    fs.rmSync(stagingDir, { recursive: true, force: true });
    fs.mkdirSync(stagingDir, { recursive: true });
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "gloop-"));
    const archivePath = path.join(tmpDir, archiveName);

    try {
      const bundledArchive = bundledArchivePath();
      if (bundledArchive) {
        fs.copyFileSync(bundledArchive, archivePath);
      } else {
        let lastError;
        for (const url of resolveDownloadUrls(process.env)) {
          try {
            download(url, archivePath);
            lastError = null;
            break;
          } catch (error) {
            lastError = error;
          }
        }
        if (lastError) throw lastError;
      }

      verifyChecksum(archivePath, getExpectedChecksum(archiveName));
      extractArchive(archivePath, tmpDir);

      for (const name of ["gloop", "gloopd"]) {
        const binary = name + (isWindows ? ".exe" : "");
        const source = path.join(tmpDir, binary);
        const target = path.join(stagingDir, binary);
        fs.copyFileSync(source, target);
        if (!isWindows) fs.chmodSync(target, 0o755);
      }

      writeVersionMarker(stagingDir, VERSION);
      fs.rmSync(binDir, { recursive: true, force: true });
      fs.renameSync(stagingDir, binDir);
      writeShims();
      console.log(`${NAME} v${VERSION} installed successfully to ${binDir}`);
    } finally {
      fs.rmSync(stagingDir, { recursive: true, force: true });
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  } finally {
    releaseLock();
  }
}

if (require.main === module) {
  try {
    install();
  } catch (error) {
    console.error(`Failed to install ${NAME}: ${error.message}`);
    console.error(
      "\nIf the binary archive has not been uploaded yet, publish it to one of:\n" +
        `  ${DEFAULT_REGISTRY}/-/binary/gloop/v${VERSION}/${archiveName}\n` +
        "or set:\n" +
        "  GLOOP_BINARY_BASE_URL=https://your-host/gloop\n"
    );
    process.exit(1);
  }
}

module.exports = { resolveDownloadUrls, getExpectedChecksum, verifyChecksum, assertAllowedHost, getBinDir, getShimDir, install };
