"""
common.py — Shared infrastructure for all GatherYourDeals load test groups.

Provides:
  - load_config()          : read all GYD_* env vars
  - login()                : POST /api/v1/auth/login → {"access": ..., "refresh": ...}
  - TokenPool              : thread-safe queue of login token pairs (Group 4)
  - token_pool             : module-level TokenPool singleton
  - make_shape()           : factory for per-group LoadTestShape subclass
  - configure_context()    : register event listeners that write _context.json
  - set_context_token_pool_stats() : Group 4 hook to add pool stats to context
"""

import json
import os
import sys
import threading
import time
from datetime import datetime, timezone
from queue import Empty, Queue

import requests
from locust import events


# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------

def load_config():
    """Read all GYD_* env vars and return a plain dict. Applies defaults and warnings."""
    target_url = os.environ.get("GYD_TARGET_URL", "").rstrip("/")
    if not target_url:
        target_url = "http://localhost:8080"
        print(
            "WARNING: GYD_TARGET_URL is not set — defaulting to http://localhost:8080",
            file=sys.stderr,
        )

    phase = os.environ.get("GYD_PHASE", "moderate").lower()
    if phase not in ("moderate", "stress"):
        print(
            f"WARNING: GYD_PHASE='{phase}' is not 'moderate' or 'stress' — defaulting to 'moderate'",
            file=sys.stderr,
        )
        phase = "moderate"

    return {
        "target_url": target_url,
        "username": os.environ.get("GYD_TEST_USERNAME", ""),
        "password": os.environ.get("GYD_TEST_PASSWORD", ""),
        "platform_name": os.environ.get("GYD_PLATFORM_NAME", "local"),
        "db_type": os.environ.get("GYD_DB_TYPE", "sqlite"),
        "phase": phase,
        "results_dir": os.environ.get("GYD_RESULTS_DIR", "results"),
        # Load profile tuning — main endpoints
        "rps_moderate":        int(os.environ.get("GYD_RPS_MODERATE",        "100")),
        "rps_stress":          int(os.environ.get("GYD_RPS_STRESS",          "500")),
        "ramp_time":           int(os.environ.get("GYD_RAMP_TIME",           "60")),
        "moderate_duration":   int(os.environ.get("GYD_MODERATE_DURATION",   "120")),
        "stress_hold":         int(os.environ.get("GYD_STRESS_HOLD",         "90")),
        # Load profile tuning — Group 4 fixed-rate endpoints
        "logout_rps_moderate": int(os.environ.get("GYD_LOGOUT_RPS_MODERATE", "10")),
        "logout_rps_stress":   int(os.environ.get("GYD_LOGOUT_RPS_STRESS",   "40")),
        "meta_rps_moderate":   int(os.environ.get("GYD_META_RPS_MODERATE",   "5")),
        "meta_rps_stress":     int(os.environ.get("GYD_META_RPS_STRESS",     "20")),
    }


# ---------------------------------------------------------------------------
# Auth helper
# ---------------------------------------------------------------------------

def login(base_url, username, password):
    """
    POST /api/v1/auth/login and return {"access": <token>, "refresh": <token>}.
    Raises RuntimeError on non-200 response.
    Uses a plain requests.Session (not tracked by Locust stats).
    """
    resp = requests.post(
        f"{base_url}/api/v1/auth/login",
        json={"username": username, "password": password},
        timeout=10,
    )
    if resp.status_code != 200:
        raise RuntimeError(
            f"login() failed: HTTP {resp.status_code} — {resp.text[:200]}"
        )
    data = resp.json()
    return {
        "access": data["access_token"],
        "refresh": data["refresh_token"],
    }


# ---------------------------------------------------------------------------
# Token pool (Group 4 only)
# ---------------------------------------------------------------------------

class TokenPool:
    """Thread-safe FIFO pool of {"access": ..., "refresh": ...} token pairs."""

    def __init__(self):
        self._queue = Queue()
        self._lock = threading.Lock()
        self._generated = 0
        self._consumed = 0

    def put(self, token_pair):
        self._queue.put(token_pair)
        with self._lock:
            self._generated += 1

    def get(self):
        """Return a token pair dict, or None if the pool is empty."""
        try:
            pair = self._queue.get_nowait()
            with self._lock:
                self._consumed += 1
            return pair
        except Empty:
            return None

    def size(self):
        return self._queue.qsize()

    def stats(self):
        with self._lock:
            return {
                "pre_generated": self._generated,
                "consumed": self._consumed,
                "remaining": self._queue.qsize(),
            }


# Module-level singleton — imported by group4_misc.py
token_pool = TokenPool()


# ---------------------------------------------------------------------------
# LoadTestShape factory
# ---------------------------------------------------------------------------

