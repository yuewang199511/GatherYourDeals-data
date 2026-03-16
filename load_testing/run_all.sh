#!/usr/bin/env bash
# run_all.sh — Run the full GatherYourDeals load test suite.
#
# Runs all 4 groups sequentially. Each group runs Moderate phase first,
# then Stress phase (skipped if Moderate error rate >= 5%).
#
# Total runtime: ~18 minutes against a responsive target.
#
# Usage:
#   cd load_testing
#   ./run_all.sh
#
# Required env vars: GYD_TEST_USERNAME, GYD_TEST_PASSWORD
# Optional env vars: GYD_TARGET_URL (default http://localhost:8080)
#                    GYD_PLATFORM_NAME, GYD_DB_TYPE, GYD_RESULTS_DIR

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

# ── Helpers ──────────────────────────────────────────────────────────────────

log()  { echo "[run_all] $*"; }
warn() { echo "[run_all] WARNING: $*" >&2; }
err()  { echo "[run_all] ERROR: $*" >&2; }

# Print usage and exit 1
usage() {
    echo "Usage: cd load_testing && ./run_all.sh"
    echo ""
    echo "Required env vars:"
    echo "  GYD_TEST_USERNAME   username of the pre-seeded test account"
    echo "  GYD_TEST_PASSWORD   password of the pre-seeded test account"
    echo ""
    echo "Optional env vars:"
    echo "  GYD_TARGET_URL      default: http://localhost:8080"
    echo "  GYD_PLATFORM_NAME   default: local"
    echo "  GYD_DB_TYPE         default: sqlite"
    echo "  GYD_RESULTS_DIR     default: results"
    exit 1
}

# Return 0 if error rate in the given stats CSV is below threshold (5%).
# Returns 1 (skip stress) if error rate >= 5% or CSV cannot be read.
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

# Run one Locust invocation (one group, one phase).
# Args: <locust_file> <group_label> <phase>
run_phase() {
    local locust_file="$1"
    local group_label="$2"
    local phase="$3"
    local csv_prefix="${RUN_DIR}/${group_label}_${phase}"
    local html_file="${RUN_DIR}/${group_label}_${phase}.html"

    log "Starting ${group_label} / ${phase} …"

    GYD_PHASE="$phase" GYD_RESULTS_DIR="results/${RUN_TIMESTAMP}" docker compose run --rm locust \
        -f "locust/${locust_file}" \
        --headless \
        --host "${GYD_TARGET_URL:-http://localhost:8080}" \
        --csv  "$csv_prefix" \
        --html "$html_file" \
        --stop-timeout 10

    local exit_code=$?
    if [ "$exit_code" -ne 0 ]; then
        err "Locust exited with code ${exit_code} for ${group_label}/${phase}"
        return 2
    fi
    log "Finished ${group_label} / ${phase} → ${csv_prefix}_stats.csv  ${html_file}"
}

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

# ── Ensure results directory exists ──────────────────────────────────────────

RUN_TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RUN_DIR="${GYD_RESULTS_DIR:-results}/${RUN_TIMESTAMP}"
mkdir -p "$RUN_DIR"
log "Results directory: ${RUN_DIR}"

# ── Group definitions: (locust_file, group_label) ────────────────────────────

declare -a LOCUST_GROUPS=(
    "group1_cpu.py|group1_cpu"
    "group2_reads.py|group2_reads"
    "group3_writes.py|group3_writes"
    "group4_misc.py|group4_misc"
)

# ── Run all groups ────────────────────────────────────────────────────────────

FAILED_GROUPS=()

for entry in "${LOCUST_GROUPS[@]}"; do
    locust_file="${entry%%|*}"
    group_label="${entry##*|}"

    log "═══════════════════════════════════════════════════"
    log "Group: ${group_label}"
    log "═══════════════════════════════════════════════════"

    # Moderate phase
    if ! run_phase "$locust_file" "$group_label" "moderate"; then
        FAILED_GROUPS+=("${group_label}/moderate")
        warn "Moderate phase failed for ${group_label} — skipping Stress phase, continuing to next group"
        continue
    fi

    # Check error rate before running Stress
    moderate_csv="${RUN_DIR}/${group_label}_moderate_stats.csv"
    if ! check_error_rate "$moderate_csv"; then
        # Warning already printed inside check_error_rate
        continue
    fi

    # Stress phase
    if ! run_phase "$locust_file" "$group_label" "stress"; then
        FAILED_GROUPS+=("${group_label}/stress")
        warn "Stress phase failed for ${group_label} — continuing to next group"
    fi
done

# ── Summary ───────────────────────────────────────────────────────────────────

log ""
log "═══════════════════════════════════════════════════"
log "Run complete. Results in: ${RUN_DIR}/"
log "═══════════════════════════════════════════════════"

if [ "${#FAILED_GROUPS[@]}" -gt 0 ]; then
    warn "The following phases had failures:"
    for g in "${FAILED_GROUPS[@]}"; do
        warn "  • ${g}"
    done
    exit 2
fi

log "All groups completed successfully."
exit 0
