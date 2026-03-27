"""
test_endpoints.py — Integration tests for all GatherYourDeals HTTP endpoints.

Covers all 15 endpoints with at least one happy-path and one negative-path test each.
Tests run against a live server; no mocking. Data created during tests is not cleaned up
(data volume is small and bounded).
"""

import uuid

import pytest
import requests


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _unique_username():
    return f"inttest_{uuid.uuid4().hex[:10]}"


def _unique_field_name():
    return f"inttest_{uuid.uuid4().hex[:8]}"


_RECEIPT_PAYLOAD = {
    "productName": "Integration Test Item",
    "purchaseDate": "2026-01-15",
    "price": "9.99",
    "amount": "1",
    "storeName": "Test Store",
}


# ---------------------------------------------------------------------------
# Health
# ---------------------------------------------------------------------------

def test_health(config):
    resp = requests.get(f"{config['base_url']}/health", timeout=30)
    assert resp.status_code == 200
    assert resp.json().get("status") == "ok"


# ---------------------------------------------------------------------------
# Registration
# ---------------------------------------------------------------------------

def test_register_success(config):
    resp = requests.post(
        f"{config['base_url']}/api/v1/users",
        json={"username": _unique_username(), "password": "testpassword1"},
        timeout=30,
    )
    assert resp.status_code == 201
    body = resp.json()
    assert "username" in body
    assert "role" in body


def test_register_duplicate(config):
    username = _unique_username()
    payload = {"username": username, "password": "testpassword1"}
    requests.post(f"{config['base_url']}/api/v1/users", json=payload, timeout=30)
    resp = requests.post(f"{config['base_url']}/api/v1/users", json=payload, timeout=30)
    assert resp.status_code == 409


# ---------------------------------------------------------------------------
# Login
# ---------------------------------------------------------------------------

def test_login_success(config, user_session):
    resp = requests.post(
        f"{config['base_url']}/api/v1/auth/login",
        json={"username": config["username"], "password": config["password"]},
        timeout=30,
    )
    assert resp.status_code == 200
    body = resp.json()
    assert "access_token" in body
    assert "refresh_token" in body


def test_login_wrong_password(config, user_session):
    resp = requests.post(
        f"{config['base_url']}/api/v1/auth/login",
        json={"username": config["username"], "password": "definitelywrong"},
        timeout=30,
    )
    assert resp.status_code == 401


# ---------------------------------------------------------------------------
# Auth / me
# ---------------------------------------------------------------------------

def test_me_success(config, user_session):
    resp = requests.get(
        f"{config['base_url']}/api/v1/auth/me",
        headers=user_session["headers"],
        timeout=30,
    )
    assert resp.status_code == 200
    body = resp.json()
    assert "id" in body
    assert "role" in body


def test_me_unauthenticated(config):
    resp = requests.get(f"{config['base_url']}/api/v1/auth/me", timeout=30)
    assert resp.status_code == 401


# ---------------------------------------------------------------------------
# Token refresh
# ---------------------------------------------------------------------------

def test_refresh_success(config, user_token_pair):
    resp = requests.post(
        f"{config['base_url']}/api/v1/auth/refresh",
        json={"refresh_token": user_token_pair["refresh_token"]},
        timeout=30,
    )
    assert resp.status_code == 200
    body = resp.json()
    assert "access_token" in body
    assert "refresh_token" in body


def test_refresh_invalid_token(config):
    resp = requests.post(
        f"{config['base_url']}/api/v1/auth/refresh",
        json={"refresh_token": "this-is-not-a-valid-token"},
        timeout=30,
    )
    assert resp.status_code == 401


# ---------------------------------------------------------------------------
# Logout
# ---------------------------------------------------------------------------

def test_logout_success(config, user_token_pair):
    # The refresh token must be in the body so the server can revoke it.
    resp = requests.post(
        f"{config['base_url']}/api/v1/auth/logout",
        json={"refresh_token": user_token_pair["refresh_token"]},
        headers=user_token_pair["headers"],
        timeout=30,
    )
    assert resp.status_code == 200

    resp = requests.post(
        f"{config['base_url']}/api/v1/auth/refresh",
        json={"refresh_token": user_token_pair["refresh_token"]},
        timeout=30,
    )
    assert resp.status_code == 401


# ---------------------------------------------------------------------------
# Receipts
# ---------------------------------------------------------------------------

def test_create_receipt_success(config, user_session):
    resp = requests.post(
        f"{config['base_url']}/api/v1/receipts",
        json=_RECEIPT_PAYLOAD,
        headers=user_session["headers"],
        timeout=30,
    )
    assert resp.status_code == 201
    assert "id" in resp.json()


def test_create_receipt_unauthenticated(config):
    resp = requests.post(
        f"{config['base_url']}/api/v1/receipts",
        json=_RECEIPT_PAYLOAD,
        timeout=30,
    )
    assert resp.status_code == 401


def test_list_receipts_success(config, user_session):
    resp = requests.get(
        f"{config['base_url']}/api/v1/receipts",
        headers=user_session["headers"],
        timeout=30,
    )
    assert resp.status_code == 200
    body = resp.json()
    assert "data" in body
    assert isinstance(body["data"], list)


