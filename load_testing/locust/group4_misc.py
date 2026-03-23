"""
Group 4: Lightweight Misc (Concurrent) — three low-resource endpoints.

Task weights are set dynamically from config so the refresh endpoint tracks
GYD_RPS_MODERATE/STRESS while meta stays at fixed lower rates
(namespace constraints).

Configurable via env vars (see load_testing/.env):
  GYD_META_RPS_MODERATE / GYD_META_RPS_STRESS  (default 5 / 20)

User counts and timing are driven by env vars (see load_testing/.env):
  GYD_RPS_MODERATE       refresh moderate RPS  (default 100)
  GYD_RPS_STRESS         refresh stress RPS    (default 500)
  GYD_RAMP_TIME          ramp duration in s    (default 60)
  GYD_MODERATE_DURATION  moderate hold in s    (default 120)
  GYD_STRESS_HOLD        stress hold in s      (default 90)
"""

import uuid

from locust import constant_throughput

from common import BaseGYDUser, configure_context, load_config, make_shape, safe_login

_cfg = load_config()

_META_MODERATE = _cfg["meta_rps_moderate"]
_META_STRESS   = _cfg["meta_rps_stress"]

_moderate_total = _cfg["rps_moderate"] + _META_MODERATE * 2
_stress_total   = _cfg["rps_stress"]   + _META_STRESS   * 2

configure_context(
    "misc_lightweight",
    moderate_target_rps=_cfg["rps_moderate"],
    stress_target_rps=_cfg["rps_stress"],
    moderate_users=_moderate_total,
    stress_peak_users=_stress_total,
)

GroupShape = make_shape(moderate_total=_moderate_total, stress_total=_stress_total)

_META_FIELD_PREFIX = "loadtest"


class MiscUser(BaseGYDUser):
    """Exercises refresh and meta endpoints concurrently."""

    _test_group = "misc_lightweight"
    wait_time = constant_throughput(1.0)

    def on_start(self):
        super().on_start()
        cfg = load_config()
        pair = safe_login(self.environment, cfg["target_url"], cfg["username"], cfg["password"])
        self._access = pair["access"]
        self._refresh = pair["refresh"]
        self._headers = {"Authorization": f"Bearer {self._access}"}

        admin_pair = safe_login(self.environment, cfg["target_url"], cfg["admin_username"], cfg["admin_password"])
        self._admin_headers = {"Authorization": f"Bearer {admin_pair['access']}"}

        # Create a unique meta field for this user's PUT tests
        self._meta_field = f"{_META_FIELD_PREFIX}_{uuid.uuid4().hex[:8]}"
        self.client.post(
            "/api/v1/meta",
            json={
                "fieldName": self._meta_field,
                "description": "load test field",
                "type": "string",
            },
            headers=self._admin_headers,
            name="/api/v1/meta [setup]",
        )

    def refresh_token(self):
        with self.client.post(
            "/api/v1/auth/refresh",
            json={"refresh_token": self._refresh},
            headers=self._headers,
            name="/api/v1/auth/refresh",
            catch_response=True,
        ) as resp:
            if resp.status_code == 200:
                data = resp.json()
                self._access = data["access_token"]
                self._refresh = data["refresh_token"]
                self._headers = {"Authorization": f"Bearer {self._access}"}
                resp.success()
            else:
                resp.failure(f"refresh failed: HTTP {resp.status_code}")

    def create_meta_field(self):
        """Create a uniquely named meta field (avoids 409 conflicts)."""
        field_name = f"{_META_FIELD_PREFIX}_{uuid.uuid4().hex[:8]}"
        self.client.post(
            "/api/v1/meta",
            json={
                "fieldName": field_name,
                "description": "load test field",
                "type": "string",
            },
            headers=self._admin_headers,
            name="/api/v1/meta [POST]",
        )

    def update_meta_field(self):
        """Update the description of the field created in on_start."""
        self.client.put(
            f"/api/v1/meta/{self._meta_field}",
            json={"description": f"updated at {uuid.uuid4().hex[:4]}"},
            headers=self._admin_headers,
            name="/api/v1/meta/:fieldName [PUT]",
        )


# Set task weights dynamically so refresh tracks GYD_RPS_MODERATE
# while meta stays at its fixed lower rate (namespace constraints).
# Use a repeated list instead of a dict — Locust's post-definition dict
# assignment with method-reference keys does not apply weights correctly.
MiscUser.tasks = (
    [MiscUser.refresh_token]     * _cfg["rps_moderate"] +
    [MiscUser.create_meta_field] * _META_MODERATE +
    [MiscUser.update_meta_field] * _META_MODERATE
)
