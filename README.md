# MCP File Tools

MCP server that lets AI assistants work with legacy-encoded files. Reads and writes files in Windows-1251, CP1251, and other non-UTF-8 encodings that would otherwise corrupt data.

**Perfect for:** Delphi/Pascal projects, legacy VB6 apps, old PHP/HTML sites, config files with non-UTF-8 text.

## What It Does

Provides 5 tools for file operations with automatic encoding conversion:
- `read_text_file` - Read files with encoding detection/conversion
- `write_file` - Write files in specific encodings
- `list_directory` - Browse directories with pattern filtering
- `detect_encoding` - Auto-detect file encoding with confidence score
- `list_encodings` - Show all supported encodings

**Supported encodings:** UTF-8, CP1251/Windows-1251 (Cyrillic)

See [TOOLS.md](TOOLS.md) for detailed parameters and examples.

**Security:** All operations restricted to allowed directories only.

## Installation

### From MCP Registry (Recommended)

This server is published to the [Official MCP Registry](https://registry.modelcontextprotocol.io).  MCP clients that support the registry can install it directly:

```bash
# Claude Code
claude mcp add io.github.dimitar-grigorov/mcp-file-tools
# Or search for it
claude mcp search file-tools
```

For clients without registry support, use the manual installation methods below.

### Windows x64

```powershell
# Download
mkdir -Force "$env:LOCALAPPDATA\Programs\mcp-file-tools"
iwr "https://github.com/dimitar-grigorov/mcp-file-tools/releases/latest/download/mcp-file-tools_windows_amd64.exe" -OutFile "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools.exe"
# Install with Claude Code
claude mcp add file-tools "$env:LOCALAPPDATA\Programs\mcp-file-tools\mcp-file-tools.exe"
```

### Linux x64

```bash
# Download
mkdir -p ~/.local/bin
curl -L "https://github.com/dimitar-grigorov/mcp-file-tools/releases/latest/download/mcp-file-tools_linux_amd64" -o ~/.local/bin/mcp-file-tools
chmod +x ~/.local/bin/mcp-file-tools
# Install with Claude Code
claude mcp add file-tools ~/.local/bin/mcp-file-tools
```

### macOS ARM64

```bash
# Download
mkdir -p ~/.local/bin
curl -L "https://github.com/dimitar-grigorov/mcp-file-tools/releases/latest/download/mcp-file-tools_darwin_arm64" -o ~/.local/bin/mcp-file-tools
chmod +x ~/.local/bin/mcp-file-tools
# Install with Claude Code
claude mcp add file-tools ~/.local/bin/mcp-file-tools
```

### Go Install (All Platforms)

```bash
# Install with Go (requires Go 1.23+)
go install github.com/dimitar-grigorov/mcp-file-tools/cmd/mcp-file-tools@latest
# Add to Claude Code
claude mcp add file-tools $(go env GOPATH)/bin/mcp-file-tools
```

### Other Clients

For Claude Desktop, VSCode, or Cursor, use the downloaded binary path in your config:

**Claude Desktop** (`%APPDATA%\Claude\claude_desktop_config.json` on Windows, `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):
```json
{
  "mcpServers": {
    "file-tools": {
      "command": "/path/to/mcp-file-tools"
    }
  }
}
```

**VSCode / Cursor** (`.vscode/mcp.json`):
```json
{
  "mcpServers": {
    "file-tools": {
      "command": "/path/to/mcp-file-tools"
    }
  }
}
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

**Prerequisites:** Go 1.23+

```bash
# Run tests
go test ./...

# Build
go build -o mcp-file-tools ./cmd/mcp-file-tools
```

### Debugging with MCP Inspector

[MCP Inspector](https://github.com/modelcontextprotocol/inspector) provides a web UI for testing MCP servers.

**Prerequisites:** Node.js v18+

```bash
# Run with allowed directory (required)
npx @modelcontextprotocol/inspector go run ./cmd/mcp-file-tools -- /path/to/allowed/dir

# Or with built binary
npx @modelcontextprotocol/inspector ./mcp-file-tools.exe C:\Projects
```

Opens a browser where you can view tools, call them with custom arguments, and inspect responses.

### Manual Debugging

Run the server with an allowed directory and send JSON-RPC commands via stdin:

```bash
# Specify allowed directory
go run ./cmd/mcp-file-tools /path/to/project
```

Example commands (paste into terminal):

```json
{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_directory","arguments":{"path":"/path/to/project","pattern":"*.go"}}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"read_text_file","arguments":{"path":"/path/to/project/main.pas","encoding":"cp1251"}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"detect_encoding","arguments":{"path":"/path/to/project/file.txt"}}}
```

## License

GPL-3.0 - see [LICENSE](LICENSE)
