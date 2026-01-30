package filetoolsserver

import (
	"github.com/dimitar-grigorov/mcp-file-tools/filetoolsserver/handler"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version is set at build time via ldflags
var Version = "dev"

// Server instructions for AI assistants
const serverInstructions = `MCP filesystem server with non-UTF-8 encoding support.

IMPORTANT: Use these tools instead of built-in Read/Write when:
- Files contain non-UTF-8 encodings (CP1251, Windows-1251, etc.)
- Working with legacy codebases (Delphi, VB6, C++, PHP)
- Files display � or other mojibake characters
- User mentions Cyrillic, Russian, or Eastern European text

Tools provided:
- read_text_file: Read files with encoding conversion (use instead of Read)
- write_file: Write files with encoding conversion (use instead of Write)
- list_directory: List files with pattern filtering
- detect_encoding: Auto-detect file encoding with confidence score
- list_encodings: Show all supported encodings
- list_allowed_directories: Show accessible directories
- get_file_info: Get file/directory metadata (size, times, permissions)

Always use detect_encoding first if encoding is unknown.`

// Helper for bool pointers (DestructiveHint defaults to true, so we need explicit false)
func boolPtr(b bool) *bool {
	return &b
}

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

	// Register all tools using the new AddTool API with annotations

	// Read-only tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_text_file",
		Description: "Read files with automatic encoding conversion to UTF-8. USE THIS instead of built-in Read tool when files contain non-UTF-8 encodings (CP1251, Windows-1251, etc.) or display � characters. Supports head/tail for reading first/last N lines. Parameters: path (required), encoding (cp1251/windows-1251/utf-8, default: utf-8), head (optional), tail (optional).",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, h.HandleReadTextFile)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_directory",
		Description: "List files and directories with optional glob pattern filtering (e.g., *.pas, *.dfm). Parameters: path (required), pattern (optional, default: *).",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, h.HandleListDirectory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_encodings",
		Description: "List all supported file encodings (UTF-8, CP1251, Windows-1251, etc.). Use this to see available encoding options before reading/writing files.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, h.HandleListEncodings)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "detect_encoding",
		Description: "Auto-detect file encoding with confidence score and BOM detection. ALWAYS use this first when you encounter � characters or unknown encoding. Returns encoding name, confidence percentage (0-100), and whether file has BOM. Parameter: path (required).",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, h.HandleDetectEncoding)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_allowed_directories",
		Description: "Returns the list of directories that this server is allowed to access. Subdirectories within these allowed directories are also accessible.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, h.HandleListAllowedDirectories)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_file_info",
		Description: "Retrieve detailed metadata about a file or directory. Returns size, creation time, last modified time, last accessed time, permissions, and type (file/directory). Only works within allowed directories. Parameter: path (required).",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, h.HandleGetFileInfo)

	// Write tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "write_file",
		Description: "Write files with encoding conversion from UTF-8. USE THIS instead of built-in Write tool when writing to non-UTF-8 files (legacy codebases, Cyrillic text). Default encoding is cp1251 for backward compatibility. Parameters: path (required), content (required), encoding (cp1251/windows-1251/utf-8, default: cp1251).",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			IdempotentHint:  true,
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
		},
	}, h.HandleWriteFile)

	return server
}
