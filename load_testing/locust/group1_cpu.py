"""
Group 1: CPU-Bound (Solo) — POST /api/v1/auth/login

Runs alone because bcrypt password hashing saturates CPU and would skew
all other endpoints if run concurrently.

User counts and timing are driven by env vars (see load_testing/.env):
  GYD_RPS_MODERATE       moderate target RPS  (default 100)
  GYD_RPS_STRESS         stress peak RPS      (default 500)
  GYD_RAMP_TIME          ramp duration in s   (default 60)
  GYD_MODERATE_DURATION  moderate hold in s   (default 120)
  GYD_STRESS_HOLD        stress hold in s     (default 90)
"""

from locust import constant_throughput, task

from common import BaseGYDUser, configure_context, load_config, make_shape

_cfg = load_config()

configure_context(
    "cpu_bound",
    moderate_target_rps=_cfg["rps_moderate"],
    stress_target_rps=_cfg["rps_stress"],
    moderate_users=_cfg["rps_moderate"],
    stress_peak_users=_cfg["rps_stress"],
)

# 1 endpoint → 1× multiplier: moderate_users = GYD_RPS_MODERATE
GroupShape = make_shape(endpoint_multiplier=1)


class LoginUser(BaseGYDUser):
    """Hammers POST /api/v1/auth/login with bcrypt-hashed credentials."""

    _test_group = "cpu_bound"
    wait_time = constant_throughput(1.0)

    def on_start(self):
        super().on_start()
        cfg = load_config()
        self._username = cfg["username"]
        self._password = cfg["password"]

    @task
    def login(self):
        self.client.post(
            "/api/v1/auth/login",
            json={"username": self._username, "password": self._password},
            name="/api/v1/auth/login",
        )
