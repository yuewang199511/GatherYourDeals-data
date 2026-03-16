"""
Group 2: Read-Heavy (Concurrent) — four GET endpoints run simultaneously.

All four endpoints are weighted equally so each receives GYD_RPS_MODERATE RPS.
Total users = GYD_RPS_MODERATE * 4 (moderate) / GYD_RPS_STRESS * 4 (stress).

User counts and timing are driven by env vars (see load_testing/.env):
  GYD_RPS_MODERATE       moderate RPS per endpoint  (default 100)
  GYD_RPS_STRESS         stress peak RPS per endpoint (default 500)
  GYD_RAMP_TIME          ramp duration in s           (default 60)
  GYD_MODERATE_DURATION  moderate hold in s           (default 120)
  GYD_STRESS_HOLD        stress hold in s             (default 90)
"""

import random
import sys

import requests as _requests
from locust import constant_throughput, events, task
from locust.contrib.fasthttp import FastHttpUser

from common import TEST_RUN_ID, configure_context, load_config, login, make_shape

_cfg = load_config()
_ENDPOINTS = 4

configure_context(
    "read_heavy",
    moderate_target_rps=_cfg["rps_moderate"],   # per endpoint
    stress_target_rps=_cfg["rps_stress"],
    moderate_users=_cfg["rps_moderate"] * _ENDPOINTS,
    stress_peak_users=_cfg["rps_stress"] * _ENDPOINTS,
)

# 4 equal-weight endpoints → 4× multiplier
GroupShape = make_shape(endpoint_multiplier=_ENDPOINTS)

# Pre-fetched receipt IDs — populated once before users are spawned
_receipt_ids: list[str] = []


@events.test_start.add_listener
def _prefetch_receipt_ids(environment, **kwargs):
    """Fetch a page of receipt IDs using plain requests (not tracked in Locust stats)."""
    global _receipt_ids
    cfg = load_config()
    try:
        pair = login(cfg["target_url"], cfg["username"], cfg["password"])
        resp = _requests.get(
            f"{cfg['target_url']}/api/v1/receipts?limit=50&offset=0",
            headers={"Authorization": f"Bearer {pair['access']}"},
            timeout=10,
        )
        if resp.status_code == 200:
            data = resp.json()
            _receipt_ids = [item["id"] for item in data.get("data", []) if "id" in item]
            print(
                f"[group2] Pre-fetched {len(_receipt_ids)} receipt IDs for GET :id tasks",
                file=sys.stderr,
            )
    except Exception as exc:
        print(f"[group2] WARNING: Could not pre-fetch receipt IDs: {exc}", file=sys.stderr)


class ReadUser(FastHttpUser):
    """Concurrent reader across all four GET endpoints."""

    wait_time = constant_throughput(1.0)

    def on_start(self):
        cfg = load_config()
        phase = cfg["phase"]
        target_rps = cfg["rps_moderate"] if phase == "moderate" else cfg["rps_stress"]
        pair = login(cfg["target_url"], cfg["username"], cfg["password"])
        self._headers = {
            "Authorization": f"Bearer {pair['access']}",
            "X-Test-Run-Id": TEST_RUN_ID,
            "X-Test-Group": "read_heavy",
            "X-Test-Phase": phase,
            "X-Test-Target-Rps": str(target_rps),
        }

    @task(1)
    def list_receipts(self):
        self.client.get(
            "/api/v1/receipts?limit=20&offset=0",
            headers=self._headers,
            name="/api/v1/receipts",
        )

    @task(1)
    def get_receipt(self):
        if _receipt_ids:
            receipt_id = random.choice(_receipt_ids)
        else:
            return
        self.client.get(
            f"/api/v1/receipts/{receipt_id}",
            headers=self._headers,
            name="/api/v1/receipts/:id",
        )

    @task(1)
    def get_me(self):
        self.client.get(
            "/api/v1/auth/me",
            headers=self._headers,
            name="/api/v1/auth/me",
        )

    @task(1)
    def list_meta(self):
        self.client.get(
            "/api/v1/meta",
            headers=self._headers,
            name="/api/v1/meta",
        )
