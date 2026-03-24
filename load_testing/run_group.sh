#!/usr/bin/env bash
# run_group.sh — Run a single load test group.
#
# Usage:
#   cd load_testing
#   ./run_group.sh <GROUP> [PHASE]
#
#   GROUP  : 1, 2, 3, or 4 (required)
#   PHASE  : moderate | stress | all  (optional, default: all)
#
# Examples:
#   ./run_group.sh 2            # Group 2, both phases
#   ./run_group.sh 1 stress     # Group 1, stress phase only
#   ./run_group.sh 3 moderate   # Group 3, moderate phase only
#
# Required env vars: GYD_TEST_USERNAME, GYD_TEST_PASSWORD
# Optional env vars: GYD_TARGET_URL, GYD_PLATFORM_NAME, GYD_DB_TYPE, GYD_RESULTS_DIR

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Auto-load .env if present (values are overridden by any already-exported vars)
if [ -f "$SCRIPT_DIR/.env" ]; then
    set -a
    # shellcheck source=/dev/null
    source "$SCRIPT_DIR/.env"
    set +a
fi

# ── Helpers ───────────────────────────────────────────────────────────────────

log()  { echo "[run_group] $*"; }
warn() { echo "[run_group] WARNING: $*" >&2; }
err()  { echo "[run_group] ERROR: $*" >&2; }

usage() {
    echo "Usage: cd load_testing && ./run_group.sh <GROUP> [PHASE]"
    echo ""
    echo "  GROUP  : 1 | 2 | 3 | 4"
    echo "  PHASE  : moderate | stress | all  (default: all)"
    echo ""
    echo "Required env vars: GYD_TEST_USERNAME, GYD_TEST_PASSWORD"
    echo "Optional env vars: GYD_TARGET_URL (default: http://localhost:8080)"
    echo "                   GYD_PLATFORM_NAME, GYD_DB_TYPE, GYD_RESULTS_DIR"
    exit 1
}

check_error_rate() {
    local csv_file="$1"

    if [ ! -f "$csv_file" ]; then
        warn "Stats CSV not found: $csv_file — skipping error rate check"
        return 0
    fi

    local rate
    rate=$(python3 "$SCRIPT_DIR/check_error_rate.py" "$csv_file" 5)
    local exit_code=$?

    if [ "$exit_code" -ne 0 ]; then
        warn "Moderate phase error rate ${rate}% >= 5% — skipping Stress phase"
        return 1
    fi
    return 0
}

run_phase() {
    local locust_file="$1"
    local group_label="$2"
    local phase="$3"
    local csv_prefix="${RUN_DIR}/${group_label}_${phase}"
    local html_file="${RUN_DIR}/${group_label}_${phase}.html"

    log "Starting ${group_label} / ${phase} …"

    # Start 4 workers in the background; they wait for the master to connect.
    LOCUST_FILE="$locust_file" GYD_PHASE="$phase" \
        docker compose up -d --scale locust-worker=4 --no-recreate locust-worker

    # Run the master (blocks until the test completes).
    GYD_PHASE="$phase" GYD_RESULTS_DIR="results/${RUN_TIMESTAMP}" \
        docker compose run --rm locust-master \
            -f "locust/${locust_file}" \
            --master \
            --expect-workers 4 \
            --headless \
            --host "${GYD_TARGET_URL:-http://localhost:8080}" \
            --csv  "$csv_prefix" \
            --html "$html_file" \
            --stop-timeout 10

    local exit_code=$?

    # Always stop and remove workers after each phase.
    docker compose stop locust-worker
    docker compose rm -f locust-worker

    if [ "$exit_code" -ne 0 ]; then
        err "Locust exited with code ${exit_code} for ${group_label}/${phase}"
        return 2
    fi
    log "Finished ${group_label} / ${phase} → ${csv_prefix}_stats.csv  ${html_file}"
}

# ── Parse arguments ───────────────────────────────────────────────────────────

GROUP="${1:-}"
PHASE="${2:-all}"

if [ -z "$GROUP" ]; then
    err "GROUP argument is required."
    usage
fi

case "$GROUP" in
    1) LOCUST_FILE="group1_cpu.py";    GROUP_LABEL="group1_cpu" ;;
    2) LOCUST_FILE="group2_reads.py";  GROUP_LABEL="group2_reads" ;;
    3) LOCUST_FILE="group3_writes.py"; GROUP_LABEL="group3_writes" ;;
    4) LOCUST_FILE="group4_misc.py";   GROUP_LABEL="group4_misc" ;;
    *)
        err "Invalid GROUP '${GROUP}'. Must be 1, 2, 3, or 4."
        usage
        ;;
esac

case "$PHASE" in
    moderate|stress|all) ;;
    *)
        err "Invalid PHASE '${PHASE}'. Must be 'moderate', 'stress', or 'all'."
        usage
        ;;
esac

# ── Validate env vars ─────────────────────────────────────────────────────────

if [ -z "${GYD_TEST_USERNAME:-}" ]; then
    err "GYD_TEST_USERNAME is not set."
    usage
fi
if [ -z "${GYD_TEST_PASSWORD:-}" ]; then
    err "GYD_TEST_PASSWORD is not set."
    usage
fi
if [ -z "${GYD_TARGET_URL:-}" ]; then
    warn "GYD_TARGET_URL is not set — defaulting to http://localhost:8080"
    export GYD_TARGET_URL="http://localhost:8080"
fi

# ── Connectivity pre-check ────────────────────────────────────────────────────

log "Checking connectivity to ${GYD_TARGET_URL} …"
if ! curl -sf --max-time 10 "${GYD_TARGET_URL}/health" -o /dev/null 2>/dev/null; then
    err "Cannot reach ${GYD_TARGET_URL}/health — service is not running or needs a rebuild."
    err "  Start:   docker compose up --build -d   (or: ./gatheryourdeals serve)"
    err "  Verify:  curl ${GYD_TARGET_URL}/health"
    exit 1
fi
log "Service is reachable."

# ── Seed guardrail check ──────────────────────────────────────────────────────

log "Running seed guardrail check …"
if ! "$SCRIPT_DIR/run_guardrail.sh"; then
    err "Seed guardrail failed — re-seed the database before running load tests."
    err "  Seed: cd load_testing && python3 seed.py"
    exit 4
fi
log "Guardrail passed."

# ── Ensure results directory exists ──────────────────────────────────────────

RUN_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RUN_DIR="${GYD_RESULTS_DIR:-results}/${RUN_TIMESTAMP}"
mkdir -p "$RUN_DIR"
log "Results directory: ${RUN_DIR}"

# ── Execute ───────────────────────────────────────────────────────────────────

log "Group ${GROUP} (${GROUP_LABEL}), phase: ${PHASE}"

case "$PHASE" in
    moderate)
        run_phase "$LOCUST_FILE" "$GROUP_LABEL" "moderate" || exit 2
        ;;
    stress)
        run_phase "$LOCUST_FILE" "$GROUP_LABEL" "stress" || exit 2
        ;;
    all)
        # Moderate first
        run_phase "$LOCUST_FILE" "$GROUP_LABEL" "moderate" || exit 2

        # Check error rate before stress
        moderate_csv="${RUN_DIR}/${GROUP_LABEL}_moderate_stats.csv"
        if check_error_rate "$moderate_csv"; then
            run_phase "$LOCUST_FILE" "$GROUP_LABEL" "stress" || exit 2
        fi
        ;;
esac

log "Done. Results in: ${RUN_DIR}/"
exit 0
