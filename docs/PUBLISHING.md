# Publishing notes

How this server is distributed — maintainer reference, not user documentation.

## Releasing

`git tag vX.Y.Z && git push` → release.yml (version check, then GoReleaser) →
publish-registry.yml, which release.yml calls as a `workflow_call`. Checklist:
`.claude/RELEASING.md`; workflow_dispatch re-publishes a tag by hand.

Don't reintroduce a `release: published` trigger — GoReleaser creates the release with the
workflow's `GITHUB_TOKEN`, whose events suppress triggers. That silently published nothing
in 2.0.1 and 3.0.0.

`server.json` is not committed. `server.template.json` holds the metadata with a `0.0.0`
version and zero checksums; `scripts/generate-server-json.js` fills in the real values in
CI, and `scripts/verify-release-version.js` rejects a tag disagreeing with
plugin.json/marketplace.json.

A **major** bump also moves the module path (`/v4` → `/v5`), or the proxy ignores every new
tag. `go run github.com/icholy/gomajor@latest path -next` rewrites go.mod and the imports;
the `-X` ldflag in `.goreleaser.yml` and the `go install` lines in the two READMEs are by hand.

"Auto-registered" only means the registry entry updates on release — clients still install
via .mcp.json, the plugin, or a .mcpb bundle.

## Allowed directories

Resolved at startup by internal/config/baseline.go: CLI args, then
`MCP_FILE_TOOLS_ALLOWED_DIRS`, then the working directory — which is what makes the open
workspace reachable with no config, a drive root or home being refused. MCP roots
(filetoolsserver/roots.go) merge on top where a client still sends them; 2026-07-28 clients
cannot be asked at all (SEP-2322), so that path is a bonus, not the mechanism. Smithery
prompts via configSchema.

## Claude Code plugin

- `.claude-plugin/marketplace.json` — enables `/plugin marketplace add dimitar-grigorov/mcp-file-tools`
- `plugin/.mcp.json` — declares the server. It must live here; an inline `mcpServers` block
  in plugin.json is not picked up.
- `plugin/bin/run.js` — downloads the pinned binary on first run, verifies its SHA-256,
  caches it under CLAUDE_PLUGIN_DATA, hands over stdio.

The launcher is Node, not bash: Claude Code spawns MCP servers without a shell, and on
Windows `bash` resolves to the WSL stub, failing with "Connection closed". The Go binary
itself has no runtime dependency.

## Self-update

internal/updater/updater.go only *notifies* — it checks GitHub on startup and prints a
message, gated by MCP_NO_UPDATE_CHECK=1 and skipped on dev builds. Its "re-download the
binary" advice is wrong for registry/Smithery/package installs. Don't build real
auto-update for a filesystem server.

## TODO

1. Run `/plugin marketplace add` + `/plugin install` end to end on macOS/Linux.
2. Make the update check opt-in.
3. Optional: ship a .mcpb bundle for one-click Claude Desktop installs.

The GoReleaser `mcp:` block in .goreleaser.yml stays disabled — it can't emit fileSha256 for
the mcpb type (goreleaser#6251). publish-registry.yml is the workaround.
