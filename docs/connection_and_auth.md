# Connection and Authentication

This document explains the design choices made to meet the following requirements:

1. Users can choose to host the service locally or on cloud
2. Different users hosting locally or on cloud can still connect to each other — the client-service can form a peer-to-peer, location and connection agnostic system
3. The server manages all user credentials (username and password); the client does not generate or store cryptographic keys
4. There is one admin and multiple regular users; the admin has full control over user management and data, regular users have permissions granted by the admin
5. For peer-to-peer connection, the admin can create a shared access key; anyone with the key and IP can read data but never write
6. The admin can revoke a key and users of that key will immediately lose access
7. JWT is used for session and token management

# Local Host Solution

As the target users will be small groups of people and we want it to be free, we will explore free solutions that can host locally (unless the cost of remote storage and service hosting is free).

Current choices are [Tailscale](https://tailscale.com/) and [headscale](https://github.com/juanfont/headscale).

# Location-Agnostic Compatibility

```
UserGroup A                           UserGroup B
┌────────────┐                  ┌────────────┐
│  Client    │                  │  Client    │
│  (Browser/ │                  │  (CLI/App) │
│   App)     │                  │            │
└──────┬─────┘                  └──────┬─────┘
       │                               │
       │ Connects to                   │ Connects to
       │ own server                    │ A's server (with key + IP)
       ↓                               ↓
┌────────────┐                  ┌────────────┐
│  Server A  │◄─────────────────│  Server B  │
│  (Local/   │  Can also query  │  (Local/   │
│   Cloud)   │  each other      │   Cloud)   │
│            │                  │            │
│ - API      │                  │ - API      │
│ - Database │                  │ - Database │
│ - Auth     │                  │ - Auth     │
└────────────┘                  └────────────┘
```

To achieve this, we avoid special IP parsing logic in the client. API versioning ensures clients always connect with compatible endpoints.

# Server-Side Authentication

## Overview

The server is responsible for all credential management. Users register with a username and password, and the server stores a securely hashed version of the password. There are no client-side cryptographic keys.

Authentication uses **JWT (JSON Web Tokens)**. The server issues a short-lived access token and a longer-lived refresh token on login. The access token is stateless — the server verifies it by checking its signature with the JWT secret, with no database lookup required. The refresh token is stored in the database so it can be revoked on logout.

## Roles

There are two roles in the system:

**Admin** — the person who sets up and runs the server. The admin can:
- Read and write all data
- Create and revoke user accounts
- Create and revoke shared access keys
- Manage metadata fields

**User** — a registered person. Users can:
- Read all data
- Write their own data (the server tags each record with the authenticated user's ID)
- Users cannot modify or delete other users' records

## Admin Bootstrapping

When the server starts for the first time with an empty database, it refuses to serve traffic and asks you to run `init`:

```bash
./gatheryourdeals init
```

This prompts for an admin username and password, hashes the password, and stores the admin account. It only runs once — subsequent calls are a no-op.

## User Registration

Registration is open — any user can create an account and log in immediately.

1. A new user calls `POST /api/v1/users` with a username and password
2. The server hashes the password and stores the account
3. The user can log in right away

## Login and Token Flow

1. User calls `POST /api/v1/auth/login` with username and password
2. Server verifies the password against the stored hash
3. On success, server issues a signed **JWT access token** and a **refresh token**
4. The client includes the access token in the `Authorization: Bearer <token>` header for all subsequent requests
5. When the access token expires, the client calls `POST /api/v1/auth/refresh` with the refresh token to obtain a new pair — no re-entry of credentials required
6. On logout, the client calls `POST /api/v1/auth/logout` with the refresh token, which revokes it from the database

## JWT Access Token

The access token is a signed JWT containing:

```json
{
  "uid": "user-uuid",
  "role": "user",
  "iat": 1234567890,
  "exp": 1234571490
}
```

The server verifies it by re-signing with the `GYD_JWT_SECRET` and comparing signatures. No database call is needed per request. The role is read directly from the token claims, so there is no extra user lookup in the auth middleware either.

Sensitive information (passwords, secrets) is never stored in the token.

## Refresh Token

The refresh token is also a signed JWT, but it is stored in the `refresh_tokens` database table. This enables revocation — on logout, the token is deleted from the table and cannot be used again.

Refresh tokens are **rotated**: every time a new access token is requested via the refresh endpoint, the old refresh token is consumed and a new one is issued. This limits the damage if a refresh token is stolen.

## JWT Signing Secret

The JWT signing secret (`GYD_JWT_SECRET`) is an HMAC-SHA256 signing key. It is used to produce an unforgeable signature on every token. Anyone who possesses it can mint valid tokens for any user, so it must:

- Be at least 32 characters of random data (generate with `openssl rand -hex 32`)
- Never be stored in `config.yaml` or committed to source control
- Be rotated immediately if it is ever leaked (note: this invalidates all active sessions)

## Password Hashing

Passwords are hashed with **bcrypt** before storage. Plain text or weak hashing algorithms (MD5, SHA-256) are never used for passwords.

## Data Authorship

When a user uploads data, the server automatically sets the `userId` field on each record based on the authenticated token. The client does not send a user identifier — the server infers it from the token. This means users cannot forge records as another user.

# Access Key Management

Access keys serve a different purpose from user accounts. They are for **anonymous read-only access** — a social/follower model where people can read your deals without needing a full account on your server.

1. Only the admin can create and revoke access keys. Access keys are shared and not tied to individual users. Anyone with a valid key and the server's address can read data, but never write.

2. If the admin wants to stop sharing, they revoke the key. This blocks all users of that key at once. To selectively block one person, revoke the old key, create a new one, and share it with everyone except the person being blocked.

# Recovery Scenarios

1. **Production database lost:** Reconstruct from staging data. User accounts need to be recreated, but since the server manages credentials this is just re-running `init` and re-registering users.

2. **User forgets password:** The admin resets it with `gatheryourdeals admin reset-password`.

3. **Admin forgets password:** Run `gatheryourdeals admin reset-password` directly on the host machine. This proves physical access and does not require the server to be running.

4. **JWT secret lost or leaked:** Set a new `GYD_JWT_SECRET` and restart. All active sessions are invalidated and users must log in again.

# Summary of Authentication Methods

| Method | Who uses it | What it grants |
|:-------|:------------|:---------------|
| Admin login (username + password) | Server owner | Full read, write, and admin access — JWT access + refresh token |
| User login (username + password) | Registered users | Read all data, write own data — JWT access + refresh token |
| Shared access key (bearer token) | Anyone the admin shares with | Read-only access, anonymous — static API key |
