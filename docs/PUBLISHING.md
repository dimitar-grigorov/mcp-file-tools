# Publishing notes

Internal notes on how this server is distributed. Not meant for the public repo.

## Releasing

`git tag vX.Y.Z && git push` → release.yml (version check, then GoReleaser: binaries +
checksums.txt) → publish-registry.yml on `release: published` (generates server.json,
`mcp-publisher publish`). Fully automatic; workflow_dispatch re-publishes a tag by hand.
Checklist: `.claude/RELEASING.md`.

`server.json` is not committed — `server.template.json` holds the hand-maintained
metadata with a `0.0.0` version and all-zero checksums, and `scripts/generate-server-json.js`
fills in the real values in CI. That's what stopped the committed copy drifting behind
plugin.json. `scripts/verify-release-version.js` rejects a tag that disagrees with
plugin.json/marketplace.json before anything is built.

"Auto-registered" only means the registry entry updates on release — clients still
install via .mcp.json, the plugin, or a .mcpb bundle.

## Allowed directories

Taken from the client via the MCP roots protocol (filetoolsserver/roots.go), so in Claude
Code the open workspace is allowed with no config. CLI args are the fallback for clients
that don't send roots; Smithery prompts via configSchema.

## Claude Code plugin

- `.claude-plugin/marketplace.json` — enables `/plugin marketplace add dimitar-grigorov/mcp-file-tools`
- `plugin/.mcp.json` — declares the server as `node bin/run.js`. It must live here;
  an inline `mcpServers` block in plugin.json is not picked up.
- `plugin/bin/run.js` — downloads the pinned binary on first run, verifies its SHA-256,
  caches it under CLAUDE_PLUGIN_DATA, hands over stdio.

The launcher is Node, not bash, because Claude Code spawns MCP servers without a shell and
on Windows `bash` resolves to the WSL/WindowsApps stub, failing with "Connection closed".
The Go binary itself has no runtime dependency.

## Self-update

internal/updater/updater.go only *notifies* — it checks GitHub on startup and prints a
message. Gated by MCP_NO_UPDATE_CHECK=1, skipped on dev builds. Its "re-download the
binary" advice is wrong for registry/Smithery/package installs. Don't build real
auto-update for a filesystem server.

## TODO

1. Run `/plugin marketplace add` + `/plugin install` end to end on macOS/Linux.
2. Make the update check opt-in.
3. Optional: ship a .mcpb bundle for one-click Claude Desktop installs.

The GoReleaser `mcp:` block in .goreleaser.yml stays disabled — it can't emit fileSha256
for the mcpb type (goreleaser#6251). publish-registry.yml is the workaround.
