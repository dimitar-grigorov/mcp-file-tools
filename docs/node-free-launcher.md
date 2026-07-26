# Node-free plugin launcher

Prepared, not switched on. `plugin/.mcp.json` still starts the server with
`node bin/run.js`; the pieces to drop Node are in place but the compiled launcher is
not committed yet.

## Why

Claude Code ships as a standalone binary and does not bundle Node, so on a machine
without Node the server never starts and `/mcp` only says *not connected*.

## How it works

On Windows an MCP server `command` must resolve to a real `.exe` — a shell script,
`.cmd`, `.bat` or an extensionless PE all fail to spawn. Claude Code does append `.exe`
to an extensionless command, so one config line can serve every platform:

```json
{ "mcpServers": { "file-tools": { "command": "${CLAUDE_PLUGIN_ROOT}/launcher/mcp-file-tools" } } }
```

| path | what |
|---|---|
| `plugin/launcher/mcp-file-tools` | POSIX `sh` launcher, used on macOS and Linux |
| `plugin/launcher/mcp-file-tools.exe` | Windows launcher, built by CI (not committed yet) |
| `cmd/mcp-file-tools-launcher/` | its source |
| `.github/workflows/launcher.yml` | builds, verifies, optionally commits and attests |

Both launchers read the pinned version from `plugin/.claude-plugin/plugin.json`, download
that release's binary for the current OS and architecture, check it against the release
`checksums.txt`, cache it under `${CLAUDE_PLUGIN_DATA}` and hand over stdio. Only one
Windows launcher is committed and it is amd64, so it asks the OS for the host architecture
rather than its own and fetches the native server on arm64.

The directory is `launcher/` rather than `bin/` because a plugin's `bin/` is added to the
Bash tool's `PATH`, and a launcher that speaks JSON-RPC on stdin should not be callable as
a bare command.

## Verifying the committed binary

The build is bit-for-bit reproducible, so it can be checked rather than trusted:

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -buildvcs=false -o /tmp/launcher.exe ./cmd/mcp-file-tools-launcher
sha256sum /tmp/launcher.exe plugin/launcher/mcp-file-tools.exe   # must match
```

`-buildvcs=false` is required: VCS stamping makes the hash depend on the commit and on a
dirty tree. The Go toolchain version is embedded, so `go.mod` pins it exactly and bumping
it changes the hash. Once a binary is committed the workflow rebuilds it on every run and
fails if it does not match, and attaches keyless Sigstore provenance —
`gh attestation verify plugin/launcher/mcp-file-tools.exe -R dimitar-grigorov/mcp-file-tools`.

The binary is deliberately not packed and not stripped: packers are the main cause of
antivirus false positives, and nothing is downloaded by proxy through `certutil` or
similar. HTTPS goes through WinInet, which also means TLS from Schannel and the system
proxy for free. Running the workflow with `scan: true` uploads it to VirusTotal, which
needs a `VT_API_KEY` secret and is informational — it never fails the build.

A `posix` job runs the shell launcher on real Linux (x64 and arm64) and macOS runners and
checks that it downloads the server and completes an MCP handshake.

## Switching over

```diff
-{ "command": "node", "args": ["${CLAUDE_PLUGIN_ROOT}/bin/run.js"] }
+{ "command": "${CLAUDE_PLUGIN_ROOT}/launcher/mcp-file-tools" }
```

Three things have to be settled first, in this order:

1. **The executable bit on macOS and Linux.**
   [#40280](https://github.com/anthropics/claude-code/issues/40280) reports `claude plugin
   update` dropping it from plugin scripts. A launcher that is spawned directly needs `+x`,
   and `.mcp.json` has no way to add it, so if the bit is lost this design does not work
   there at all. Test a fresh install *and* an update, not just a clone.
2. **The `.exe` fallback in the real client.** Resolving an extensionless command to `.exe`
   is a libuv implementation detail, not documented behaviour. Check the Windows CLI and the
   desktop app separately, then state a minimum Claude Code version in both READMEs.
3. **The reproducibility gate.** Uncomment the `pull_request` trigger and mark the check
   required in the same change that first commits the `.exe`; until then nothing enforces it.

Then run the workflow with `commit: true`, drop the Node requirement from both READMEs and
delete `plugin/bin/run.js`.
