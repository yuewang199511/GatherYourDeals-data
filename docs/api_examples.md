# API Usage Examples

All examples use `curl` against a local server running on port 8080.

## 1. Register a user

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"username": "alice", "password": "password123", "clientId": "gatheryourdeals"}'
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "username": "alice",
  "role": "user"
}
```

## 2. Login (get access token)

```bash
curl -X POST http://localhost:8080/api/v1/oauth/token \
  -d "grant_type=password&username=alice&password=password123&client_id=gatheryourdeals"
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "dGhpcyBpcyBhIHJlZnJl..."
}
```

## 3. Use the access token

Include the token in the `Authorization` header for any protected endpoint:

```bash
curl -H "Authorization: Bearer <access_token>" \
  http://localhost:8080/api/v1/some-protected-endpoint
```

## 4. Refresh the token

```bash
curl -X POST http://localhost:8080/api/v1/oauth/token \
  -d "grant_type=refresh_token&refresh_token=<refresh_token>&client_id=gatheryourdeals"
```

## 5. Logout (delete session)

```bash
curl -X DELETE http://localhost:8080/api/v1/oauth/sessions \
  -H "Authorization: Bearer <access_token>"
```

Response:
```json
{
  "message": "logged out"
}
```

## 6. Create an OAuth2 client (admin only)

```bash
curl -X POST http://localhost:8080/api/v1/admin/clients \
  -H "Authorization: Bearer <admin_access_token>" \
  -H "Content-Type: application/json" \
  -d '{"id": "mobile-app", "secret": "my-secret", "domain": "https://mobile.example.com"}'
```

Response:
```json
{
  "id": "mobile-app",
  "domain": "https://mobile.example.com",
  "createdAt": "2026-02-16T12:00:00Z"
}
```

## 7. List all OAuth2 clients (admin only)

```bash
curl -H "Authorization: Bearer <admin_access_token>" \
  http://localhost:8080/api/v1/admin/clients
```

Response:
```json
[
  {
    "id": "gatheryourdeals",
    "domain": "http://localhost",
    "createdAt": "2026-02-16T10:00:00Z"
  },
  {
    "id": "mobile-app",
    "domain": "https://mobile.example.com",
    "createdAt": "2026-02-16T12:00:00Z"
  }
]
```

## 8. Revoke an OAuth2 client (admin only)

```bash
curl -X DELETE http://localhost:8080/api/v1/admin/clients/mobile-app \
  -H "Authorization: Bearer <admin_access_token>"
```

Response:
```json
{
  "message": "client revoked"
}
```
