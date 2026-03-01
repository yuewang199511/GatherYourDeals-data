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

## 6. List all fields (meta)

Any authenticated user can list the registered fields:

```bash
curl -H "Authorization: Bearer <access_token>" \
  http://localhost:8080/api/v1/meta
```

Response:
```json
[
  {
    "fieldName": "productName",
    "description": "name of the product",
    "type": "string",
    "native": true
  },
  {
    "fieldName": "brand",
    "description": "brand of the product",
    "type": "string",
    "native": false
  }
]
```

## 7. Register a new field

Any authenticated user can register a new field:

```bash
curl -X POST http://localhost:8080/api/v1/meta \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"fieldName": "brand", "description": "brand of the product", "type": "string"}'
```

Response:
```json
{
  "fieldName": "brand",
  "description": "brand of the product",
  "type": "string",
  "native": false
}
```

## 8. Update a field description (admin only)

```bash
curl -X PUT http://localhost:8080/api/v1/meta/brand \
  -H "Authorization: Bearer <admin_access_token>" \
  -H "Content-Type: application/json" \
  -d '{"description": "brand or manufacturer of the product"}'
```

Response:
```json
{
  "message": "description updated"
}
```

## 9. Create a receipt

```bash
curl -X POST http://localhost:8080/api/v1/receipts \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "productName": "Milk 2%",
    "purchaseDate": "2025.04.05",
    "price": "5.49CAD",
    "amount": "1",
    "storeName": "Costco",
    "latitude": 49.2827,
    "longitude": -123.1207,
    "extras": {"brand": "Kirkland"}
  }'
```

Response:
```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "productName": "Milk 2%",
  "purchaseDate": "2025.04.05",
  "price": "5.49CAD",
  "amount": "1",
  "storeName": "Costco",
  "latitude": 49.2827,
  "longitude": -123.1207,
  "extras": {"brand": "Kirkland"},
  "uploadTime": 1770620311,
  "userId": "550e8400-e29b-41d4-a716-446655440000"
}
```

The server sets `id`, `uploadTime`, and `userId` automatically. Any key in `extras` must be registered in the meta table, or the request is rejected with 400.

## 10. List own receipts

```bash
curl -H "Authorization: Bearer <access_token>" \
  http://localhost:8080/api/v1/receipts
```

Response:
```json
[
  {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "productName": "Milk 2%",
    "purchaseDate": "2025.04.05",
    "price": "5.49CAD",
    "amount": "1",
    "storeName": "Costco",
    "latitude": 49.2827,
    "longitude": -123.1207,
    "extras": {"brand": "Kirkland"},
    "uploadTime": 1770620311,
    "userId": "550e8400-e29b-41d4-a716-446655440000"
  }
]
```

Returns only receipts belonging to the authenticated user, newest first.

## 11. Get a receipt by ID

```bash
curl -H "Authorization: Bearer <access_token>" \
  http://localhost:8080/api/v1/receipts/a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

## 12. Delete a receipt

```bash
curl -X DELETE http://localhost:8080/api/v1/receipts/a1b2c3d4-e5f6-7890-abcd-ef1234567890 \
  -H "Authorization: Bearer <access_token>"
```

Response:
```json
{
  "message": "receipt deleted"
}
```

## 13. List all users (admin only)

```bash
curl -H "Authorization: Bearer <admin_access_token>" \
  http://localhost:8080/api/v1/users
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

## 14. Delete a user (admin only)

```bash
curl -X DELETE http://localhost:8080/api/v1/users/661f9511-f30c-52e5-b827-557766551111 \
  -H "Authorization: Bearer <admin_access_token>"
```

Response:
```json
{
  "message": "user deleted"
}
```

All active refresh tokens for that user are immediately revoked.
