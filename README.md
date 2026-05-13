# Timich MCP

`timich-mcp` is a local stdio MCP server for searching and previewing photos and
videos through a paired Timich Agent on your LAN.

It does not connect to Immich directly. Timich Agent stays near your Immich
server, and `timich-mcp` runs on the machine where your MCP client runs.

## What It Does

- Pairs with Timich Agent using the normal Agent Admin UI pairing code.
- Stores only the Agent URL and Timich Agent session tokens locally.
- Refreshes the saved session lazily before status checks and tool calls.
- Exposes three MCP tools:
  - `get_search_capabilities`
  - `search_assets`
  - `get_asset_preview`

## Install From A Release

Download the `timich-mcp` archive for your platform from GitHub Releases,
extract it, and put the `timich-mcp` binary somewhere on your `PATH`.

Then check the binary:

```sh
timich-mcp version
```

## Pair With Timich Agent

1. Open the Timich Agent Admin UI on your trusted LAN.
2. Create a new pairing code.
3. Pair this machine:

```sh
timich-mcp pair --agent-url http://HOME-SERVER:8082 --pairing-code CODE
```

You can override the paired device name:

```sh
timich-mcp pair \
  --agent-url http://HOME-SERVER:8082 \
  --pairing-code CODE \
  --device-name "Timich MCP on MacBook"
```

`timich-mcp` stores state in:

```text
~/.local/state/timich-mcp/state.json
```

The state directory is created with `0700` permissions and the state file with
`0600` permissions. Treat this file like a password because it contains session
tokens for your Agent.

Check the saved pairing:

```sh
timich-mcp status
```

Remove local state:

```sh
timich-mcp logout
```

## Configure Codex MCP

Configure your MCP client to run `timich-mcp serve` over stdio. For example:

```json
{
  "mcpServers": {
    "timich": {
      "command": "/usr/local/bin/timich-mcp",
      "args": ["serve"]
    }
  }
}
```

If you want to keep state somewhere else, pass `--state-dir`:

```json
{
  "mcpServers": {
    "timich": {
      "command": "/usr/local/bin/timich-mcp",
      "args": ["serve", "--state-dir", "/path/to/timich-mcp-state"]
    }
  }
}
```

## Try The Search Tools

`get_search_capabilities` returns the paired Agent search capability response.

`search_assets` accepts a thin wrapper over the Timich Agent search API:

```json
{
  "text": "soccer",
  "mode": "auto",
  "mediaType": "image",
  "capturedAt": {
    "from": "2026-01-01T00:00:00Z",
    "to": "2026-05-01T00:00:00Z"
  },
  "sort": "default",
  "page": 0,
  "pageSize": 20
}
```

Omit `text` to browse the timeline. `page` is zero-based. `capturedAt` values
must be UTC RFC3339 timestamps ending in `Z`.

`get_asset_preview` fetches a preview for an asset returned by `search_assets`
and writes it to a temporary local file:

```json
{
  "assetId": "ta1_...",
  "filename": "IMG_0001.JPG"
}
```

Preview files are written under your system temporary directory in
`timich-mcp/previews`. Files older than 24 hours are cleaned up best-effort when
the server starts or a preview is fetched.

## Security Notes

- This first version is LAN-only. Use an Agent URL that is reachable from the
  MCP client machine, such as `http://192.168.1.20:8082`.
- Anyone with the local state file can use the saved Agent session until it is
  revoked or expires.
- Access tokens are refreshed when they are near expiry. Refresh tokens are also
  rotated when they have less than 14 days remaining, so weekly use should not
  require re-pairing.
- If refresh fails, run `timich-mcp pair` again with a new Agent pairing code.

## Developer Source Build

From a source checkout:

```sh
make test
make build
./build/timich-mcp version
```

You can also run the server directly:

```sh
go run ./cmd/timich-mcp serve
```
