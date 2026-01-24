# MCP File Tools

MCP server for file operations with legacy encoding support. Handles reading and writing files in non-UTF-8 encodings (Windows-1251, etc.) that AI assistants can't process natively.

## Tools

### File Operations

- **read_file**
  - Read file contents. UTF-8 files pass through unchanged; other encodings convert to UTF-8
  - Parameters: `path` (required), `encoding` (optional): utf-8, cp1251, windows-1251 (default: cp1251)

- **write_file**
  - Write content to file. UTF-8 writes as-is; other encodings convert from UTF-8
  - Parameters: `path` (required), `content` (required), `encoding` (optional): utf-8, cp1251, windows-1251 (default: cp1251)

### Directory Operations

- **list_directory**
  - List files and directories with optional pattern filtering
  - Parameters: `path` (required): Absolute path to directory, `pattern` (optional): Glob pattern like `*.pas` or `*.dfm` (default: `*`)

### Utility

- **list_encodings**
  - Returns all supported encodings
  - Parameters: None

## Supported Encodings

| Encoding | Aliases | Description |
|----------|---------|-------------|
| UTF-8 | utf8 | No conversion (passthrough) |
| CP1251 | windows-1251 | Cyrillic (Bulgarian, Russian, Serbian, Ukrainian) |

Planned: CP1250 (Central European), CP1252 (Western European), CP866 (DOS Cyrillic)

## Installation

### Pre-built Binaries

Download from [Releases](https://github.com/dimitar-grigorov/mcp-file-tools/releases):

- `mcp-file-tools-windows-amd64.exe`
- `mcp-file-tools-darwin-arm64` (Apple Silicon)
- `mcp-file-tools-darwin-amd64` (Intel Mac)
- `mcp-file-tools-linux-amd64`

### Build from Source

```bash
go install github.com/dimitar-grigorov/mcp-file-tools/cmd/mcp-file-tools@latest
```

Or clone and build:

```bash
git clone https://github.com/dimitar-grigorov/mcp-file-tools.git
cd mcp-file-tools
go build -o mcp-file-tools ./cmd/mcp-file-tools
```

## Usage

### Claude Code

```bash
claude mcp add file-tools -- "/path/to/mcp-file-tools"
```

### Claude Desktop / Cursor / VSCode

Add to your MCP configuration:

```json
{
  "mcpServers": {
    "file-tools": {
      "command": "/path/to/mcp-file-tools"
    }
  }
}
```

Configuration file locations:
- **Claude Desktop (Windows)**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Claude Desktop (macOS)**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **VSCode/Cursor**: `.vscode/mcp.json` in your project

## Use Case: Delphi Legacy Projects

Delphi 7/2007 projects store source files (`.pas`, `.dfm`) in Windows-1251 encoding for Cyrillic text. Standard file tools corrupt these files because they assume UTF-8.

This server lets AI assistants read and modify these files correctly:

```
User: Read MainForm.dfm and change the button caption to "Запази"
Assistant: [uses read_file with cp1251] → [modifies UTF-8 content] → [uses write_file with cp1251]
```

The original encoding is preserved.

## Development

```bash
# Run tests
go test ./...

# Build
go build -o mcp-file-tools ./cmd/mcp-file-tools

# Cross-compile
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o mcp-file-tools-windows-amd64.exe ./cmd/mcp-file-tools
```

## License

GPL-3.0 - see [LICENSE](LICENSE)