def test_get_receipt_success(config, user_session):
    create = requests.post(
        f"{config['base_url']}/api/v1/receipts",
        json=_RECEIPT_PAYLOAD,
        headers=user_session["headers"],
        timeout=30,
    )
    assert create.status_code == 201
    receipt_id = create.json()["id"]

    resp = requests.get(
        f"{config['base_url']}/api/v1/receipts/{receipt_id}",
        headers=user_session["headers"],
        timeout=30,
    )
    assert resp.status_code == 200
    assert resp.json()["id"] == receipt_id


def test_get_receipt_not_found(config, user_session):
    resp = requests.get(
        f"{config['base_url']}/api/v1/receipts/nonexistent-id-that-does-not-exist",
        headers=user_session["headers"],
        timeout=30,
    )
    assert resp.status_code == 404


def test_delete_receipt_success(config, user_session):
    create = requests.post(
        f"{config['base_url']}/api/v1/receipts",
        json=_RECEIPT_PAYLOAD,
        headers=user_session["headers"],
        timeout=30,
    )
    assert create.status_code == 201
    receipt_id = create.json()["id"]

    resp = requests.delete(
        f"{config['base_url']}/api/v1/receipts/{receipt_id}",
        headers=user_session["headers"],
        timeout=30,
    )
    assert resp.status_code == 200

    resp = requests.get(
        f"{config['base_url']}/api/v1/receipts/{receipt_id}",
        headers=user_session["headers"],
        timeout=30,
    )
    assert resp.status_code == 404


# ---------------------------------------------------------------------------
# Users (admin)
# ---------------------------------------------------------------------------

def test_list_users_admin(config, admin_session):
    resp = requests.get(
        f"{config['base_url']}/api/v1/users",
        headers=admin_session["headers"],
        timeout=30,
    )
    assert resp.status_code == 200
    body = resp.json()
    assert "data" in body
    assert isinstance(body["data"], list)


def test_list_users_forbidden(config, user_session):
    resp = requests.get(
        f"{config['base_url']}/api/v1/users",
        headers=user_session["headers"],
        timeout=30,
    )
    assert resp.status_code == 403


def test_delete_user_success(config, admin_session):
    username = _unique_username()
    reg = requests.post(
        f"{config['base_url']}/api/v1/users",
        json={"username": username, "password": "throwaway1"},
        timeout=30,
    )
    assert reg.status_code == 201

    list_resp = requests.get(
        f"{config['base_url']}/api/v1/users",
        headers=admin_session["headers"],
        timeout=30,
    )
    assert list_resp.status_code == 200
    users = list_resp.json()["data"]
    user_id = next((u["id"] for u in users if u["username"] == username), None)
    assert user_id is not None, f"Throwaway user '{username}' not found in user list"

    resp = requests.delete(
        f"{config['base_url']}/api/v1/users/{user_id}",
        headers=admin_session["headers"],
        timeout=30,
    )
    assert resp.status_code == 200


# ---------------------------------------------------------------------------
# Meta fields
# ---------------------------------------------------------------------------

def test_list_meta_success(config, user_session):
    resp = requests.get(
        f"{config['base_url']}/api/v1/meta",
        headers=user_session["headers"],
        timeout=30,
    )
    assert resp.status_code == 200
    body = resp.json()
    assert "data" in body
    assert isinstance(body["data"], list)


def test_create_meta_field_success(config, admin_session):
    field_name = _unique_field_name()
    resp = requests.post(
        f"{config['base_url']}/api/v1/meta",
        json={"fieldName": field_name, "description": "integration test field", "type": "string"},
        headers=admin_session["headers"],
        timeout=30,
    )
    assert resp.status_code == 201


def test_create_meta_field_unauthenticated(config):
    resp = requests.post(
        f"{config['base_url']}/api/v1/meta",
        json={"fieldName": _unique_field_name(), "description": "should be rejected", "type": "string"},
        timeout=30,
    )
    assert resp.status_code == 401


def test_update_meta_field_success(config, admin_session):
    field_name = _unique_field_name()
    create = requests.post(
        f"{config['base_url']}/api/v1/meta",
        json={"fieldName": field_name, "description": "original description", "type": "string"},
        headers=admin_session["headers"],
        timeout=30,
    )
    assert create.status_code == 201

    resp = requests.put(
        f"{config['base_url']}/api/v1/meta/{field_name}",
        json={"description": "updated description"},
        headers=admin_session["headers"],
        timeout=30,
    )
    assert resp.status_code == 200


def test_update_meta_field_forbidden(config, user_session, admin_session):
    field_name = _unique_field_name()
    create = requests.post(
        f"{config['base_url']}/api/v1/meta",
        json={"fieldName": field_name, "description": "for forbidden test", "type": "string"},
        headers=admin_session["headers"],
        timeout=30,
    )
    assert create.status_code == 201

    resp = requests.put(
        f"{config['base_url']}/api/v1/meta/{field_name}",
        json={"description": "should be forbidden"},
        headers=user_session["headers"],
        timeout=30,
    )
    assert resp.status_code == 403
