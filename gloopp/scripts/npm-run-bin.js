#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");
const { spawn } = require("node:child_process");
const { getBinDir, getShimDir, install } = require("./npm-install.js");

function runUpdateNotifier() {
  if (process.env.GLOOP_NO_UPDATE_NOTIFIER === "1" || process.env.NO_UPDATE_NOTIFIER) {
    return;
  }
  try {
    const pkg = JSON.parse(
      fs.readFileSync(path.join(__dirname, "..", "package.json"), "utf8")
    );
    // eslint-disable-next-line global-require
    const { default: updateNotifier } = require("update-notifier");
    const notifier = updateNotifier({
      pkg,
      updateCheckInterval: 1000 * 60 * 60 * 24,
      distTag: "latest",
      registryUrl: "https://bnpm.byted.org",
      shouldNotifyInNpmScript: false,
    });
    notifier.notify({
      defer: true,
      isGlobal: true,
      message: [
        "{packageName} 有新版本可用：{currentVersion} → {latestVersion}",
        "执行  npx @bytedance-dev/gloop update  或重新安装升级。",
        "（关闭提示：export GLOOP_NO_UPDATE_NOTIFIER=1）",
      ].join("\n"),
    });
  } catch (_) {
  }
}

runUpdateNotifier();

const invoked = path.basename(process.argv[1] || "gloop", ".js");
const nativeName = "gloop";
const exe = process.platform === "win32" ? ".exe" : "";
const binDir = getBinDir();
const shimDir = getShimDir();
const nativeBin = path.join(binDir, `${nativeName}${exe}`);

function ensureBinInPath() {
  if (process.env.GLOOP_NO_PATH_HINT === "1") return;
  const envPath = process.env.PATH || "";
  if (!envPath.split(path.delimiter).includes(shimDir)) {
    const shellRc = process.platform === "darwin" ? "~/.zshrc" : "~/.bashrc";
    console.error(`[gloop] Tip: Add ${shimDir} to your PATH for direct access:`);
    console.error(`       echo 'export PATH="${shimDir}:$PATH"' >> ${shellRc} && source ${shellRc}`);
    console.error("");
  }
}

function warnIfNpmArgsMisrouted() {
  const args = process.argv.slice(2);
  const npmFlags = [];
  for (const arg of args) {
    if (arg === "--registry" || arg.startsWith("--registry=") || arg === "-reg" || arg.startsWith("-reg=")) {
      npmFlags.push(arg);
    }
  }
  if (npmFlags.length > 0) {
    console.error("[gloop] 检测到 npm/npx 参数被误传给 gloop。");
    console.error("       npx 参数需要放在包名**前面**，包名后面的参数都会传给 gloop 本身：");
    console.error("");
    console.error("  错误：npx @bytedance-dev/gloop start --registry https://bnpm.byted.org");
    console.error("  正确：npm_config_registry=https://bnpm.byted.org npx @bytedance-dev/gloop start");
    console.error("  正确：npx --registry https://bnpm.byted.org @bytedance-dev/gloop start");
    console.error("");
    process.exit(2);
  }
}

warnIfNpmArgsMisrouted();

try {
  install();
  if (!fs.existsSync(nativeBin)) {
    throw new Error("binary still missing after install");
  }
  ensureBinInPath();
} catch (error) {
  console.error(
    [
      `gloop: failed to install native binary: ${error.message}`,
      `Target path: ${nativeBin}`,
      "Try running: node scripts/npm-install.js",
    ].join("\n")
  );
  process.exit(1);
}

const child = spawn(nativeBin, process.argv.slice(2), {
  stdio: "inherit",
  env: process.env,
  argv0: invoked,
});

child.on("exit", (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
    return;
  }
  process.exit(code || 0);
});

child.on("error", (error) => {
  console.error(`gloop: failed to start native binary: ${error.message}`);
  process.exit(1);
});
