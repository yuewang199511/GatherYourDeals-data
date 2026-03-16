#!/usr/bin/env python3
"""
seed.py — Create the load test user and seed 1,000 purchase receipts.

The service must already be running before this script is executed.
Reads GYD_* env vars from load_testing/.env automatically (python-dotenv).

No local PostgreSQL seeding provided — SQLite snapshots are simpler for local use.

Setup (first time only):
    cd load_testing
    python3 -m venv .venv
    .venv/bin/pip install -r requirements.txt

Usage:
    cd load_testing
    cp .env.example .env        # fill in GYD_TEST_USERNAME + GYD_TEST_PASSWORD
    .venv/bin/python3 seed.py

    # Or activate the venv first:
    source .venv/bin/activate
    python3 seed.py

After seeding, save the database as a local snapshot so you can restore it
before each load test run without re-seeding:

    # SQLite local binary:
    cp ../gatheryourdeals.db seed_snapshot.db

    # SQLite via Docker Compose (from the repo root):
    cp data/db/gatheryourdeals.db load_testing/seed_snapshot.db

To restore before a test run:
    cp load_testing/seed_snapshot.db data/db/gatheryourdeals.db  # Docker
    cp load_testing/seed_snapshot.db gatheryourdeals.db           # local binary

Note: seed_snapshot.db is covered by the root .gitignore (*.db pattern) and
will not be committed.

Optional env vars:
    GYD_SEED_COUNT    number of receipts to create (default: 1000)
    GYD_SEED_WORKERS  concurrent HTTP workers      (default: 10)
"""

import os
import sys
import time
from concurrent.futures import ThreadPoolExecutor, as_completed

import requests
from dotenv import load_dotenv

load_dotenv(os.path.join(os.path.dirname(__file__), ".env"))

BASE_URL = os.environ.get("GYD_TARGET_URL", "http://localhost:8080").rstrip("/")
USERNAME = os.environ.get("GYD_TEST_USERNAME", "")
PASSWORD = os.environ.get("GYD_TEST_PASSWORD", "")
NUM_RECEIPTS = int(os.environ.get("GYD_SEED_COUNT", "1000"))
WORKERS = int(os.environ.get("GYD_SEED_WORKERS", "10"))

_STORES = ["Supermart", "TechStore", "FreshGrocer", "BookWorld", "CafePrime"]
_PRODUCTS = ["Laptop", "Coffee", "Notebook", "Groceries", "Headphones", "Monitor", "Keyboard"]


def _register(username: str, password: str) -> None:
    resp = requests.post(
        f"{BASE_URL}/api/v1/users",
        json={"username": username, "password": password},
        timeout=10,
    )
    if resp.status_code == 201:
        print(f"  Created user '{username}'")
    elif resp.status_code == 409:
        print(f"  User '{username}' already exists — continuing")
    else:
        print(f"ERROR: register failed: HTTP {resp.status_code} — {resp.text[:200]}", file=sys.stderr)
        sys.exit(1)


def _login(username: str, password: str) -> str:
    resp = requests.post(
        f"{BASE_URL}/api/v1/auth/login",
        json={"username": username, "password": password},
        timeout=10,
    )
    if resp.status_code != 200:
        print(f"ERROR: login failed: HTTP {resp.status_code} — {resp.text[:200]}", file=sys.stderr)
        sys.exit(1)
    return resp.json()["access_token"]


def _post_receipt(session: requests.Session, token: str, index: int) -> bool:
    payload = {
        "productName": f"{_PRODUCTS[index % len(_PRODUCTS)]} #{index}",
        "purchaseDate": f"2026-{(index % 12) + 1:02d}-{(index % 28) + 1:02d}",
        "price": f"{9.99 + (index % 90):.2f}",
        "amount": str(1 + (index % 5)),
        "storeName": _STORES[index % len(_STORES)],
    }
    resp = session.post(
        f"{BASE_URL}/api/v1/receipts",
        json=payload,
        headers={"Authorization": f"Bearer {token}"},
        timeout=10,
    )
    return resp.status_code == 201


def main() -> None:
    if not USERNAME or not PASSWORD:
        print(
            "ERROR: GYD_TEST_USERNAME and GYD_TEST_PASSWORD must be set.\n"
            "       Source load_testing/.env or export them in your shell.",
            file=sys.stderr,
        )
        sys.exit(1)

    print(f"Target  : {BASE_URL}")
    print(f"User    : {USERNAME}")
    print(f"Receipts: {NUM_RECEIPTS}")
    print(f"Workers : {WORKERS}")
    print()

    print("Step 1/3 — Register test user (idempotent)")
    _register(USERNAME, PASSWORD)

    print("Step 2/3 — Login")
    token = _login(USERNAME, PASSWORD)
    print("  OK — access token acquired")

    print(f"Step 3/3 — Seeding {NUM_RECEIPTS} receipts …")
    start = time.time()
    success = 0
    failure = 0

    with requests.Session() as session:
        with ThreadPoolExecutor(max_workers=WORKERS) as pool:
            futures = {
                pool.submit(_post_receipt, session, token, i): i
                for i in range(1, NUM_RECEIPTS + 1)
            }
            for n, future in enumerate(as_completed(futures), 1):
                if future.result():
                    success += 1
                else:
                    failure += 1
                if n % 100 == 0 or n == NUM_RECEIPTS:
                    elapsed = time.time() - start
                    rps = n / elapsed if elapsed > 0 else 0
                    print(f"  {n:>4}/{NUM_RECEIPTS}  {rps:5.0f} req/s  ✓ {success}  ✗ {failure}")

    elapsed = time.time() - start
    print(f"\nDone in {elapsed:.1f} s — {success} receipts created, {failure} failed")

    if failure > 0:
        print("WARNING: some receipts failed to create — check the output above.", file=sys.stderr)
        sys.exit(1)

    print(
        "\nNext steps — save a snapshot so you can restore before each test run:\n"
        "\n  # Docker Compose (from repo root):\n"
        "  cp data/db/gatheryourdeals.db load_testing/seed_snapshot.db\n"
        "\n  # Local binary (from repo root):\n"
        "  cp gatheryourdeals.db load_testing/seed_snapshot.db\n"
        "\n  # Restore before a test run (Docker):\n"
        "  cp load_testing/seed_snapshot.db data/db/gatheryourdeals.db\n"
    )


if __name__ == "__main__":
    main()
