#!/usr/bin/env bash
# run_integration.sh — Run the GatherYourDeals integration test suite.
#
# Exercises all HTTP endpoints against a live server using pytest.
# Reads configuration from load_testing/.env (same variables as seed.py and
# the locust run scripts).
#
# Usage:
#   cd load_testing
#   ./run_integration.sh
#
# Required env vars (or .env entries):
#   GYD_TEST_USERNAME    username of the test account
#   GYD_TEST_PASSWORD    password of the test account
#   GYD_ADMIN_USERNAME   username of the admin account
#   GYD_ADMIN_PASSWORD   password of the admin account
#
# Optional env vars:
#   GYD_TARGET_URL       default: http://localhost:8080
#
# Exit codes:
#   0  all tests passed
#   1  one or more tests failed
#   2  configuration error (required env vars missing)
#   3  connectivity error (server unreachable)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Auto-load .env if present (already-exported vars take precedence).
if [ -f "$SCRIPT_DIR/.env" ]; then
    set -a
    # shellcheck source=/dev/null
    source "$SCRIPT_DIR/.env"
    set +a
fi

# ── Helpers ───────────────────────────────────────────────────────────────────

log()  { echo "[run_integration] $*"; }
err()  { echo "[run_integration] ERROR: $*" >&2; }

# ── Validate required env vars ────────────────────────────────────────────────

missing=()
for var in GYD_TEST_USERNAME GYD_TEST_PASSWORD GYD_ADMIN_USERNAME GYD_ADMIN_PASSWORD; do
    if [ -z "${!var:-}" ]; then
        missing+=("$var")
    fi
done

if [ "${#missing[@]}" -gt 0 ]; then
    err "Missing required environment variables: ${missing[*]}"
    err "Set them in load_testing/.env or export them before running."
    exit 2
fi

GYD_TARGET_URL="${GYD_TARGET_URL:-http://localhost:8080}"

# ── Connectivity pre-check ────────────────────────────────────────────────────

log "Checking connectivity to ${GYD_TARGET_URL} ..."
if ! curl -sf --max-time 10 "${GYD_TARGET_URL}/health" -o /dev/null 2>/dev/null; then
    err "Cannot reach ${GYD_TARGET_URL}/health — is the server running?"
    err "  Start (Docker):  docker compose up --build -d"
    err "  Start (local):   ./gatheryourdeals serve"
    err "  Verify:          curl ${GYD_TARGET_URL}/health"
    exit 3
fi
log "Server is reachable."

# ── Run integration tests ─────────────────────────────────────────────────────

log "Running integration tests ..."
"$SCRIPT_DIR/.venv/bin/python3" -m pytest integration/ -v
