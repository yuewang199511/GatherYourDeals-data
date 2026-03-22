"""
Group 3: Write Operations (Create-Then-Delete Cycle)

Each cycle: POST /api/v1/receipts → capture id → DELETE /api/v1/receipts/:id
Self-sustaining — no pre-seeding or queue exhaustion.
Tests real write-lock contention with mixed create/delete pressure.

POST and DELETE are tracked as separate Locust stats entries so P95/P99 are
independently visible (see docs/load_testing.md analysis tips).

Each user runs 1 cycle/s (constant_throughput(1.0)), so:
  moderate_users = GYD_RPS_MODERATE cycles/s → GYD_RPS_MODERATE * 2 effective writes/s
  stress_users   = GYD_RPS_STRESS   cycles/s → GYD_RPS_STRESS   * 2 effective writes/s

User counts and timing are driven by env vars (see load_testing/.env):
  GYD_RPS_MODERATE       moderate cycles/s    (default 100)
  GYD_RPS_STRESS         stress peak cycles/s (default 500)
  GYD_RAMP_TIME          ramp duration in s   (default 60)
  GYD_MODERATE_DURATION  moderate hold in s   (default 120)
  GYD_STRESS_HOLD        stress hold in s     (default 90)
"""

import uuid

from locust import constant_throughput, task

from common import BaseGYDUser, configure_context, load_config, login, make_shape, safe_login

_cfg = load_config()

configure_context(
    "write_ops",
    moderate_target_rps=_cfg["rps_moderate"] * 2,   # 2 HTTP requests per cycle
    stress_target_rps=_cfg["rps_stress"] * 2,
    moderate_users=_cfg["rps_moderate"],
    stress_peak_users=_cfg["rps_stress"],
)

# 1 task = 2 HTTP requests → endpoint_multiplier=1 (user count matches cycles/s)
GroupShape = make_shape(endpoint_multiplier=1)

# Minimal receipt payload satisfying all required fields
_RECEIPT_TEMPLATE = {
    "productName": "Load Test Item",
    "purchaseDate": "2026-01-01",
    "price": "9.99",
    "amount": "1",
    "storeName": "Load Test Store",
}


class WriteUser(BaseGYDUser):
    """Creates a receipt then immediately deletes it — one cycle per task invocation."""

    _test_group = "write_ops"
    wait_time = constant_throughput(1.0)

    def on_start(self):
        super().on_start()
        cfg = load_config()
        pair = safe_login(self.environment, cfg["target_url"], cfg["username"], cfg["password"])
        self._headers = {
            "Authorization": f"Bearer {pair['access']}",
            "Content-Type": "application/json",
        }

    @task
    def create_then_delete(self):
        # Step 1 — create receipt
        payload = {**_RECEIPT_TEMPLATE, "productName": f"LT-{uuid.uuid4().hex[:8]}"}
        with self.client.post(
            "/api/v1/receipts",
            json=payload,
            headers=self._headers,
            name="/api/v1/receipts [POST]",
            catch_response=True,
        ) as post_resp:
            if post_resp.status_code != 201:
                post_resp.failure(
                    f"POST /receipts failed: HTTP {post_resp.status_code}"
                )
                return
            try:
                receipt_id = post_resp.json()["id"]
            except (KeyError, ValueError) as exc:
                post_resp.failure(f"POST /receipts: could not parse id — {exc}")
                return
            post_resp.success()

        # Step 2 — delete the receipt we just created
        self.client.delete(
            f"/api/v1/receipts/{receipt_id}",
            headers=self._headers,
            name="/api/v1/receipts/:id [DELETE]",
        )
