"""
Group 5: Logout (Token-Pool Driven)

POST /auth/logout uses a pre-generated token pool to avoid contaminating
Locust stats with login requests during the test.

The pool is filled concurrently at test_start using a ThreadPoolExecutor so
that startup completes within GYD_SETUP_TIME seconds (default 30 s) even for
large stress-phase pools. If the fill budget is exhausted before all tokens
are ready, the group proceeds with a partial pool and reports the shortfall
via stderr, a named Locust stat entry, and context.json.

Configurable via env vars (see load_testing/.env):
  GYD_LOGOUT_RPS_MODERATE / GYD_LOGOUT_RPS_STRESS   (default 10 / 40)
  GYD_TOKEN_POOL_WORKERS                             (default 300)
  GYD_SETUP_TIME                                     (default 30 s)
  GYD_TOKEN_POOL_HEADROOM                            (default 1.17)

User counts and timing are driven by env vars (see load_testing/.env):
  GYD_LOGOUT_RPS_MODERATE  moderate target RPS  (default 10)
  GYD_LOGOUT_RPS_STRESS    stress target RPS    (default 40)
  GYD_RAMP_TIME            ramp duration in s   (default 60)
  GYD_STRESS_HOLD          stress hold in s     (default 90)
"""

import concurrent.futures
import sys
import time

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

# Token pool covers _LOGOUT_STRESS × (ramp_time + stress_hold) with configured headroom.
_TOKEN_POOL_TARGET = int(
    _LOGOUT_STRESS
    * (_cfg["ramp_time"] + _cfg["stress_hold"])
    * _cfg["token_pool_headroom"]
)

configure_context(
    "logout",
    moderate_target_rps=_LOGOUT_MODERATE,
    stress_target_rps=_LOGOUT_STRESS,
    moderate_users=_LOGOUT_MODERATE,
    stress_peak_users=_LOGOUT_STRESS,
)

GroupShape = make_shape(moderate_total=_LOGOUT_MODERATE, stress_total=_LOGOUT_STRESS)


@events.test_start.add_listener
def _prefill_token_pool_concurrent(environment, **kwargs):
    """
    Pre-generate TOKEN_POOL_TARGET login token pairs using concurrent HTTP requests.
    Runs once before any virtual users are spawned.

    Fills up to GYD_TOKEN_POOL_WORKERS concurrent workers and respects a
    GYD_SETUP_TIME second deadline. Any shortfall is reported via:
      - stderr (immediate visibility)
      - a named Locust stat failure entry (captured in results CSV/HTML)
      - context.json token_pool_stats (via record_fill_metadata + set_context_token_pool_stats)
    """
    cfg = load_config()
    workers    = cfg["token_pool_workers"]
    deadline   = cfg["setup_time"]
    target_url = cfg["target_url"]
    username   = cfg["username"]
    password   = cfg["password"]

    print(
        f"[group5] Pre-filling token pool: {_TOKEN_POOL_TARGET} tokens "
        f"({workers} workers, {deadline}s budget) …",
        file=sys.stderr,
    )

    fill_start = time.monotonic()
    filled = 0
    failures = 0

    def _do_login(_):
        return login(target_url, username, password)

    with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as executor:
        futures = {
            executor.submit(_do_login, i): i
            for i in range(_TOKEN_POOL_TARGET)
        }
        done, not_done = concurrent.futures.wait(
            futures, timeout=deadline
        )

        # Cancel remaining futures that haven't started yet
        for f in not_done:
            f.cancel()

        for f in done:
            try:
                pair = f.result()
                token_pool.put(pair)
                filled += 1
            except Exception as exc:
                failures += 1
                if failures <= 5:
                    print(
                        f"[group5] token pool fill failure #{failures}: {exc}",
                        file=sys.stderr,
                    )

    elapsed = time.monotonic() - fill_start
    shortfall = _TOKEN_POOL_TARGET - filled

    token_pool.record_fill_metadata(targeted=_TOKEN_POOL_TARGET, fill_time_s=elapsed)

    if shortfall > 0:
        print(
            f"[group5] WARNING: fill shortfall — {filled}/{_TOKEN_POOL_TARGET} tokens "
            f"filled in {elapsed:.1f}s ({failures} login failures). "
            f"Proceeding with partial pool.",
            file=sys.stderr,
        )
        environment.events.request.fire(
            request_type="token_pool",
            name="fill_shortfall",
            response_time=0,
            response_length=0,
            exception=Exception(
                f"fill shortfall: {filled}/{_TOKEN_POOL_TARGET} tokens in {elapsed:.1f}s"
            ),
            context={},
        )
    else:
        print(
            f"[group5] Token pool ready: {filled} tokens filled in {elapsed:.1f}s "
            f"({failures} failures during fill)",
            file=sys.stderr,
        )


@events.test_stop.add_listener
def _record_token_pool_stats(environment, **kwargs):
    """Snapshot pool stats into context data before the quitting listener writes the file."""
    set_context_token_pool_stats(token_pool.stats())


class LogoutUser(BaseGYDUser):
    """Consumes one pre-generated token pair per task and logs it out."""

    _test_group = "logout"
    wait_time = constant_throughput(1.0)

    def logout(self):
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


LogoutUser.tasks = {LogoutUser.logout: 1}
