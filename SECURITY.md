# Security Policy

Timich MCP runs on the user's MCP client machine and connects to a paired
Timich Agent on a trusted LAN. It handles Agent URLs, paired-device sessions,
local state files, MCP tool requests, and preview files. Please report security
issues privately before sharing details in public.

## Supported Versions

| Track | Support |
| --- | --- |
| Latest GitHub release | Supported for security fixes. |
| `main` | Receives fixes before the next release when practical. |
| Older releases | Best effort only; users should upgrade to the latest release. |

## Reporting a Vulnerability

Use GitHub's private vulnerability reporting flow for this repository from the
Security tab's "Report a vulnerability" action.

Do not open a public issue, discussion, or pull request for a vulnerability
report. Do not include exploit details, secrets, tokens, logs, private network
information, Agent URLs, local state files, preview file paths, or private media
metadata in public project spaces.

Helpful reports include:

- affected version, commit, or release asset
- MCP client and operating system
- Timich Agent version or release track, if relevant
- impact and expected attacker position
- minimal reproduction steps
- sanitized logs or request examples, if relevant
- whether the issue is already public or under coordinated disclosure elsewhere

## Security Expectations

- Timich MCP is intended to connect to Timich Agent over trusted local networks
  or host-local access.
- Do not expose Timich Agent's LAN ports directly to the public internet.
- Treat Agent URLs, pairing codes, access tokens, refresh tokens, local state
  files, and preview files as sensitive.
- Redact secrets and private media metadata from issues, pull requests, logs,
  screenshots, and test fixtures.
- Remove local state with `timich-mcp logout` before sharing a machine, support
  bundle, or development environment.

Security fixes may be released without full details until users have had a
reasonable opportunity to upgrade.
