package filetoolsserver

import (
	"github.com/dimitar-grigorov/mcp-file-tools/filetoolsserver/handler"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version is set at build time via ldflags
var Version = "dev"

// Server instructions for AI assistants
const serverInstructions = `MCP filesystem server with non-UTF-8 encoding support.

IMPORTANT: If "no allowed directories configured" error occurs, inform user to add directory paths as args in .mcp.json config.

Workflow for non-UTF-8 files:
1. detect_encoding - check file encoding first
2. write_file with detected encoding - prevents corruption

Supports 20 encodings (CP1251, KOI8-R, ISO-8859-x, etc). Use list_encodings to see all.`

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
		Description: "Read files with automatic encoding detection and conversion to UTF-8. USE THIS instead of built-in Read tool when files may contain non-UTF-8 encodings. Auto-detects encoding if not specified. Parameters: path (required), encoding (optional), head (optional), tail (optional).",
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
		Description: "List all supported file encodings (UTF-8, CP1251, CP1252, KOI8-R, ISO-8859-x, and others). Returns name, aliases, and description for each.",
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
		Description: "Returns the list of directories this server is allowed to access. Subdirectories are also accessible. If empty, user needs to add directory paths as args in .mcp.json.",
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

	mcp.AddTool(server, &mcp.Tool{
		Name:        "directory_tree",
		Description: "Get a recursive tree view of files and directories as a JSON structure. Each entry includes 'name', 'type' (file/directory), and 'children' for directories. Files have no children array, while directories always have a children array (which may be empty). The output is formatted with 2-space indentation for readability. Only works within allowed directories. Parameters: path (required), excludePatterns (optional array of glob patterns to exclude).",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, h.HandleDirectoryTree)

	// Write tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_directory",
		Description: "Create a directory recursively (mkdir -p). Succeeds silently if already exists. Parameter: path (required).",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			IdempotentHint:  true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, h.HandleCreateDirectory)

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

	mcp.AddTool(server, &mcp.Tool{
		Name:        "move_file",
		Description: "Move or rename files and directories. Can move files between directories and rename them in a single operation. Fails if destination already exists. Works for both files and directories. Parameters: source (required), destination (required).",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    false,
			IdempotentHint:  false,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, h.HandleMoveFile)

	return server
}
