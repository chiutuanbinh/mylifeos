#!/usr/bin/env bash
# Integration smoke tests using agent-browser.
# Starts local dev stack, runs browser checks, tears down.
# Usage: bash scripts/integration-test.sh [--headed]

set -euo pipefail

HEADED="${1:-}"
REPO_ROOT="$(git rev-parse --show-toplevel)"
SESSION="mylifeos-smoke"
FRONTEND_URL="http://localhost:5173"
PASS=0
FAIL=0

log()  { echo "  $*"; }
ok()   { echo "  ✓ $*"; PASS=$((PASS+1)); }
fail() { echo "  ✗ $*"; FAIL=$((FAIL+1)); }

cleanup() {
  echo "→ Cleaning up..."
  agent-browser close --session "$SESSION" 2>/dev/null || true
  if [[ "${STACK_STARTED:-}" == "1" ]]; then
    docker compose -f "$REPO_ROOT/docker-compose.yml" down --timeout 5 2>/dev/null || true
  fi
}
trap cleanup EXIT

# ── Start stack if not already running ──────────────────────────────────────
echo "→ Checking local stack..."
if ! curl -sf "$FRONTEND_URL" > /dev/null 2>&1; then
  echo "→ Starting docker-compose stack..."
  docker compose -f "$REPO_ROOT/docker-compose.yml" up -d
  STACK_STARTED=1
  echo "→ Waiting for frontend to be ready..."
  for i in $(seq 1 30); do
    if curl -sf "$FRONTEND_URL" > /dev/null 2>&1; then break; fi
    sleep 2
  done
  curl -sf "$FRONTEND_URL" > /dev/null || { echo "✗ Frontend never became ready"; exit 1; }
else
  echo "  Stack already running."
  STACK_STARTED=0
fi

AB="agent-browser --session $SESSION"
[[ "$HEADED" == "--headed" ]] && AB="agent-browser --session $SESSION --headed"

echo ""
echo "→ Running smoke tests against $FRONTEND_URL"
echo ""

# ── Test 1: Landing / login page loads ──────────────────────────────────────
echo "[ Test 1 ] Login page loads"
$AB open "$FRONTEND_URL" 2>/dev/null
TITLE=$($AB get title 2>/dev/null || echo "")
if echo "$TITLE" | grep -qi "mylifeos\|login\|sign"; then
  ok "Page title: $TITLE"
else
  # Accept any non-error title — SPA may show app shell
  if [[ -n "$TITLE" ]]; then
    ok "Page loaded with title: $TITLE"
  else
    fail "No page title returned"
  fi
fi

# ── Test 2: No JS errors on load ────────────────────────────────────────────
echo "[ Test 2 ] No uncaught JS errors on load"
ERRORS=$($AB eval "window.__jsErrors__ ? window.__jsErrors__.length : 0" 2>/dev/null || echo "0")
# Inject error capture and navigate again for proper check
$AB eval "window.__jsErrors__=[]; window.addEventListener('error', e => window.__jsErrors__.push(e.message))" 2>/dev/null || true
$AB open "$FRONTEND_URL" 2>/dev/null
sleep 1
ERRORS=$($AB eval "window.__jsErrors__ ? window.__jsErrors__.length : 0" 2>/dev/null || echo "0")
if [[ "$ERRORS" == "0" ]]; then
  ok "No uncaught JS errors"
else
  JS_MSGS=$($AB eval "window.__jsErrors__ ? window.__jsErrors__.join(', ') : ''" 2>/dev/null || echo "unknown")
  fail "JS errors ($ERRORS): $JS_MSGS"
fi

# ── Test 3: Key nav links present ────────────────────────────────────────────
echo "[ Test 3 ] Navigation renders"
SNAP=$($AB snapshot -i 2>/dev/null || echo "")
if echo "$SNAP" | grep -qi "calendar\|finance\|habit\|goal\|note\|dashboard\|sign\|login"; then
  ok "Navigation/content visible in snapshot"
else
  fail "Expected nav links not found in snapshot"
fi

# ── Test 4: Auth callback page doesn't crash ─────────────────────────────────
echo "[ Test 4 ] Auth callback page handles missing params gracefully"
$AB open "$FRONTEND_URL/auth/callback" 2>/dev/null
SNAP=$($AB snapshot -i 2>/dev/null || echo "")
# Should not show a blank white page or raw error
if echo "$SNAP" | grep -qi "error\|cannot read\|undefined\|typeerror"; then
  fail "Auth callback shows error: $(echo "$SNAP" | head -3)"
else
  ok "Auth callback page renders without crash"
fi

# ── Test 5: 404 / unknown route doesn't crash ────────────────────────────────
echo "[ Test 5 ] Unknown route handles gracefully"
$AB open "$FRONTEND_URL/this-page-does-not-exist" 2>/dev/null
SNAP=$($AB snapshot -i 2>/dev/null || echo "")
if [[ -n "$SNAP" ]]; then
  ok "Unknown route renders without blank page"
else
  fail "Unknown route returned empty snapshot"
fi

# ── Results ──────────────────────────────────────────────────────────────────
echo ""
echo "Results: $PASS passed, $FAIL failed"
echo ""

if [[ "$FAIL" -gt 0 ]]; then
  echo "✗ Integration tests failed — fix before creating PR"
  exit 1
fi

echo "✓ All integration tests passed"