def make_shape(endpoint_multiplier=1, moderate_total=None, stress_total=None):
    """
    Return a LoadTestShape subclass whose user counts and timing are read from config.

    endpoint_multiplier:
        For groups where N equal-weight endpoints share the same user pool.
        moderate_users = GYD_RPS_MODERATE * endpoint_multiplier
        stress_peak    = GYD_RPS_STRESS   * endpoint_multiplier

    moderate_total / stress_total:
        Explicit overrides — use when the group has mixed per-endpoint targets
        that don't fit a simple multiplier (Group 4).

    Timing is always read from config:
        GYD_MODERATE_DURATION  hold time at moderate level (default 120 s)
        GYD_RAMP_TIME          ramp duration for stress phase (default 60 s)
        GYD_STRESS_HOLD        hold time at stress peak (default 90 s)
    """
    from locust import LoadTestShape  # imported here to avoid circular issues

    cfg = load_config()
    _moderate_users = moderate_total if moderate_total is not None \
        else cfg["rps_moderate"] * endpoint_multiplier
    _stress_peak    = stress_total   if stress_total   is not None \
        else cfg["rps_stress"]   * endpoint_multiplier
    _stress_baseline  = _moderate_users
    _ramp_time        = cfg["ramp_time"]
    _moderate_duration = cfg["moderate_duration"]
    _stress_hold      = cfg["stress_hold"]

    class GroupShape(LoadTestShape):
        def tick(self):
            phase = os.environ.get("GYD_PHASE", "moderate").lower()
            t = self.get_run_time()

            if phase == "stress":
                if t < _ramp_time:
                    progress = t / _ramp_time
                    users = int(
                        _stress_baseline
                        + (_stress_peak - _stress_baseline) * progress
                    )
                    spawn_rate = max(
                        1.0, (_stress_peak - _stress_baseline) / _ramp_time
                    )
                    return users, spawn_rate
                elif t < _ramp_time + _stress_hold:
                    return _stress_peak, 1
                else:
                    return None
            else:  # moderate
                if t < _moderate_duration:
                    return _moderate_users, _moderate_users
                else:
                    return None

    return GroupShape


# ---------------------------------------------------------------------------
# Context JSON writer
# ---------------------------------------------------------------------------

_context_data = {}


def configure_context(group_name, moderate_target_rps, stress_target_rps,
                      moderate_users=0, stress_peak_users=0):
    """
    Register Locust event listeners that write a _context.json file for each run.

    Call this once at module level in each group locustfile:
        configure_context("cpu_bound", moderate_target_rps=10, stress_target_rps=30)

    The file is written on the 'quitting' event (after all test_stop listeners),
    so Group 4 can update token_pool_stats before the file is flushed.
    """

    @events.test_start.add_listener
    def _on_test_start(environment, **kwargs):
        cfg = load_config()
        phase = cfg["phase"]
        _context_data.update(
            {
                "target_url": cfg["target_url"],
                "platform_name": cfg["platform_name"],
                "database_type": cfg["db_type"],
                "group": group_name,
                "phase": phase,
                "target_rps": moderate_target_rps
                if phase == "moderate"
                else stress_target_rps,
                "num_users": moderate_users
                if phase == "moderate"
                else stress_peak_users,
                "test_start": datetime.now(timezone.utc).isoformat(),
                "token_pool_stats": None,
                # internal — stripped before writing
                "_start_ts": time.monotonic(),
                "_results_dir": cfg["results_dir"],
            }
        )

    @events.test_stop.add_listener
    def _on_test_stop(environment, **kwargs):
        _context_data["test_end"] = datetime.now(timezone.utc).isoformat()
        elapsed = time.monotonic() - _context_data.get("_start_ts", time.monotonic())
        _context_data["duration_seconds"] = round(elapsed, 1)
        # num_users is set at test_start from the shape config (peak users for the phase)

    @events.quitting.add_listener
    def _on_quitting(environment, **kwargs):
        phase = _context_data.get("phase", "moderate")
        results_dir = _context_data.get("_results_dir", "results")
        output_path = os.path.join(results_dir, f"{group_name}_{phase}_context.json")
        os.makedirs(results_dir, exist_ok=True)
        # Strip internal tracking keys before serialising
        output = {k: v for k, v in _context_data.items() if not k.startswith("_")}
        with open(output_path, "w") as f:
            json.dump(output, f, indent=2)
        print(f"\nContext written → {output_path}", file=sys.stderr)


def set_context_token_pool_stats(stats):
    """
    Override token_pool_stats in the pending context output.
    Call from a test_stop listener in group4_misc.py BEFORE quitting fires.
    """
    _context_data["token_pool_stats"] = stats
