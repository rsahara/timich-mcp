# Contributing to timich-mcp

Thanks for helping improve Timich MCP. This repository is the public
distribution repository for the local MCP adapter that pairs with Timich Agent.

## Project Shape

Timich MCP source snapshots are exported from the Timich codebase. Keep changes
compatible with that export flow. Product source, README content, OSS
governance files, and release bundle contents should normally be changed in the
Timich source of truth and then synced here.

Good standalone-repository changes include:

- GitHub-only CI, release, issue, or pull request metadata
- repository settings and hosting process documentation
- emergency fixes that are also ready to be carried back into the Timich source
  tree

## Local Setup

Requirements:

- Go 1.26 or newer
- A paired Timich Agent on a trusted LAN, if you are testing end-to-end MCP
  search and preview behavior

Common checks:

```bash
make test
make build
```

For local runtime testing, pair with a Timich Agent and run:

```bash
go run ./cmd/timich-mcp serve
```

Do not commit `.local`, release bundles, build output, credentials, generated
state, preview files, or private deployment files.

## Pull Requests

- Keep pull requests focused and explain the user-facing behavior change.
- Add or update tests for behavior changes and bug fixes.
- Update relevant docs or specs when interfaces, setup, or security
  expectations change.
- Prefer existing project patterns over new abstractions.
- Use the project logging path for diagnostics, and never log secrets or
  sensitive user data.
- Call out whether the change needs to be reflected in the companion Timich
  Agent repository.

## Security-Sensitive Changes

Changes touching pairing, session refresh, token storage, Agent URL handling,
MCP tool input/output, preview file handling, or local network exposure should
include extra verification notes in the pull request. Avoid putting real
tokens, private hostnames, IP addresses, Agent URLs, media metadata, local state
paths, or preview file paths in tests and examples.

## License

By contributing to this repository, you agree that your contribution is
licensed under the MIT License unless you explicitly state otherwise in writing
before the contribution is accepted.
