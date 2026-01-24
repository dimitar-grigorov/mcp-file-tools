package filetoolsserver

import (
	"github.com/dimitar-grigorov/mcp-file-tools/filetoolsserver/handler"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version is set at build time via ldflags
var Version = "dev"

// NewServer creates a new MCP server with all file tools registered
func NewServer() *mcp.Server {
	h := handler.NewHandler()

	server := mcp.NewServer("mcp-file-tools", Version, nil)

	// Register all tools
	server.AddTools(
		mcp.NewServerTool(
			"read_file",
			"Reads a file and returns its content. For UTF-8 files, returns content as-is. For other encodings (cp1251, etc.), converts to UTF-8. Default: cp1251.",
			h.HandleReadFile,
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
	)

	return server
}
