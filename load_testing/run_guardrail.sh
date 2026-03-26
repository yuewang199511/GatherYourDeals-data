#!/usr/bin/env bash
# run_guardrail.sh — Verify seed state before running load tests.
#
# Runs a focused subset of the integration test suite (health, login,
# list-receipts, and receipt-count check) to confirm the database has been
# correctly seeded and the API is functional.
#
# Usage:
#   cd load_testing
#   ./run_guardrail.sh
#
# Required env vars (or .env entries):
#   GYD_TEST_USERNAME    username of the seeded test account
#   GYD_TEST_PASSWORD    password of the seeded test account
#   GYD_ADMIN_USERNAME   username of the admin account
#   GYD_ADMIN_PASSWORD   password of the admin account
#
# Optional env vars:
#   GYD_TARGET_URL       default: http://localhost:8080
#   GYD_SEED_COUNT       minimum expected receipt count (default: 1000)
#
# Exit codes:
#   0  all guardrail checks passed
#   1  one or more pytest checks failed
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

log()  { echo "[run_guardrail] $*"; }
err()  { echo "[run_guardrail] ERROR: $*" >&2; }

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
log "Service is reachable."

# ── Resolve Python interpreter ────────────────────────────────────────────────

if [ -x "$SCRIPT_DIR/.venv/bin/python3" ]; then
    PYTHON="$SCRIPT_DIR/.venv/bin/python3"
else
    PYTHON="python3"
fi

# ── Run integration guardrail checks ──────────────────────────────────────────

log "Running guardrail checks (health, login, list-receipts) ..."
if "$PYTHON" -m pytest integration/ \
    -k "test_health or test_login_success or test_list_receipts_success" \
    -v; then
    log "Integration checks passed."
else
    err "Guardrail checks failed — is the server running and configured correctly?"
    exit 1
fi

# ── Check seed receipt count ───────────────────────────────────────────────────

log "Checking seed receipt count ..."
if "$PYTHON" check_seed.py; then
    log "Guardrail passed — seed state is valid."
else
    err "Seed check failed — re-seed the database before running load tests."
    err "  Seed: cd load_testing && python3 seed.py"
    exit 1
fi
