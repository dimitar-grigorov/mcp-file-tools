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

	impl := &mcp.Implementation{
		Name:    "mcp-file-tools",
		Version: Version,
	}

	opts := &mcp.ServerOptions{
		Instructions:            serverInstructions,
		InitializedHandler:      createInitializedHandler(h),
		RootsListChangedHandler: createRootsListChangedHandler(h),
	}
	server := mcp.NewServer(impl, opts)

	// Register all tools using the new AddTool API
	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_text_file",
		Description: "Reads a file and returns its content. Supports optional head/tail parameters to read first or last N lines. For UTF-8 files, returns content as-is. For other encodings (cp1251, etc.), converts to UTF-8. Default: UTF-8.",
	}, h.HandleReadTextFile)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "write_file",
		Description: "Writes content to a file. For UTF-8, writes as-is. For other encodings (cp1251, etc.), converts from UTF-8. Default: cp1251.",
	}, h.HandleWriteFile)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_directory",
		Description: "Lists files and directories in the specified path. Optionally filter by pattern.",
	}, h.HandleListDirectory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_encodings",
		Description: "Returns a list of all supported encodings.",
	}, h.HandleListEncodings)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "detect_encoding",
		Description: "Detects the encoding of a file. Returns encoding name, confidence percentage, and BOM presence.",
	}, h.HandleDetectEncoding)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_allowed_directories",
		Description: "Returns the list of directories that this server is allowed to access. Subdirectories within these allowed directories are also accessible.",
	}, h.HandleListAllowedDirectories)

	return server
}
