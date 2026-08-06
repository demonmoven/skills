#!/bin/bash
# setup.sh - Auto-install all fixloop-universal dependencies (idempotent)
# Usage: bash setup.sh [--browser-mode system_chrome|bundled|cdp_connect|headless]

set -euo pipefail

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info()  { echo -e "${BLUE}[setup]${NC} $*"; }
ok()    { echo -e "${GREEN}[setup]${NC} $*"; }
warn()  { echo -e "${YELLOW}[setup]${NC} $*"; }
fail()  { echo -e "${RED}[setup]${NC} $*"; exit 1; }

# --- Parse args ---
BROWSER_MODE="bundled"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --browser-mode) BROWSER_MODE="$2"; shift 2 ;;
    *) shift ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="${FIXLOOP_OUTPUT_DIR:-./fixloop-output}"

info "Starting fixloop-universal setup..."
info "Browser mode: $BROWSER_MODE"
info "Script dir: $SCRIPT_DIR"

# =============================================================================
# 1. Check Node.js >= 18
# =============================================================================
info "Checking Node.js version..."
if ! command -v node &>/dev/null; then
  fail "Node.js is not installed. Please install Node.js >= 18."
fi

NODE_VERSION=$(node -e "console.log(process.versions.node.split('.')[0])")
if [ "$NODE_VERSION" -lt 18 ]; then
  fail "Node.js >= 18 required, found v$(node -v). Please upgrade."
fi
ok "Node.js v$(node -v) OK"

# =============================================================================
# 2. Check @playwright/mcp
# =============================================================================
info "Checking @playwright/mcp..."
if npx @playwright/mcp --version &>/dev/null 2>&1; then
  ok "@playwright/mcp already available"
else
  info "Installing @playwright/mcp globally..."
  npm install -g @playwright/mcp
  if npx @playwright/mcp --version &>/dev/null 2>&1; then
    ok "@playwright/mcp installed successfully"
  else
    fail "Failed to install @playwright/mcp"
  fi
fi

# =============================================================================
# 3. Check Playwright browsers (skip for system_chrome)
# =============================================================================
if [ "$BROWSER_MODE" = "system_chrome" ]; then
  info "Browser mode is system_chrome, skipping Playwright browser install"
else
  info "Checking Playwright browsers..."
  # Check if chromium is already installed by looking for browser executable
  if npx playwright install --dry-run chromium 2>&1 | grep -q "already installed" || \
     [ -d "$HOME/Library/Caches/ms-playwright/chromium-"* ] 2>/dev/null || \
     [ -d "$HOME/.cache/ms-playwright/chromium-"* ] 2>/dev/null; then
    ok "Playwright chromium browser appears to be installed"
  else
    info "Installing Playwright chromium browser..."
    npx playwright install chromium
    ok "Playwright chromium installed"
  fi
fi

# =============================================================================
# 4. Install npm deps into scripts/node_modules/
# =============================================================================
info "Checking local npm dependencies..."

DEPS=(
  "@metoto/playwright-trace-analyzer-mcp"
  "gremlins.js"
  "pixelmatch"
  "pngjs"
  "@axe-core/playwright"
)

MISSING_DEPS=()
for dep in "${DEPS[@]}"; do
  # Resolve the package directory name for checking
  if [ -d "$SCRIPT_DIR/node_modules/$dep" ]; then
    ok "  $dep already installed"
  else
    MISSING_DEPS+=("$dep")
  fi
done

if [ ${#MISSING_DEPS[@]} -gt 0 ]; then
  info "Installing missing deps: ${MISSING_DEPS[*]}"
  npm install --prefix "$SCRIPT_DIR" "${MISSING_DEPS[@]}"
  ok "Dependencies installed to $SCRIPT_DIR/node_modules/"
else
  ok "All npm dependencies already present"
fi

# =============================================================================
# 5. Create output directories
# =============================================================================
info "Ensuring output directories exist..."

mkdir -p "$OUTPUT_DIR/traces"
mkdir -p "$OUTPUT_DIR/timelines"
mkdir -p "$OUTPUT_DIR/iterations"

ok "Output directories ready at $OUTPUT_DIR/"

# =============================================================================
# 6. Health check — verify all deps can be required
# =============================================================================
info "Running health check..."

HEALTH_CHECK_SCRIPT=$(cat <<'HEALTHEOF'
const deps = [
  'pixelmatch',
  'pngjs',
  '@axe-core/playwright',
  'gremlins.js'
];
let allOk = true;
for (const dep of deps) {
  try {
    require(dep);
    console.log(`  OK: ${dep}`);
  } catch (e) {
    console.error(`  FAIL: ${dep} — ${e.message}`);
    allOk = false;
  }
}
try {
  require.resolve('@metoto/playwright-trace-analyzer-mcp');
  console.log('  OK: @metoto/playwright-trace-analyzer-mcp');
} catch (e) {
  console.error('  FAIL: @metoto/playwright-trace-analyzer-mcp — ' + e.message);
  allOk = false;
}
if (!allOk) {
  process.exit(1);
}
HEALTHEOF
)

# Run health check with scripts/node_modules in the resolve path
if NODE_PATH="$SCRIPT_DIR/node_modules" node -e "$HEALTH_CHECK_SCRIPT"; then
  ok "Health check passed"
else
  fail "Health check failed — some dependencies cannot be loaded"
fi

# =============================================================================
# Done
# =============================================================================
echo ""
ok "fixloop-universal setup complete!"
info "  Browser mode : $BROWSER_MODE"
info "  Deps location: $SCRIPT_DIR/node_modules/"
info "  Output dir   : $OUTPUT_DIR/"
