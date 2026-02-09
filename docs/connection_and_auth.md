# Connection and Authentication

This document will explain the design choice to meet the following requirements:

1. User can choose to host the service locally or on cloud
2. Different users hosting locally or cloud can still connect to each other, the client-service can form peer-to-peer location and connection agnostic system
3. Client side generate and stores the credential for current user, not the service side.
4. For the peer-to-peer connection, the current user can create an access key, anyone with the access key and ip **can and only can** call this service for reading data, but never writing.
5. User can revoke a key and other users use that key will lose access rights to server.

# Local Host Solution

As the target users will be small groups of people and we want it to be free, we will try to explore free solutions that can host locally (unless the cost of remote storage and service hosting is free).

Right now, the choices are [Tailscale ](https://tailscale.com/) and [headscale](https://github.com/juanfont/headscale)

# Location-Agnostic Compatibility

User A                           User B
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

# Client-Side Login and Authentication

Unlike regular platforms with centralized authentication system, we have the client to remember user name, generate key pair, and signature.

The service will also have one admin which the maintainer should create and remember the admin key for.

Then the database will request and save the owner's signature so the service can verify which requests do have write access.

We consider the following situations where a user can lose data or authentication information:

1. If the production database is lost, we hope the user can reconstruct with the help of client and staging data saved. 

2. If the client is lost, the user needs to reconstruct the database as well, the database records the signature of the owner. The only added step is to create a client again.

The core of recovery is to ensure your staging data is not lost.

# Access Key Management

Only the owner can issue access key and it does not get tied to individual user yet. We would like to simulate a social network model of people follow things but also have the same access management like in data lakes.