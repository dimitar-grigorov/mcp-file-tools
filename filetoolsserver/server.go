package filetoolsserver

import (
	"github.com/dimitar-grigorov/mcp-file-tools/filetoolsserver/handler"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version is set at build time via ldflags
var Version = "dev"

// Server instructions for AI assistants
const serverInstructions = `MCP filesystem server with non-UTF-8 encoding support.
Provides standard tools (read_text_file, write_file, list_directory) plus encoding detection (detect_encoding).
Use this server for legacy files (Delphi, C++, etc.) with Cyrillic or other non-UTF-8 text.`

// NewServer creates a new MCP server with all file tools registered
func NewServer(allowedDirs []string) *mcp.Server {
	h := handler.NewHandler(allowedDirs)

	opts := &mcp.ServerOptions{
		Instructions: serverInstructions,
	}
	server := mcp.NewServer("mcp-file-tools", Version, opts)

	// Register all tools
	server.AddTools(
		mcp.NewServerTool(
			"read_text_file",
			"Reads a file and returns its content. Supports optional head/tail parameters to read first or last N lines. For UTF-8 files, returns content as-is. For other encodings (cp1251, etc.), converts to UTF-8. Default: UTF-8.",
			h.HandleReadTextFile,
		),
		mcp.NewServerTool(
			"write_file",
			"Writes content to a file. For UTF-8, writes as-is. For other encodings (cp1251, etc.), converts from UTF-8. Default: cp1251.",
			h.HandleWriteFile,
		),
		mcp.NewServerTool(
			"list_directory",
			"Lists files and directories in the specified path. Optionally filter by pattern.",
			h.HandleListDirectory,
		),
		mcp.NewServerTool(
			"list_encodings",
			"Returns a list of all supported encodings.",
			h.HandleListEncodings,
		),
		mcp.NewServerTool(
			"detect_encoding",
			"Detects the encoding of a file. Returns encoding name, confidence percentage, and BOM presence.",
			h.HandleDetectEncoding,
		),
		mcp.NewServerTool(
			"list_allowed_directories",
			"Lists all directories accessible to this server.",
			h.HandleListAllowedDirectories,
		),
	)

	return server
}
