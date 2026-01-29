# MCP File Tools

MCP server for file operations with legacy encoding support. Handles reading and writing files in non-UTF-8 encodings (Windows-1251, etc.) that AI assistants can't process natively.

**Security**: All file operations are restricted to explicitly allowed directories for safe operation.

## Tools

### File Operations

- **read_text_file**
  - Read file contents with optional partial reading (head/tail)
  - UTF-8 files pass through unchanged; other encodings convert to UTF-8
  - Parameters:
    - `path` (required): Path to the file
    - `encoding` (optional): utf-8 (default), cp1251, windows-1251
    - `head` (optional): Read only the first N lines
    - `tail` (optional): Read only the last N lines

- **write_file**
  - Write content to file. UTF-8 writes as-is; other encodings convert from UTF-8
  - Parameters:
    - `path` (required): Path to the file
    - `content` (required): Content to write
    - `encoding` (optional): utf-8, cp1251, windows-1251 (default: cp1251)

### Directory Operations

- **list_directory**
  - List files and directories with optional pattern filtering
  - Parameters:
    - `path` (required): Path to directory
    - `pattern` (optional): Glob pattern like `*.pas` or `*.dfm` (default: `*`)

### Encoding Tools

- **detect_encoding**
  - Detect the encoding of a file with confidence percentage
  - Returns encoding name, confidence, and BOM presence
  - Parameters: `path` (required)

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

### Security Model

The server uses an **allowed directories** system for security:
- File operations are restricted to allowed directories only
- **Automatic via MCP Roots Protocol**: Clients like Claude Desktop automatically provide workspace directories
- **Manual via CLI args**: Optionally specify directories at startup for clients that don't support roots
- If no directories are configured, all file operations will fail with a clear error message

### Claude Desktop / Cursor / VSCode

Add to your MCP configuration. The client will automatically provide workspace roots:

```json
{
  "mcpServers": {
    "file-tools": {
      "command": "/path/to/mcp-file-tools"
    }
  }
}
```

**Optional: Pre-configure directories via CLI args**
```json
{
  "mcpServers": {
    "file-tools": {
      "command": "/path/to/mcp-file-tools",
      "args": [
        "C:\\Projects\\MyProject",
        "C:\\Users\\YourName\\Documents"
      ]
    }
  }
}
```

**macOS/Linux example**:
```json
{
  "mcpServers": {
    "file-tools": {
      "command": "/usr/local/bin/mcp-file-tools",
      "args": [
        "/home/user/projects",
        "/home/user/documents"
      ]
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

**Prerequisites:**
- Go 1.21+
- [Delve](https://github.com/go-delve/delve) (for debugging): `go install github.com/go-delve/delve/cmd/dlv@latest`

```bash
# Run tests
go test ./...

# Build
go build -o mcp-file-tools ./cmd/mcp-file-tools

# Cross-compile
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o mcp-file-tools-windows-amd64.exe ./cmd/mcp-file-tools
```

### Debugging with MCP Inspector

[MCP Inspector](https://github.com/modelcontextprotocol/inspector) provides a web UI for testing MCP servers.

**Prerequisites:** [Node.js](https://nodejs.org/) v18+

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
