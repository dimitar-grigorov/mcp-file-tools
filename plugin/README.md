# mcp-file-tools (Claude Code plugin)

Installs the [`mcp-file-tools`](https://github.com/dimitar-grigorov/mcp-file-tools)
MCP server into Claude Code via `/plugin install`.

The server provides filesystem operations on text that isn't UTF-8: it auto-detects legacy
charsets (CP1251, CP1252, KOI8-R, ISO-8859, UTF-16, GBK/GB18030, ...), hands Claude UTF-8,
and writes back in the file's original encoding with BOM and line endings intact. The
[full list](https://github.com/dimitar-grigorov/mcp-file-tools#supported-encodings) is in
the main README.

## Install

```
/plugin marketplace add dimitar-grigorov/mcp-file-tools
/plugin install mcp-file-tools
```

## Bundled skill

`skills/fixing-text-encodings/` — diagnosing mojibake and auditing or bulk-converting a
legacy codebase. Claude loads it on demand when a task matches; it costs nothing until
then. Not needed for ordinary UTF-8 work.

## How it works

`.mcp.json` declares one MCP server (`file-tools`) launched as
`node ${CLAUDE_PLUGIN_ROOT}/bin/run.js`. On first launch the script downloads the
pinned release binary for your OS/arch, verifies its SHA-256 against the release
`checksums.txt`, caches it, then hands stdio to it. Later launches reuse the cache.

No directory configuration is needed: Claude Code sends the workspace folder via the
MCP roots protocol and the server allows it automatically.

## Requirements

Node.js 18+ on your PATH — `bin/run.js` is a Node script, and Claude Code ships as a
standalone binary that does not bundle Node. If the server shows as not connected,
check `node --version` first. The server binary itself is a standalone Go executable
with no runtime dependencies — see below to skip Node entirely.

## Alternative without the plugin

Install the binary with `go install
github.com/dimitar-grigorov/mcp-file-tools/v4/cmd/mcp-file-tools@latest` (or download it
from Releases), then add to `.mcp.json`:

```json
{
  "mcpServers": {
    "file-tools": { "command": "mcp-file-tools" }
  }
}
```
