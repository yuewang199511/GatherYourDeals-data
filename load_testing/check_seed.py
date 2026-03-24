#!/usr/bin/env python3
"""
check_seed.py — Verify the database has been seeded with enough receipts.

Exits 0 if the receipt count is at or above GYD_SEED_COUNT (default 1000).
Exits 1 otherwise, printing a message to stderr.

Usage:
    python3 check_seed.py
"""

import os
import sys

import requests
from dotenv import load_dotenv

load_dotenv(os.path.join(os.path.dirname(__file__), ".env"))

base_url = os.environ.get("GYD_TARGET_URL", "http://localhost:8080").rstrip("/")
username = os.environ.get("GYD_TEST_USERNAME", "")
password = os.environ.get("GYD_TEST_PASSWORD", "")
seed_count = int(os.environ.get("GYD_SEED_COUNT", "1000"))

if not username or not password:
    print("ERROR: GYD_TEST_USERNAME and GYD_TEST_PASSWORD must be set.", file=sys.stderr)
    sys.exit(2)

# Log in
resp = requests.post(f"{base_url}/api/v1/auth/login", json={"username": username, "password": password}, timeout=10)
if resp.status_code != 200:
    print(f"ERROR: login failed: HTTP {resp.status_code} — {resp.text[:200]}", file=sys.stderr)
    sys.exit(2)

token = resp.json()["access_token"]

# Check receipt count
resp = requests.get(f"{base_url}/api/v1/receipts", headers={"Authorization": f"Bearer {token}"}, timeout=10)
if resp.status_code != 200:
    print(f"ERROR: failed to list receipts: HTTP {resp.status_code} — {resp.text[:200]}", file=sys.stderr)
    sys.exit(2)

actual = resp.json().get("total", 0)
if actual < seed_count:
    print(
        f"ERROR: Expected at least {seed_count} receipts (GYD_SEED_COUNT), got {actual}. "
        "Re-run seed.py to create the missing receipts.",
        file=sys.stderr,
    )
    sys.exit(1)

print(f"OK: {actual} receipts found (>= {seed_count}).")
