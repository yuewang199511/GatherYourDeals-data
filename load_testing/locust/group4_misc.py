"""
Group 4: Lightweight Misc (Concurrent) — four low-resource endpoints.

POST /auth/logout uses a pre-generated token pool to avoid contaminating
Locust stats with login requests during the test.

Task weights are set dynamically from config so the refresh endpoint tracks
GYD_RPS_MODERATE/STRESS while logout and meta stay at fixed lower rates
(token pool and namespace constraints).

Configurable via env vars (see load_testing/.env):
  GYD_LOGOUT_RPS_MODERATE / GYD_LOGOUT_RPS_STRESS   (default 10 / 40)
  GYD_META_RPS_MODERATE   / GYD_META_RPS_STRESS     (default 5  / 20)

User counts and timing are driven by env vars (see load_testing/.env):
  GYD_RPS_MODERATE       refresh moderate RPS  (default 100)
  GYD_RPS_STRESS         refresh stress RPS    (default 500)
  GYD_RAMP_TIME          ramp duration in s    (default 60)
  GYD_MODERATE_DURATION  moderate hold in s    (default 120)
  GYD_STRESS_HOLD        stress hold in s      (default 90)
"""

import sys
import uuid

from locust import constant_throughput, events

from common import (
    BaseGYDUser,
    configure_context,
    load_config,
    login,
    make_shape,
    set_context_token_pool_stats,
    token_pool,
)

_cfg = load_config()

_LOGOUT_MODERATE = _cfg["logout_rps_moderate"]
_LOGOUT_STRESS   = _cfg["logout_rps_stress"]
_META_MODERATE   = _cfg["meta_rps_moderate"]
_META_STRESS     = _cfg["meta_rps_stress"]

# Token pool covers _LOGOUT_STRESS × (ramp_time + stress_hold) with ~17% headroom.
_TOKEN_POOL_TARGET = int(_LOGOUT_STRESS * (_cfg["ramp_time"] + _cfg["stress_hold"]) * 1.17)

_moderate_total = _cfg["rps_moderate"] + _LOGOUT_MODERATE + _META_MODERATE * 2
_stress_total   = _cfg["rps_stress"]   + _LOGOUT_STRESS   + _META_STRESS   * 2

configure_context(
    "misc_lightweight",
    moderate_target_rps=_cfg["rps_moderate"],
    stress_target_rps=_cfg["rps_stress"],
    moderate_users=_moderate_total,
    stress_peak_users=_stress_total,
)

GroupShape = make_shape(moderate_total=_moderate_total, stress_total=_stress_total)

_META_FIELD_PREFIX = "loadtest"


@events.test_start.add_listener
def _prefill_token_pool(environment, **kwargs):
    """
    Pre-generate TOKEN_POOL_TARGET login token pairs using plain HTTP.
    Runs once before any virtual users are spawned.
    """
    cfg = load_config()
    print(
        f"[group4] Pre-filling token pool with {_TOKEN_POOL_TARGET} token pairs …",
        file=sys.stderr,
    )
    failures = 0
    for i in range(_TOKEN_POOL_TARGET):
        try:
            pair = login(cfg["target_url"], cfg["username"], cfg["password"])
            token_pool.put(pair)
        except Exception as exc:
            failures += 1
            if failures <= 5:
                print(f"[group4] token pool fill failure #{failures}: {exc}", file=sys.stderr)
            if failures > 100:
                print(
                    "[group4] WARNING: >100 login failures during pool fill — aborting early",
                    file=sys.stderr,
                )
                break
    print(
        f"[group4] Token pool ready: {token_pool.size()} pairs "
        f"({failures} failures during fill)",
        file=sys.stderr,
    )


@events.test_stop.add_listener
def _record_token_pool_stats(environment, **kwargs):
    """Snapshot pool stats into context data before the quitting listener writes the file."""
    set_context_token_pool_stats(token_pool.stats())


class MiscUser(BaseGYDUser):
    """Exercises refresh, logout, and meta endpoints concurrently."""

    _test_group = "misc_lightweight"
    wait_time = constant_throughput(1.0)

    def on_start(self):
        super().on_start()
        cfg = load_config()
        pair = login(cfg["target_url"], cfg["username"], cfg["password"])
        self._access = pair["access"]
        self._refresh = pair["refresh"]
        self._headers = {"Authorization": f"Bearer {self._access}"}

        # Create a unique meta field for this user's PUT tests
        self._meta_field = f"{_META_FIELD_PREFIX}_{uuid.uuid4().hex[:8]}"
        self.client.post(
            "/api/v1/meta",
            json={
                "fieldName": self._meta_field,
                "description": "load test field",
                "type": "string",
            },
            headers=self._headers,
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

    def logout(self):
        """Consume one pre-generated token pair and log it out."""
        pair = token_pool.get()
        if pair is None:
            self.environment.events.request.fire(
                request_type="token_pool",
                name="pool_exhausted",
                response_time=0,
                response_length=0,
                exception=Exception("token pool exhausted"),
                context={},
            )
            return
        self.client.post(
            "/api/v1/auth/logout",
            json={"refresh_token": pair["refresh"]},
            headers={"Authorization": f"Bearer {pair['access']}"},
            name="/api/v1/auth/logout",
        )

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
            headers=self._headers,
            name="/api/v1/meta [POST]",
        )

    def update_meta_field(self):
        """Update the description of the field created in on_start."""
        self.client.put(
            f"/api/v1/meta/{self._meta_field}",
            json={"description": f"updated at {uuid.uuid4().hex[:4]}"},
            headers=self._headers,
            name="/api/v1/meta/:fieldName [PUT]",
        )


# Set task weights dynamically so refresh tracks GYD_RPS_MODERATE
# while logout/meta stay at their fixed lower rates.
MiscUser.tasks = {
    MiscUser.refresh_token:    _cfg["rps_moderate"],
    MiscUser.logout:           _LOGOUT_MODERATE,
    MiscUser.create_meta_field: _META_MODERATE,
    MiscUser.update_meta_field: _META_MODERATE,
}
