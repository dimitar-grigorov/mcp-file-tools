# MCP File Tools

[![Release](https://img.shields.io/github/v/release/dimitar-grigorov/mcp-file-tools)](https://github.com/dimitar-grigorov/mcp-file-tools/releases/latest)
[![Downloads](https://img.shields.io/github/downloads/dimitar-grigorov/mcp-file-tools/total)](https://github.com/dimitar-grigorov/mcp-file-tools/releases)
[![Test](https://github.com/dimitar-grigorov/mcp-file-tools/actions/workflows/test.yml/badge.svg)](https://github.com/dimitar-grigorov/mcp-file-tools/actions/workflows/test.yml)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/dimitar-grigorov/mcp-file-tools/badge)](https://scorecard.dev/viewer/?uri=github.com/dimitar-grigorov/mcp-file-tools)
[![License: GPL-3.0](https://img.shields.io/github/license/dimitar-grigorov/mcp-file-tools)](LICENSE)
[![MCP Registry](https://img.shields.io/badge/MCP-Registry-blue)](https://registry.modelcontextprotocol.io/?search=mcp-file-tools)
[![Glama score](https://glama.ai/mcp/servers/dimitar-grigorov/mcp-file-tools/badges/score.svg)](https://glama.ai/mcp/servers/dimitar-grigorov/mcp-file-tools)

Claude sees `Настройки` — not `????` or `Íàñòðîéêè`.

MCP server for file operations on text that isn't UTF-8. It detects the encoding from the
file's bytes rather than its extension, hands the model UTF-8, and writes back in the
original encoding — BOM and CRLF/LF intact, still byte-compatible with whatever legacy
tool owns the file.

- **25 encodings, read and write** — Cyrillic (CP1251, KOI8-R/U, CP866), Windows-125x, ISO-8859-x, UTF-16 LE/BE, GBK/GB18030 ([full list](#supported-encodings))
- **Encoding-aware across the whole tool set** — `edit_file`, `grep_text_files` and `search_files` decode the same way, not just read and write
- **Detection you can inspect** — `detect_encoding` reports the charset, a confidence score and any BOM, so garbled text becomes diagnosable
- **BOM and line endings are first-class** — including on UTF-16, where a naive byte-level rewrite corrupts the file
- **Sandboxed** — every path, symlink and junction targets included, is checked against the directories you allowed

**Built for:** Delphi/Pascal units with Cyrillic UI text, VB6 forms, legacy PHP/HTML with
localized content, and INI or data files whose encoding you can't tell from the filename.

```
User: Read config.ini and change the title to "Настройки"
Claude: [read_text_file → cp1251 detected] → [edits UTF-8] → [write_file → back to cp1251]
```

> **PRs welcome and merged fast** — no CLA, no style review, one-line fixes count. Forked this to fix something? Please [send it back](#contributing) instead.

## Installation

### Claude Code plugin (recommended)

```bash
claude plugin marketplace add dimitar-grigorov/mcp-file-tools
claude plugin install mcp-file-tools
```

The same two steps inside a session are `/plugin marketplace add …` and `/plugin install …`.

**Requires [Node.js](https://nodejs.org) 18+ on your PATH** — the launcher is a Node
script, and Claude Code ships as a standalone binary that does not bundle Node. Without
it the server shows as *not connected* in `/mcp`.

On first launch the plugin downloads the right binary for your OS, verifies its SHA-256
and caches it, pinned to a known version. It is scoped to the folder you have open, so
there is nothing to configure. For directories outside the workspace, or a machine
without Node, use a [manual install](#manual-install-other-mcp-clients-or-access-outside-your-workspace).

#### Coming from a manual install

```bash
claude mcp list                                              # find the old file-tools entry
claude mcp remove file-tools                                 # drop it, or you run two copies
claude plugin marketplace add dimitar-grigorov/mcp-file-tools
claude plugin install mcp-file-tools
# Old binary, once /mcp shows the plugin connected:
#   Windows      Remove-Item "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools.exe"
#   Linux/macOS  rm ~/.local/bin/mcp-file-tools
```

The tool names change with it — `mcp__file-tools__*` becomes
`mcp__plugin_mcp-file-tools_file-tools__*` — so permission rules need updating too, see
[Auto-approve tools](#auto-approve-tools-claude-code).

### Updating the plugin

```bash
claude plugin marketplace update mcp-file-tools
claude plugin update mcp-file-tools@mcp-file-tools
```

Use the full `plugin@marketplace` id here, not the bare name. Or turn on auto-update in
`/plugin` → **Marketplaces**.

### Registries and directories

This server is listed in the [Official MCP Registry](https://registry.modelcontextprotocol.io/?search=mcp-file-tools) for discovery by any MCP client, and indexed on
[Glama](https://glama.ai/mcp/servers/dimitar-grigorov/mcp-file-tools), which scores it A
for license, quality and maintenance.

### Manual install (other MCP clients, or access outside your workspace)

Download the binary for your platform, then register it with the directories it may access.

| Platform | Release asset | Suggested path |
|----------|---------------|----------------|
| Windows x64 | `mcp-file-tools_windows_amd64.exe` | `%LOCALAPPDATA%\Programs\mcp-file-tools\mcp-file-tools.exe` |
| Linux x64 | `mcp-file-tools_linux_amd64` | `~/.local/bin/mcp-file-tools` |
| macOS ARM64 | `mcp-file-tools_darwin_arm64` | `~/.local/bin/mcp-file-tools` |

Windows (PowerShell, not CMD):

```powershell
mkdir -Force "$env:LOCALAPPDATA\Programs\mcp-file-tools"
iwr "https://github.com/dimitar-grigorov/mcp-file-tools/releases/latest/download/mcp-file-tools_windows_amd64.exe" -OutFile "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools.exe"
claude mcp add --scope user file-tools -- "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools.exe" "C:\Projects"
```

Linux / macOS (swap the asset name from the table for your platform):

```bash
mkdir -p ~/.local/bin
curl -L "https://github.com/dimitar-grigorov/mcp-file-tools/releases/latest/download/mcp-file-tools_linux_amd64" -o ~/.local/bin/mcp-file-tools
chmod +x ~/.local/bin/mcp-file-tools
claude mcp add --scope user file-tools -- ~/.local/bin/mcp-file-tools ~/Projects
```

### Go install (all platforms)

```bash
# Requires Go 1.26+
go install github.com/dimitar-grigorov/mcp-file-tools/v4/cmd/mcp-file-tools@latest
# Linux / macOS
claude mcp add --scope user file-tools -- $(go env GOPATH)/bin/mcp-file-tools ~/Projects
```

```powershell
# Windows PowerShell
claude mcp add --scope user file-tools -- "$(go env GOPATH)\bin\mcp-file-tools.exe" "C:\Projects"
```

### Other Clients

For Claude Desktop, VSCode, or Cursor, use the downloaded binary path in your config:

**Claude Desktop** (`%APPDATA%\Claude\claude_desktop_config.json` on Windows, `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

Windows:
```json
{
  "mcpServers": {
    "file-tools": {
      "command": "C:\\Users\\YOUR_NAME\\AppData\\Local\\Programs\\mcp-file-tools\\mcp-file-tools.exe",
      "args": ["D:\\Projects", "C:\\Users\\YOUR_NAME\\Documents"]
    }
  }
}
```

macOS / Linux:
```json
{
  "mcpServers": {
    "file-tools": {
      "command": "/Users/YOUR_NAME/.local/bin/mcp-file-tools",
      "args": ["/Users/YOUR_NAME/Projects", "/Users/YOUR_NAME/Documents"]
    }
  }
}
```

`args` lists the directories the server may access — as many as you need.

**VSCode / Cursor (Claude Code extension)**

If you already ran `claude mcp add --scope user` from the installation steps above, the server is already available in VSCode — no extra config needed.

To configure separately for VSCode only:
```powershell
claude mcp add --scope user file-tools -- "%LOCALAPPDATA%\Programs\mcp-file-tools\mcp-file-tools.exe" "C:\Projects"
```

Alternatively, create a **per-project config** by adding `.mcp.json` to your project root:
```json
{
  "mcpServers": {
    "file-tools": {
      "type": "stdio",
      "command": "C:\\Users\\YOUR_NAME\\AppData\\Local\\Programs\\mcp-file-tools\\mcp-file-tools.exe",
      "args": ["D:\\Projects", "D:\\Other\\Directory"]
    }
  }
}
```

**Note:** `type: "stdio"` is required here. The VSCode extension does not add the workspace
directory by itself, so `args` must list every directory you want reachable. Adding one
later means re-running `claude mcp add` with the full list — it overwrites the previous
config rather than appending.

### OpenAI Codex CLI

Codex takes a direct MCP command, so no manual TOML editing is needed.

Windows (PowerShell):
```powershell
mkdir -Force "$env:LOCALAPPDATA\Programs\mcp-file-tools"
iwr "https://github.com/dimitar-grigorov/mcp-file-tools/releases/latest/download/mcp-file-tools_windows_amd64.exe" -OutFile "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools.exe"
codex mcp add file-tools -- "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools.exe" "C:\Projects"
```

Run `codex mcp list` to verify it, then start a new Codex session. Add more directory
arguments to grant access outside the current project.

### Auto-approve tools (Claude Code)

To skip the permission prompts, add to `.claude/settings.local.json` in your project root:

```json
{ "permissions": { "allow": ["mcp__plugin_mcp-file-tools_file-tools__*"] } }
```

That prefix is the plugin install; a manual one registered as `file-tools` is
`mcp__file-tools__*` instead, and a rule with the wrong prefix matches nothing and fails
quietly. Which modes the rules affect, and keeping `delete_file` / `move_file` behind a
prompt, are in [docs/extra.md](docs/extra.md#auto-approving-tools-in-claude-code).

### Update

The server checks for updates automatically and notifies you through tool responses when a
newer version is available; the notice carries the steps for your install. Plugin installs
update through [Updating the plugin](#updating-the-plugin) — a re-downloaded binary is
ignored there.

For a manual install, re-download the binary over the existing one — the registration does
not need repeating:

1. Close all Claude Code sessions (the binary is locked while running)
2. Re-download:

```powershell
iwr "https://github.com/dimitar-grigorov/mcp-file-tools/releases/latest/download/mcp-file-tools_windows_amd64.exe" `
    -OutFile "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools.exe"
```

To disable update checks, set the environment variable `MCP_NO_UPDATE_CHECK=1`.

### Verify & Uninstall

```bash
# Check which file-tools server is connected (plugin or manual)
claude mcp list

# Remove a manual install
claude mcp remove file-tools

# Remove the plugin
claude plugin uninstall mcp-file-tools
```

## How to Use

Once installed, just ask Claude:
- "List all .pas files in this directory"
- "Read config.ini and detect its encoding"
- "Show all supported encodings"
- "Read MainForm.dfm using CP1251 encoding"

**Security:** the server reaches only the directories you allowed. It takes them from
`args: ["/path/to/project"]` first, then `MCP_FILE_TOOLS_ALLOWED_DIRS`, and failing both
from the directory it was started in — the workspace, when a client launches it there.
Clients that still speak the MCP roots protocol add their roots on top. A drive root or
your home directory is never granted by that last fallback; name it explicitly instead.
Paths are resolved before the check, so a symlink or Windows junction pointing outside is
rejected rather than followed.

## Tools

20 tools — every one that touches text content is encoding-aware:
- [`read_text_file`](TOOLS.md#read_text_file) - Read files with encoding auto-detection and conversion
- [`read_multiple_files`](TOOLS.md#read_multiple_files) - Read multiple files concurrently with encoding support
- [`write_file`](TOOLS.md#write_file) - Write files in specific encodings
- [`edit_file`](TOOLS.md#edit_file) - Line-based edits with diff preview and whitespace-flexible matching
- [`copy_file`](TOOLS.md#copy_file) - Copy a file to a new location
- [`delete_file`](TOOLS.md#delete_file) - Delete a file
- [`list_directory`](TOOLS.md#list_directory) - Browse directories with pattern filtering
- [`tree`](TOOLS.md#tree) - Compact indented tree view, optionally annotated with each file's encoding
- [`search_files`](TOOLS.md#search_files) - Recursively search for files matching glob patterns
- [`grep_text_files`](TOOLS.md#grep_text_files) - Regex search in file contents with encoding support
- [`detect_encoding`](TOOLS.md#detect_encoding) - Auto-detect file encoding with confidence score
- [`convert_encoding`](TOOLS.md#convert_encoding) - Convert file between encodings
- [`manage_line_endings`](TOOLS.md#manage_line_endings) - Detect or convert line endings (CRLF/LF/mixed)
- [`manage_bom`](TOOLS.md#manage_bom) - Detect, strip, or add Unicode BOM
- [`list_encodings`](TOOLS.md#list_encodings) - Show all supported encodings
- [`get_file_info`](TOOLS.md#get_file_info) - Get file/directory metadata
- [`create_directory`](TOOLS.md#create_directory) - Create directories recursively (mkdir -p)
- [`move_file`](TOOLS.md#move_file) - Move or rename files and directories
- [`list_allowed_directories`](TOOLS.md#list_allowed_directories) - Show accessible directories
- [`check_for_updates`](TOOLS.md#check_for_updates) - Check whether a newer release is available

Plus three [prompts](TOOLS.md#prompts) — `audit_encodings`, `fix_mojibake`,
`migrate_to_utf8` — surfaced by clients as user commands.

See [TOOLS.md](TOOLS.md) for detailed parameters and examples. Calls shaped like
Claude Code's built-in Read/Write/Edit/Grep are accepted too — the
[alias layer](TOOLS.md#built-in-name-aliases) translates them where the semantics
match exactly, so a model's habits don't fail the call.

**Out of scope:** binary/media reading (`read_media_file`). This is a *text*
tool; agents read images with their built-in tools.

### Supported encodings

Every one below reads and writes. Name one explicitly via the `encoding` parameter, or
leave it to auto-detection.

| Script / region | Encodings |
|---|---|
| Unicode | UTF-8, UTF-16 LE, UTF-16 BE |
| Cyrillic | Windows-1251, KOI8-R, KOI8-U, CP866, ISO-8859-5, MacCyrillic |
| Western European | Windows-1252, ISO-8859-1, ISO-8859-15 |
| Central European | Windows-1250, ISO-8859-2 |
| Greek | Windows-1253, ISO-8859-7 |
| Turkish | Windows-1254, ISO-8859-9 |
| Chinese Simplified | GBK, GB18030 |
| Hebrew, Arabic, Baltic, Vietnamese, Thai | Windows-1255, 1256, 1257, 1258, 874 |

Common aliases are accepted (`cp1251`, `latin1`, `gb2312`, `tis-620`, …) —
[`list_encodings`](TOOLS.md#list_encodings) prints the whole table with aliases.

UTF-32 is partially supported: LE and BE BOMs are detected, and
[`manage_bom`](TOOLS.md#manage_bom) can add or strip them, but transcoding to or from
UTF-32 is not implemented and [`manage_line_endings`](TOOLS.md#manage_line_endings)
refuses UTF-32 files rather than corrupting their 4-byte alignment.

## Configuration

The server can be configured via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_DEFAULT_ENCODING` | Default encoding for `write_file` on **new** files when none specified. Existing files keep their detected encoding. Set to `cp1251` to restore the pre-2.0.0 default. | `utf-8` |
| `MCP_DEFAULT_LINE_ENDINGS` | Line endings for `write_file` on **new** files (`crlf`/`lf`). Existing files keep their own style regardless. | unset (write unchanged) |
| `MCP_MEMORY_THRESHOLD` | Memory threshold in bytes. Files smaller are loaded into memory for faster I/O; larger files use streaming. Also affects encoding detection mode. | `67108864` (64MB) |
| `MCP_DETECTION_CANDIDATES` | Comma-separated list pinning what detection may answer, in priority order — e.g. `utf-8,windows-1252`. See [Pinning the encodings](#pinning-the-encodings). | unset (detection unrestricted) |
| `MCP_FILE_TOOLS_ALLOWED_DIRS` | Allowed directories as an OS path list (`;` on Windows, `:` elsewhere). For clients where `env` is the only block you control, such as the Claude Code plugin. Overridden by `args`. | unset |
| `MCP_FILE_TOOLS_NO_CWD_FALLBACK` | Set to turn off granting the working directory when neither `args` nor `MCP_FILE_TOOLS_ALLOWED_DIRS` names one. | unset (fallback on) |

Set them with an `env` block in your config (Claude Desktop example):

```json
{
  "mcpServers": {
    "file-tools": {
      "command": "C:\\Users\\YOUR_NAME\\AppData\\Local\\Programs\\mcp-file-tools\\mcp-file-tools.exe",
      "args": ["D:\\Projects"],
      "env": {
        "MCP_DEFAULT_ENCODING": "utf-8"
      }
    }
  }
}
```

### Pinning the encodings

Detection is a guess, and guesses have blind spots: Spanish CP1252 like `MÓDULO
FÍSICAMENTE ÚNICO` is plausible GBK — every uppercase accent before an ASCII letter is a
valid hanzi pair — so it reads back as Chinese and edits fail with *"gbk cannot represent
2 characters"*. If you know what the repo contains, say so:

```json
"env": { "MCP_DETECTION_CANDIDATES": "utf-8,windows-1252" }
```

A BOM still wins. A guess inside the list keeps its confidence; one outside it is dropped
and the first listed encoding that decodes the bytes cleanly takes over, so order is your
priority. A file that fits none of them is read as the default and reported as an **ODD
ENCODING** in `read_text_file`'s hint, so a stray file gets said out loud rather than
guessed at. Unlisted encodings stop appearing in `detect_encoding`'s `candidates` too. UTF-16/32 are named only by a BOM
or the structural classifier, so listing them cannot make them a catch-all.

### Legacy teams (pre-2.0.0 behaviour)

Before 2.0.0 new files defaulted to `cp1251`; they now default to `utf-8`. Existing files
are unaffected — their encoding is detected and preserved — so this only matters if your
team **creates** new non-UTF-8 files, e.g. new Delphi units with Cyrillic literals. To keep
the old behaviour:

```json
"env": { "MCP_DEFAULT_ENCODING": "cp1251" }
```

Commit that in the legacy repo's `.mcp.json` rather than setting it per machine, and
everyone working in that repo gets the right default with no local setup.

**Delphi 2007 and older** read UTF-8 only when it carries a BOM, so a UTF-8 file without one
is silently treated as ANSI. Set `cp1251` (or your own ANSI code page) for such a repo and
new Cyrillic literals land in the encoding the IDE expects. Files that already exist keep
their own encoding either way, and no tool adds a BOM to them.

## Development

**Prerequisites:** Go 1.26+

```bash
make test    # go test -race ./...
make lint    # go vet, go fmt, staticcheck (same pinned version as CI)
make build
```

`test_server.go` is an end-to-end smoke test over every tool, run by CI on each push:

```bash
go run test_server.go
```

### Debugging

[MCP Inspector](https://github.com/modelcontextprotocol/inspector) gives a web UI for calling tools and inspecting responses (needs Node.js 18+):

```bash
npx @modelcontextprotocol/inspector go run ./cmd/mcp-file-tools -- /path/to/allowed/dir
```

Or pipe JSON-RPC straight to stdin:

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | go run ./cmd/mcp-file-tools /path/to/project
```

## Contributing

**If it fits the scope and works, it gets merged.** Don't ask first — just send the PR.
No CLA, no style review: `make test` and `make lint` passing is enough, and tests are
welcome but never required. One-line fixes and half-finished features behind a flag both
count. Out of scope, or breaking a tool contract other people's agents rely on, gets a
comment rather than a close. Not writing the fix yourself?
[Open an issue](https://github.com/dimitar-grigorov/mcp-file-tools/issues) with the file,
its encoding, and what the tool did.

Details in [CONTRIBUTING.md](CONTRIBUTING.md). Found a way out of an allowed
directory? That one goes to [SECURITY.md](SECURITY.md), privately, not to an issue.

## Forking

Forking is fine. That's what GPL-3.0 is for. Taking the project over is not.

**GPL-3.0 is a license, not a preference.** Distribute your fork in any form (public repo,
release binary, registry listing, product you ship to customers) and you must:

- **Keep GPL-3.0** and [LICENSE](LICENSE), copyright notice intact (§4, §5c)
- **Say what you changed** and when, prominently (§5a)
- **Give the source** to everyone you gave the binary to (§6)

> [!WARNING]
> Deleting the license or the notice, relicensing as MIT or proprietary, or shipping
> only a binary is a **license violation**, and §8 ends your rights the moment you do it.

**It will be enforced,** in this order: a request to comply, then a DMCA takedown plus
delisting from whichever registry or marketplace carries it, then legal action. Complying
costs one license file, one notice and one source link.
[Ask in an issue](https://github.com/dimitar-grigorov/mcp-file-tools/issues) if you are
unsure whether what you ship complies.

**Asked, not enforced:** leave the credit in — the copyright notice is the legal minimum,
one line saying *"Fork of [mcp-file-tools](https://github.com/dimitar-grigorov/mcp-file-tools)"*
is what tells a reader where it came from. Give your fork its own name, so a registry
listing under this one with the author swapped doesn't read as if the project moved and
send its bugs here. And try upstream first: a PR beats carrying merge conflicts forever,
and puts your name on the commit rather than in a credits list.

## Credits

Ideas that started in someone else's fork and were reimplemented here:

- [@skyispainted](https://github.com/skyispainted) - GBK/GB18030, JSON-string array args, `edit_file` retry hint
- [@haobiao](https://github.com/haobiao) - GBK/GB18030, independently
- [Hugo Rosário](https://github.com/WTC-ZoneSoft) - merging MCP roots with CLI allowed dirs
- [@zoster81](https://github.com/zoster81) - UTF-16 line endings and grep, path containment fixes, write durability, BOM policy, ordered concurrency
- [Mario Rial](https://github.com/seguridadea1) - pinning detection to known encodings, multi-pattern grep

A PR gets your name on the commit instead of this list.

## License

GPL-3.0 - see [LICENSE](LICENSE)

Copyright (C) 2026 Dimitar Grigorov. Free software, distributed WITHOUT ANY WARRANTY.
