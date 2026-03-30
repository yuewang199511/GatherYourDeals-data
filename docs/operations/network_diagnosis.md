# Network Timeout Diagnosis Ladder

When any external call (MCP tool, HTTP, CLI, TCP) times out or hangs, run this ladder in order. Stop at the first failure — it is the root cause. Do not proceed to lower layers until the layer above is confirmed working.

## The Ladder

| # | Layer | What to check | Tools | Failure means |
|---|---|---|---|---|
| 1 | Platform | Is the service provisioned and running? | Railway: `list-deployments`, `get-logs` / Azure: `az postgres flexible-server show` / AWS: `aws rds describe-db-instances` | Service never started or crashed — fix deployment first |
| 2 | Auth | Are credentials valid? | Check API keys, IAM roles, managed identity — look for 401/403 in logs | Auth problem — **not a network problem**, do not dig further into networking |
| 3 | DNS | Does the hostname resolve? | `nslookup <hostname>` / `dig <hostname>` | Missing DNS zone, wrong hostname, VNet not linked to private DNS zone |
| 4 | Routing / IP | Is the host reachable at the IP level? | `ping <ip>` / `traceroute <ip>` | Routing rule missing, VNet peering not configured, subnet misconfiguration |
| 5 | TCP | Is the port open? | `nc -zv <host> <port>` / `curl -v --connect-timeout 5 <host>:<port>` | Firewall / security group / NSG blocking the port |
| 6 | HTTP | Does the service respond? | `curl -v --max-time 5 <healthcheck-url>` | Service is up but not ready, wrong port, TLS misconfiguration |
| 7 | Application | Is the response correct? | Check status code, headers, body | Application bug, misconfigured upstream, bad environment variable |

## Key Rules

- **Do not retry a timed-out call** — a timeout is ambiguous, retrying adds no diagnostic information
- **401/403 is never a network problem** — stop at layer 2 and fix credentials
- **TCP timeout ≠ TCP refused**: timeout means dropped (firewall), refused means the host is reachable but nothing is listening
- **Stop after 3 diagnosis steps with no new information** — escalate to the user with the ladder results so far

## Cloud-Specific Notes

### Azure
- Silent TCP timeout on NSG deny — run Network Watcher Connection Troubleshoot to identify the blocking rule
- Private DNS zones must be explicitly created and linked to the VNet — DNS failure looks identical to TCP timeout at the app layer
- Application databases are not created automatically on Flexible Server — `CREATE DATABASE <name>` must be run explicitly after provisioning

### AWS
- Silent TCP timeout on Security Group deny — use VPC Reachability Analyzer to identify the exact blocking rule
- RDS DNS resolves automatically within the VPC — if DNS fails inside a VPC, check subnet DNS settings (`enableDnsSupport`, `enableDnsHostnames`)

### Railway
- Internal service hostnames are auto-registered — DNS failures are rare; check `list-deployments` and `get-logs` first
- Most connectivity issues are deployment failures, not network rules
