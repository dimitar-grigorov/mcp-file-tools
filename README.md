# MCP File Tools

[![Go Report Card](https://goreportcard.com/badge/github.com/dimitar-grigorov/mcp-file-tools)](https://goreportcard.com/report/github.com/dimitar-grigorov/mcp-file-tools)
[![Release](https://img.shields.io/github/v/release/dimitar-grigorov/mcp-file-tools)](https://github.com/dimitar-grigorov/mcp-file-tools/releases/latest)
[![License: GPL-3.0](https://img.shields.io/github/license/dimitar-grigorov/mcp-file-tools)](LICENSE)
[![MCP Registry](https://img.shields.io/badge/MCP-Registry-blue)](https://registry.modelcontextprotocol.io/?search=mcp-file-tools)

Claude sees `Настройки` — not `????` or `Íàñòðîéêè`.

MCP server for file operations with non-UTF-8 encoding support. Auto-detects and converts 24 encodings (Cyrillic, Windows-125x, ISO-8859, KOI8, UTF-16, GBK/GB18030) so AI assistants can read and write legacy files without corrupting data.

**Perfect for:** Delphi/Pascal projects, legacy VB6 apps, old PHP/HTML sites, config files with non-UTF-8 text.

> **PRs welcome and merged fast** — no CLA, no style review, one-line fixes count. Forked this to fix something? Please [send it back](#contributing) instead.

## What It Does

Provides 21 tools for file operations with automatic encoding conversion:
- [`read_text_file`](TOOLS.md#read_text_file) - Read files with encoding auto-detection and conversion
- [`read_multiple_files`](TOOLS.md#read_multiple_files) - Read multiple files concurrently with encoding support
- [`write_file`](TOOLS.md#write_file) - Write files in specific encodings
- [`edit_file`](TOOLS.md#edit_file) - Line-based edits with diff preview and whitespace-flexible matching
- [`copy_file`](TOOLS.md#copy_file) - Copy a file to a new location
- [`delete_file`](TOOLS.md#delete_file) - Delete a file
- [`list_directory`](TOOLS.md#list_directory) - Browse directories with pattern filtering
- [`tree`](TOOLS.md#tree) - Compact indented tree view (85% fewer tokens than JSON)
- [`directory_tree`](TOOLS.md#directory_tree-deprecated) - Get recursive tree view as JSON (deprecated, use `tree`)
- [`search_files`](TOOLS.md#search_files) - Recursively search for files matching glob patterns
- [`grep_text_files`](TOOLS.md#grep_text_files) - Regex search in file contents with encoding support
- [`detect_encoding`](TOOLS.md#detect_encoding) - Auto-detect file encoding with confidence score
- [`convert_encoding`](TOOLS.md#convert_encoding) - Convert file between encodings
- [`detect_line_endings`](TOOLS.md#detect_line_endings) - Detect line ending style (CRLF/LF/mixed)
- [`change_line_endings`](TOOLS.md#change_line_endings) - Convert line endings to LF or CRLF
- [`manage_bom`](TOOLS.md#manage_bom) - Detect, strip, or add Unicode BOM
- [`list_encodings`](TOOLS.md#list_encodings) - Show all supported encodings
- [`get_file_info`](TOOLS.md#get_file_info) - Get file/directory metadata
- [`create_directory`](TOOLS.md#create_directory) - Create directories recursively (mkdir -p)
- [`move_file`](TOOLS.md#move_file) - Move or rename files and directories
- [`list_allowed_directories`](TOOLS.md#list_allowed_directories) - Show accessible directories

**Supported encodings (22 total):**
- **Unicode:** UTF-8, UTF-16 LE, UTF-16 BE (with BOM detection for UTF-16 and UTF-32)
- **Cyrillic:** Windows-1251, KOI8-R, KOI8-U, CP866, ISO-8859-5
- **Western European:** Windows-1252, ISO-8859-1, ISO-8859-15
- **Central European:** Windows-1250, ISO-8859-2
- **Greek:** Windows-1253, ISO-8859-7
- **Turkish:** Windows-1254, ISO-8859-9
- **Other:** Hebrew (1255), Arabic (1256), Baltic (1257), Vietnamese (1258), Thai (874)

See [TOOLS.md](TOOLS.md) for detailed parameters and examples.

**Security:** All operations restricted to allowed directories only.

## Installation

### Claude Code plugin (recommended)

The simplest way to use this with Claude Code:

```
/plugin marketplace add dimitar-grigorov/mcp-file-tools
/plugin install mcp-file-tools
```

**Requires [Node.js](https://nodejs.org) 18+ on your PATH** — the launcher is a Node
script. Claude Code ships as a standalone binary and does not bundle Node, so
`node --version` can fail on an otherwise working install; the server then shows as
*not connected* in `/mcp`.

On first launch the plugin downloads the right binary for your OS, verifies its
SHA-256, caches it, and keeps it pinned to a known version. The server is
automatically scoped to the folder you have open (via the MCP roots protocol), so
there is nothing to configure.

The plugin only accesses your current workspace. Without Node, or to grant access to
directories outside the workspace, use a manual install (below).

**Already added the server the manual way?** Remove the old `claude mcp add` entry so
you are not running two copies:

```
claude mcp remove file-tools
```

### Updating the plugin

```
claude plugin marketplace update mcp-file-tools
claude plugin update mcp-file-tools@mcp-file-tools
```

Use the full `plugin@marketplace` id, not the bare name. Or enable auto-update in
`/plugin` → **Marketplaces**.

### MCP Registry

This server is listed in the [Official MCP Registry](https://registry.modelcontextprotocol.io/?search=mcp-file-tools) for discovery by any MCP client.

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
claude mcp add --scope user file-tools -- "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools.exe" "D:\Projects"
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
go install github.com/dimitar-grigorov/mcp-file-tools/cmd/mcp-file-tools@latest
# Linux / macOS
claude mcp add --scope user file-tools -- $(go env GOPATH)/bin/mcp-file-tools ~/Projects
```

```powershell
# Windows PowerShell
claude mcp add --scope user file-tools -- "$(go env GOPATH)\bin\mcp-file-tools.exe" "D:\Projects"
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

The `args` array specifies allowed directories the server can access. Add as many directories as you need.

**VSCode / Cursor (Claude Code extension)**

If you already ran `claude mcp add --scope user` from the installation steps above, the server is already available in VSCode — no extra config needed.

To configure separately for VSCode only:
```powershell
claude mcp add --scope user file-tools -- "%LOCALAPPDATA%\Programs\mcp-file-tools\mcp-file-tools.exe" "D:\Projects"
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

**Note:** The `type: "stdio"` field is required. The `args` array specifies allowed directories — the VSCode extension does not automatically add the workspace directory, so you must list all directories you want to access. To add more directories later, re-run the `claude mcp add` command with all directories listed (it overwrites the previous config).

**OpenAI Codex CLI**

Codex does not have an `mcp add` command -- you need to edit `~/.codex/config.toml` manually.

Windows (PowerShell):
```powershell
# Download
mkdir -Force "$env:LOCALAPPDATA\Programs\mcp-file-tools"
iwr "https://github.com/dimitar-grigorov/mcp-file-tools/releases/latest/download/mcp-file-tools_windows_amd64.exe" -OutFile "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools.exe"
```

Then add to `~/.codex/config.toml`:
```toml
[mcp_servers.file-tools]
command = "C:\\Users\\YOUR_NAME\\AppData\\Local\\Programs\\mcp-file-tools\\mcp-file-tools.exe"
args = ["D:\\Projects"]
```

### Auto-approve tools (Claude Code)

To skip the permission prompts, add to `.claude/settings.local.json` in your project root:

```json
{ "permissions": { "allow": ["mcp__file-tools__*"] } }
```

The plugin install uses a different prefix, and keeping `delete_file` / `move_file` behind
a prompt takes one more rule — see [docs/extra.md](docs/extra.md#auto-approving-tools-in-claude-code).

### Update

The server checks for updates automatically and notifies you through tool responses when a newer version is available. To update:

1. Close all Claude Code sessions (the binary is locked while running)
2. Re-download the binary:

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

**Security:** The server only accesses directories you explicitly allow:
- **Automatic:** Claude Desktop/Code provide workspace directories automatically
- **Manual:** Specify directories in config `args: ["/path/to/project"]`

## Configuration

The server can be configured via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_DEFAULT_ENCODING` | Default encoding for `write_file` when none specified | `cp1251` |
| `MCP_MEMORY_THRESHOLD` | Memory threshold in bytes. Files smaller are loaded into memory for faster I/O; larger files use streaming. Also affects encoding detection mode. | `67108864` (64MB) |

To override, set environment variables in your config (Claude Desktop example):
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

## Use Cases

### Legacy Codebases

Many legacy projects use non-UTF-8 encodings that AI assistants can't handle natively:

- **Delphi/Pascal** (Windows-1251): Source files with Cyrillic UI text
- **Visual Basic 6** (Windows-1252): Forms and config files with Western European characters
- **Legacy PHP/HTML** (CP1251, ISO-8859-1): Web apps with localized content
- **Old config files** (Various): INI, properties, registry files with legacy encodings

**How it works:**
```
User: Read config.ini and change the title to "Настройки"
Assistant: [read_text_file with cp1251] → [modify UTF-8] → [write_file with cp1251]
```

The original encoding is preserved - files remain compatible with legacy tools.

## Development

**Prerequisites:** Go 1.26+

```bash
# Run tests
go test ./...

# Build
go build -o mcp-file-tools ./cmd/mcp-file-tools
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

- No CLA, no style review. `make test` and `make lint` passing is enough; tests are welcome, never required.
- One-line fixes and half-finished features behind a flag are both fine.
- Pushed back on (with a comment, not a close): out of scope, or breaking tool contracts other people's agents rely on.
- Don't want to write the fix? [Open an issue](https://github.com/dimitar-grigorov/mcp-file-tools/issues) with the file, its encoding, and what the tool did.

Details in [CONTRIBUTING.md](CONTRIBUTING.md).

## Forking

Forking is fine — that's what GPL-3.0 is for. Taking the project over is not.

**GPL-3.0 is a license, not a preference.** If you distribute your fork — a public repo, a release binary, a registry listing, a product you ship to customers — you must keep GPL-3.0 and [LICENSE](LICENSE) in place with the copyright notice intact (§4, §5c), state prominently that you changed the files and when (§5a), and make the source available to everyone you gave it to (§6). Deleting the license or the notice, relicensing it as MIT or proprietary, or shipping only a binary is a license violation, and §8 ends your rights the moment you do it.

**And it will be enforced.** In that order: a request to comply, then a DMCA takedown and a delisting request to whichever registry or marketplace carries it, then legal action as the copyright holder. Complying costs you a license file, a notice and a source link, so none of this needs to happen. Not sure whether what you're shipping complies? [Ask in an issue](https://github.com/dimitar-grigorov/mcp-file-tools/issues) — that's a normal question, and a cheaper one.

That's the enforceable part; the rest below is asked, not enforced.

**Leave the credit in.** Keeping `Copyright (C) 2026 Dimitar Grigorov` in `LICENSE` and in the source headers is the legal minimum. One line near the top of your README — *"Fork of [dimitar-grigorov/mcp-file-tools](https://github.com/dimitar-grigorov/mcp-file-tools)"* — is what actually tells a reader where this came from, and costs you nothing.

**Don't ship it under this project's identity.** A rebranded fork in a public registry (MCP Registry, plugin marketplaces, Smithery) carrying the same product name with the author swapped reads as if the original project moved — people install it believing it's this one, and file its bugs here. Give your fork a name of its own.

**Try upstream first.** A PR here beats carrying merge conflicts forever, and it puts your name on the commit rather than in a credits list. Maintaining a long-running fork? Open an issue about upstreaming the parts that fit — that's a welcome conversation.

## Credits

Ideas that started in someone else's fork and were reimplemented here:

- [@skyispainted](https://github.com/skyispainted) - GBK/GB18030, JSON-string array args, `edit_file` retry hint
- [@haobiao](https://github.com/haobiao) - GBK/GB18030, independently
- [Hugo Rosário](https://github.com/WTC-ZoneSoft) - merging MCP roots with CLI allowed dirs
- [@zoster81](https://github.com/zoster81) - UTF-16 line endings and grep, path containment fixes, write durability, BOM policy, ordered concurrency

A PR gets your name on the commit instead of this list.

## License

GPL-3.0 - see [LICENSE](LICENSE)

Copyright (C) 2026 Dimitar Grigorov. Free software, distributed WITHOUT ANY WARRANTY.
