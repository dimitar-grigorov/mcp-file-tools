package filetoolsserver

import (
	"log/slog"

	"github.com/dimitar-grigorov/mcp-file-tools/filetoolsserver/handler"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/config"
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

// NewServer creates a new MCP server with all file tools registered.
// If logger is nil, logging middleware is disabled but recovery is still active.
// If cfg is nil, configuration is loaded from environment variables.
func NewServer(allowedDirs []string, logger *slog.Logger, cfg *config.Config) *mcp.Server {
	var handlerOpts []handler.Option
	if cfg != nil {
		handlerOpts = append(handlerOpts, handler.WithConfig(cfg))
	}
	h := handler.NewHandler(allowedDirs, handlerOpts...)

	impl := &mcp.Implementation{
		Name:    "mcp-file-tools",
		Version: Version,
	}

	serverOpts := &mcp.ServerOptions{
		Instructions:            serverInstructions,
		Logger:                  logger,
		InitializedHandler:      createInitializedHandler(h),
		RootsListChangedHandler: createRootsListChangedHandler(h),
	}
	server := mcp.NewServer(impl, serverOpts)

	// Register all tools using the new AddTool API with annotations
	// All handlers are wrapped with recovery middleware (and logging if logger is provided)

	// Read-only tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_text_file",
		Description: "Read files with automatic encoding detection and conversion to UTF-8. USE THIS instead of built-in Read tool when files may contain non-UTF-8 encodings. Auto-detects encoding if not specified. Parameters: path (required), encoding (optional), head (optional), tail (optional).",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Read Text File",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "read_text_file", h.HandleReadTextFile))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_directory",
		Description: "List files and directories with optional glob pattern filtering (e.g., *.pas, *.dfm). Parameters: path (required), pattern (optional, default: *).",
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Directory",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "list_directory", h.HandleListDirectory))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_encodings",
		Description: "List all supported file encodings (UTF-8, CP1251, CP1252, KOI8-R, ISO-8859-x, and others). Returns name, aliases, and description for each.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Encodings",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "list_encodings", h.HandleListEncodings))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "detect_encoding",
		Description: "Auto-detect file encoding with confidence score and BOM detection. ALWAYS use this first when you encounter � characters or unknown encoding. Returns encoding name, confidence percentage (0-100), and whether file has BOM. Parameter: path (required).",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Detect Encoding",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "detect_encoding", h.HandleDetectEncoding))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_allowed_directories",
		Description: "Returns the list of directories this server is allowed to access. Subdirectories are also accessible. If empty, user needs to add directory paths as args in .mcp.json.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Allowed Directories",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "list_allowed_directories", h.HandleListAllowedDirectories))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_file_info",
		Description: "Retrieve detailed metadata about a file or directory. Returns size, creation time, last modified time, last accessed time, permissions, and type (file/directory). Only works within allowed directories. Parameter: path (required).",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get File Info",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "get_file_info", h.HandleGetFileInfo))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "directory_tree",
		Description: "Get a recursive tree view of files and directories as a JSON structure. Each entry includes 'name', 'type' (file/directory), and 'children' for directories. Files have no children array, while directories always have a children array (which may be empty). The output is formatted with 2-space indentation for readability. Only works within allowed directories. Parameters: path (required), excludePatterns (optional array of glob patterns to exclude).",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Directory Tree",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "directory_tree", h.HandleDirectoryTree))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_files",
		Description: "Recursively search for files and directories matching a glob pattern. Use '*.ext' to match in current directory, '**/*.ext' to match recursively in all subdirectories. Returns full paths to matching items. Parameters: path (required), pattern (required), excludePatterns (optional array of patterns to skip).",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Search Files",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "search_files", h.HandleSearchFiles))

	// Write tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_directory",
		Description: "Create a directory recursively (mkdir -p). Succeeds silently if already exists. Parameter: path (required).",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Create Directory",
			ReadOnlyHint:    false,
			IdempotentHint:  true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handler.Wrap(logger, "create_directory", h.HandleCreateDirectory))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "write_file",
		Description: "Write files with encoding conversion from UTF-8. USE THIS instead of built-in Write tool when writing to non-UTF-8 files (legacy codebases, Cyrillic text). Default encoding is cp1251 (configurable via MCP_DEFAULT_ENCODING). Parameters: path (required), content (required), encoding (cp1251/windows-1251/utf-8, default: cp1251).",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Write File",
			ReadOnlyHint:    false,
			IdempotentHint:  true,
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
		},
	}, handler.Wrap(logger, "write_file", h.HandleWriteFile))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "move_file",
		Description: "Move or rename files and directories. Can move files between directories and rename them in a single operation. Fails if destination already exists. Works for both files and directories. Parameters: source (required), destination (required).",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Move File",
			ReadOnlyHint:    false,
			IdempotentHint:  false,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handler.Wrap(logger, "move_file", h.HandleMoveFile))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "edit_file",
		Description: "Make line-based edits to a text file. Each edit replaces exact text sequences with new content. Supports whitespace-flexible matching when exact match fails. Returns a git-style unified diff showing the changes. Parameters: path (required), edits (required array of {oldText, newText}), dryRun (optional bool, default false - if true, returns diff without writing), encoding (optional, auto-detected).",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Edit File",
			ReadOnlyHint:    false,
			IdempotentHint:  false,
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
		},
	}, handler.Wrap(logger, "edit_file", h.HandleEditFile))

	return server
}
