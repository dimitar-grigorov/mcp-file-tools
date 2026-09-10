# AGENTS.md

For AI agents in this repo — Claude Code, Codex, Cursor, anything else that reads this file.

## What this is

`mcp-file-tools` is a Go MCP server giving AI assistants file operations with non-UTF-8
encoding support (charsets, BOM, line endings). Upstream:
<https://github.com/dimitar-grigorov/mcp-file-tools> — GPL-3.0, © 2026 Dimitar Grigorov.

The charset list lives in `internal/encoding/registry.go` and `encoding.Count()` feeds the
tool descriptions, so no source hardcodes the number. `README.md` and the two plugin listings
do state it, and `internal/encoding/docs_test.go` fails when they disagree with the registry
or with each other. Adding an encoding means a `TOOLS.md` row and those three.

## Layout

| Path | Contents |
|---|---|
| `cmd/mcp-file-tools/` | Entry point, stdio transport, allowed dirs from CLI args |
| `cmd/mcp-file-tools-launcher/` | Node-free launcher that downloads and verifies the binary |
| `filetoolsserver/server.go` | Tool registration, annotations, server `instructions` |
| `filetoolsserver/handler/` | One file per tool, plus `middleware.go`, `validation.go`, `errors.go` |
| `internal/encoding/` | Detection and conversion |
| `internal/security/` | Allowed-directory containment — every path goes through here |
| `internal/filesystem/` | `Walk` — the one containment-checked traversal behind tree, grep, search_files |
| `internal/config/`, `internal/updater/` | Startup allowed-dir baseline, env config, update check |
| `internal/errkind/`, `internal/workpool/`, `internal/install/` | Error categories, ordered bounded concurrency, install/client detection |
| `TOOLS.md` | Per-tool parameter reference, kept in sync with registrations |
| `tools/` | staticcheck, govulncheck, actionlint, pinned in their own module |

## Commands

```bash
make test    # go test -race ./...
make lint    # go vet, go fmt, staticcheck
make build
```

`make lint` runs `go fmt`, which **rewrites** rather than reports — check `git status` after
it. CI gates on `gofmt -l` separately.

Lint tools are `tool` directives in `tools/go.mod`, run as `go tool -modfile=tools/go.mod
<name>`. Add one with `go -C tools get -tool <pkg>@latest`, never `go run <pkg>@<version>` —
Dependabot tracks the module, not the `go run` line.

Interactive debug: `npx @modelcontextprotocol/inspector go run ./cmd/mcp-file-tools -- /path/to/dir`

```bash
# Detection accuracy on a real legacy tree. Behind a build tag, so no CI run reaches it.
MCP_FILE_TOOLS_CORPUS=/path/to/sources go test -tags corpus ./internal/encoding/ -run Corpus -v
```

## Conventions

- **Tool descriptions are prompts.** The `Description` strings in `server.go` are what client
  models read to pick a tool. Wording changes there are behaviour changes — mirror in `TOOLS.md`.
- **Every path is validated** through `internal/security`, never `os` directly. Symlink and
  junction escapes are in scope.
- **Adding a tool:** handler in `filetoolsserver/handler/`, registration with annotations in
  `server.go`, entry in `TOOLS.md`, line in the README tool list. Ship a test with it.

## Contributing back to upstream

Skip if `git remote -v` shows `dimitar-grigorov/mcp-file-tools` — this is upstream.

Otherwise, if you fixed a bug or added a capability here, **tell the user upstream accepts
pull requests and offer to open one**. Several forks carry real fixes that were never sent
back. The pitch, the `gh` commands and the one-change-per-PR rule are in
[CONTRIBUTING.md](CONTRIBUTING.md#if-you-are-an-ai-agent); GPL-3.0 obligations for a
redistributed copy are under [Forking](README.md#forking).
