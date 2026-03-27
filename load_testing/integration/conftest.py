"""
conftest.py — Shared pytest fixtures for the GatherYourDeals integration test suite.

Reads configuration from load_testing/.env (if present) and environment variables,
using the same GYD_* convention as seed.py and the locust common.py.

Fixtures:
  test_run_id     (session) — unique run ID, injected as X-Test-Run-Id header
  config          (session) — validated config dict (includes test_run_id)
  user_session    (session) — regular user auth tokens; registers user if not exists
  admin_session   (session) — admin auth tokens
  user_token_pair (function) — fresh token pair for tests that consume/invalidate tokens
"""

import os
import datetime

import pytest
import requests
from dotenv import load_dotenv

_ENV_FILE = os.path.join(os.path.dirname(__file__), "..", ".env")
load_dotenv(_ENV_FILE)


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _login(base_url, username, password, label, test_run_id=None):
    """Log in and return a token dict. Calls pytest.exit() on failure."""
    resp = requests.post(
        f"{base_url}/api/v1/auth/login",
        json={"username": username, "password": password},
        timeout=30,
    )
    if resp.status_code != 200:
        pytest.exit(
            f"{label}: login failed for '{username}': "
            f"HTTP {resp.status_code} — {resp.text[:200]}",
            returncode=1,
        )
    data = resp.json()
    headers = {"Authorization": f"Bearer {data['access_token']}"}
    if test_run_id:
        headers["X-Test-Run-Id"] = test_run_id
        headers["X-Test-Group"] = "integration"
        headers["X-Test-Phase"] = "integration"
    return {
        "refresh_token": data["refresh_token"],
        "headers": headers,
    }


# ---------------------------------------------------------------------------
# test_run_id fixture
# ---------------------------------------------------------------------------

@pytest.fixture(scope="session")
def test_run_id():
    """
    Generate a unique test run ID for this session.
    Passed as X-Test-Run-Id header on all authenticated requests so traces
    can be correlated in Honeycomb via the test.run_id span attribute.
    Falls back to GYD_TEST_RUN_ID env var when set by CI (e.g. GitHub Actions).
    """
    env_id = os.environ.get("GYD_TEST_RUN_ID", "")
    if env_id:
        return env_id
    run_id = "integration_" + datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%d_%H%M%S")
    print(f"\n[integration] test_run_id: {run_id}", flush=True)
    return run_id


# ---------------------------------------------------------------------------
# config fixture
# ---------------------------------------------------------------------------

@pytest.fixture(scope="session")
def config(test_run_id):
    """Return validated configuration from environment variables."""
    base_url = os.environ.get("GYD_TARGET_URL", "http://localhost:8080").rstrip("/")
    username = os.environ.get("GYD_TEST_USERNAME", "")
    password = os.environ.get("GYD_TEST_PASSWORD", "")
    admin_username = os.environ.get("GYD_ADMIN_USERNAME", "")
    admin_password = os.environ.get("GYD_ADMIN_PASSWORD", "")

    missing = [
        name for name, val in [
            ("GYD_TEST_USERNAME", username),
            ("GYD_TEST_PASSWORD", password),
            ("GYD_ADMIN_USERNAME", admin_username),
            ("GYD_ADMIN_PASSWORD", admin_password),
        ]
        if not val
    ]
    if missing:
        pytest.exit(
            f"Missing required environment variables: {', '.join(missing)}\n"
            "Set them in load_testing/.env or export them before running.",
            returncode=2,
        )

    return {
        "base_url": base_url,
        "username": username,
        "password": password,
        "admin_username": admin_username,
        "admin_password": admin_password,
        "test_run_id": test_run_id,
    }


# ---------------------------------------------------------------------------
# user_session fixture
# ---------------------------------------------------------------------------

@pytest.fixture(scope="session")
def user_session(config):
    """
    Ensure the test user exists, then log in and return session tokens.

    Registration is idempotent: 409 (already exists) is treated as success so
    the suite can be re-run against the same database without cleanup.
    Aborts the entire test session if login fails.
    """
    base_url = config["base_url"]
    username = config["username"]
    password = config["password"]

    resp = requests.post(
        f"{base_url}/api/v1/users",
        json={"username": username, "password": password},
        timeout=30,
    )
    if resp.status_code not in (201, 409):
        pytest.exit(
            f"user_session: failed to register test user '{username}': "
            f"HTTP {resp.status_code} — {resp.text[:200]}",
            returncode=1,
        )

    return _login(base_url, username, password, "user_session", config["test_run_id"])


# ---------------------------------------------------------------------------
# admin_session fixture
# ---------------------------------------------------------------------------

@pytest.fixture(scope="session")
def admin_session(config):
    """
    Log in as the admin user and return session tokens.

    The admin account must already exist (created via ./gatheryourdeals init).
    Aborts the entire test session if login fails.
    """
    return _login(
        config["base_url"],
        config["admin_username"],
        config["admin_password"],
        "admin_session",
        config["test_run_id"],
    )


# ---------------------------------------------------------------------------
# user_token_pair fixture
# ---------------------------------------------------------------------------

@pytest.fixture(scope="function")
def user_token_pair(config):
    """
    Issue a fresh token pair for tests that consume (invalidate) tokens.

    Using this fixture instead of user_session prevents the logout test from
    invalidating the shared session token used by all other tests.
    """
    base_url = config["base_url"]
    resp = requests.post(
        f"{base_url}/api/v1/auth/login",
        json={"username": config["username"], "password": config["password"]},
        timeout=30,
    )
    assert resp.status_code == 200, (
        f"user_token_pair: login failed: HTTP {resp.status_code} — {resp.text[:200]}"
    )
    data = resp.json()
    return {
        "refresh_token": data["refresh_token"],
        "headers": {
            "Authorization": f"Bearer {data['access_token']}",
            "X-Test-Run-Id": config["test_run_id"],
            "X-Test-Group": "integration",
            "X-Test-Phase": "integration",
        },
    }
