# API Usage Examples

All examples use `curl` against a local server running on port 8080.

## 1. Register a user

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"username": "alice", "password": "password123"}'
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "alice",
  "role": "user"
}
```

## 2. Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "alice", "password": "password123"}'
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer"
}
```

Store both tokens. Use the `access_token` in the `Authorization` header for requests. Use the `refresh_token` to get a new access token when it expires.

## 3. Use the access token

Include the token in the `Authorization` header for any protected endpoint:

```bash
curl -H "Authorization: Bearer <access_token>" \
  http://localhost:8080/api/v1/auth/me
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "role": "user"
}
```

## 4. Refresh the access token

When the access token expires (default 1 hour), exchange the refresh token for a new pair:

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "<refresh_token>"}'
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer"
}
```

The old refresh token is now invalid. Store the new pair.

## 5. Logout

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "<refresh_token>"}'
```

Response:
```json
{
  "message": "logged out"
}
```

The refresh token is immediately revoked. The access token will expire on its own.

## 6. List all users (admin only)

```bash
curl -H "Authorization: Bearer <admin_access_token>" \
  http://localhost:8080/api/v1/admin/users
```

Response:
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "admin",
    "role": "admin",
    "createdAt": 1739700000,
    "updatedAt": 1739700000
  },
  {
    "id": "661f9511-f30c-52e5-b827-557766551111",
    "username": "alice",
    "role": "user",
    "createdAt": 1739707200,
    "updatedAt": 1739707200
  }
]
```

## 7. Delete a user (admin only)

```bash
curl -X DELETE http://localhost:8080/api/v1/admin/users/661f9511-f30c-52e5-b827-557766551111 \
  -H "Authorization: Bearer <admin_access_token>"
```

Response:
```json
{
  "message": "user deleted"
}
```

All active refresh tokens for that user are immediately revoked.

---

## Python client example

This is a reference implementation for the future Python SDK. It uses
[`httpx`](https://www.python-httpx.org/) and handles token refresh automatically
via a custom `Auth` class — the refresh logic runs transparently on every 401
response so the rest of your code never thinks about token management.

Install the dependency:

```bash
pip install httpx
```

```python
import httpx


class GatherYourDealsAuth(httpx.Auth):
    """
    httpx Auth flow that handles JWT access + refresh token lifecycle.

    - Attaches the access token to every request.
    - On a 401 response, transparently refreshes the token pair and retries once.
    - On a failed refresh, raises GYDAuthError so the caller can re-login.
    """

    def __init__(self, base_url: str, username: str, password: str):
        self._base_url = base_url.rstrip("/")
        self._username = username
        self._password = password
        self._access_token: str | None = None
        self._refresh_token: str | None = None

    # ---- httpx.Auth interface ----

    def auth_flow(self, request: httpx.Request):
        if self._access_token is None:
            self._login()

        request.headers["Authorization"] = f"Bearer {self._access_token}"
        response = yield request

        if response.status_code == 401:
            self._refresh()
            request.headers["Authorization"] = f"Bearer {self._access_token}"
            yield request  # retry once with the new token

    # ---- internal helpers ----

    def _login(self) -> None:
        with httpx.Client() as client:
            resp = client.post(
                f"{self._base_url}/api/v1/auth/login",
                json={"username": self._username, "password": self._password},
            )
        if resp.status_code != 200:
            raise GYDAuthError(f"Login failed: {resp.status_code} {resp.text}")
        data = resp.json()
        self._access_token = data["access_token"]
        self._refresh_token = data["refresh_token"]

    def _refresh(self) -> None:
        with httpx.Client() as client:
            resp = client.post(
                f"{self._base_url}/api/v1/auth/refresh",
                json={"refresh_token": self._refresh_token},
            )
        if resp.status_code != 200:
            # Refresh token expired or revoked — caller must re-login
            self._access_token = None
            self._refresh_token = None
            raise GYDAuthError("Session expired, please log in again.")
        data = resp.json()
        self._access_token = data["access_token"]
        self._refresh_token = data["refresh_token"]

    def logout(self) -> None:
        if self._access_token is None:
            return
        with httpx.Client() as client:
            client.post(
                f"{self._base_url}/api/v1/auth/logout",
                headers={"Authorization": f"Bearer {self._access_token}"},
                json={"refresh_token": self._refresh_token},
            )
        self._access_token = None
        self._refresh_token = None


class GYDAuthError(Exception):
    pass


# ---- usage ----

auth = GatherYourDealsAuth(
    base_url="http://localhost:8080",
    username="alice",
    password="password123",
)

with httpx.Client(auth=auth) as client:
    # Login happens automatically on the first request.
    # Token refresh happens automatically on any 401.
    resp = client.get("http://localhost:8080/api/v1/auth/me")
    print(resp.json())

# Logout explicitly when done
auth.logout()
```

Key design decisions for the SDK:

- **Lazy login** — credentials are not sent until the first actual request, avoiding unnecessary network calls on construction.
- **Single retry** — on a 401, refresh and retry exactly once. A second 401 after refresh means the server genuinely rejected the request (e.g. wrong permissions), not an expiry issue.
- **Stateless from the caller's perspective** — the caller just creates the `auth` object and passes it to `httpx.Client`. Token management is invisible.
- **Explicit logout** — the caller calls `auth.logout()` to revoke the refresh token. Letting the object go out of scope without logging out leaves the refresh token alive until it expires naturally.
