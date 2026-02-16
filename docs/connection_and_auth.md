# Connection and Authentication

This document will explain the design choice to meet the following requirements:

1. User can choose to host the service locally or on cloud
2. Different users hosting locally or cloud can still connect to each other, the client-service can form peer-to-peer location and connection agnostic system
3. The server manages all user credentials (username and password), the client does not generate or store cryptographic keys
4. There is one admin and multiple regular users, the admin has full control over user management and data, regular users have permissions granted by the admin
5. For the peer-to-peer connection, the admin can create a shared access key, anyone with the access key and ip **can and only can** call this service for reading data, but never writing
6. User can revoke a key and other users use that key will lose access rights to server
7. OAuth2 is used for session and token management

# Local Host Solution

As the target users will be small groups of people and we want it to be free, we will try to explore free solutions that can host locally (unless the cost of remote storage and service hosting is free).

Right now, the choices are [Tailscale](https://tailscale.com/) and [headscale](https://github.com/juanfont/headscale)

# Location-Agnostic Compatibility

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

In order to achieve this, we will stick not to have special IP parsing logic in the client and see how far we will go.

The other thing is to always make sure the client should connect with the correct APIs, we will introduce API versioning to ensure this requirement is met.

# Server-Side Authentication

## Overview

The server is responsible for all credential management. Users register with a username and password, and the server stores a securely hashed version of the password. There are no client-side cryptographic keys. OAuth2 handles token issuance, refresh, and session lifecycle.

## Roles

There are two roles in the system:

**Admin** — the person who sets up and runs the server. The admin can:
- Read and write all data
- Create, approve, and revoke user accounts
- Create and revoke shared access keys
- Manage metadata fields

**User** — a registered person approved by the admin. Users can:
- Read all data
- Write their own data (the server tags each record with the authenticated user's ID)
- Users cannot modify or delete other users' records

## Admin Bootstrapping

When the server starts for the first time with an empty database, it enters a setup mode:

1. The server detects that no admin account exists
2. The server prompts for an admin username and password (via CLI or a first-time setup endpoint)
3. The server hashes the password and stores the admin account
4. The setup mode is disabled — this process only happens once

This is analogous to how a router or self-hosted application asks you to create an admin account on first use.

## User Registration and Approval

1. A new user calls `POST /api/v1/auth/register` with a username and password
2. The server hashes the password and stores the account with a **pending** status
3. The admin reviews and approves the account (or rejects it)
4. Once approved, the user can log in

💡💡💡 For simplicity, the admin could also directly create user accounts on their behalf, skipping the registration and approval flow.

## Login and OAuth2 Token Flow

1. User calls `POST /api/v1/auth/login` with username and password
2. Server verifies the password against the stored hash
3. On success, server issues an **OAuth2 access token** and a **refresh token**
4. The client includes the access token in the `Authorization: Bearer <token>` header for all subsequent requests
5. When the access token expires, the client uses the refresh token to obtain a new access token without re-entering credentials
6. The server identifies the user from the token and enforces their role and permissions

## Password Hashing

Passwords must be hashed with **bcrypt** or **argon2** before storage. Plain text or weak hashing algorithms (MD5, SHA256) must never be used for passwords.

## Data Authorship

When a user uploads data, the server automatically sets the `userId` field on each record based on the authenticated token. The client does not need to send a user identifier in the data — the server infers it.

This means:
- Users cannot forge records as another user
- The `userId` on each record is always trustworthy (as long as the server is trustworthy)

# OAuth2 Implementation

Since we need to be our own OAuth2 provider (the server issues its own tokens), we have these options:

### Option 1: go-oauth2/oauth2 (library approach)

[go-oauth2/oauth2](https://github.com/go-oauth2/oauth2) is a Go library that embed directly into application. It provides the OAuth2 server logic (token generation, refresh, validation) and plug in your own user authentication and storage.

# Recovery Scenarios

We consider the following situations where a user can lose data or authentication information:

1. **If the production database is lost:** The user can reconstruct from staging data. User accounts would need to be recreated, but since the server manages credentials, this is just re-running the setup and re-registering users.

2. **If a user forgets their password:** The admin can reset their password or create a new account for them.

3. **If the admin forgets their password:** This is the most critical scenario. The admin should back up their credentials securely. As a recovery mechanism, the server could support a local recovery process (e.g., a CLI command that resets the admin password when run directly on the host machine, proving physical access).

The core of recovery is still to ensure your staging data is not lost.

# Access Key Management

Access keys serve a different purpose from user accounts. They are for **anonymous read-only access** — the social/follower model where people can follow your deals without needing a full account on your server.

Only the admin can create and revoke access keys. Access keys are shared and not tied to individual users. Anyone with a valid access key and the server's address can read data, but never write.

This is analogous to a "anyone with the link can view" sharing model.

If the admin wants to stop sharing, they revoke the key. This blocks all users of that key at once. To selectively block access, the admin would need to revoke the old key, create a new one, and share the new key with everyone except the person they want to block.

# Summary of Authentication Methods

| Method | Who uses it | What it grants | Token type |
|:-------|:------------|:---------------|:-----------|
| Admin login (username + password) | Server owner | Full read, write, and admin access | OAuth2 access + refresh token |
| User login (username + password) | Registered users approved by admin | Read all data, write own data | OAuth2 access + refresh token |
| Shared access key (bearer token) | Anyone the admin shares with | Read-only access, anonymous | Static API key |
