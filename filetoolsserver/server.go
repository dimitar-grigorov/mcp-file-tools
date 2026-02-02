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
const serverInstructions = `MCP filesystem server with non-UTF-8 encoding support (20 encodings: CP1251, KOI8-R, ISO-8859-x, etc).

PREFER THESE TOOLS for file operations when encoding matters:
- read_text_file: auto-detects encoding, returns UTF-8
- write_file: converts UTF-8 to target encoding (default: cp1251)
- edit_file: in-place edits with encoding support, returns diff
- detect_encoding: diagnose encoding issues (garbled text, � characters)

Workflow for non-UTF-8 files:
1. detect_encoding - identify file encoding
2. read_text_file or edit_file - read/modify with encoding
3. write_file with encoding - preserves original encoding

If "no allowed directories configured" error: add directory paths as args in .mcp.json.`

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
		Description: "Read file with encoding auto-detection, converts to UTF-8. USE THIS for non-UTF-8 files (Cyrillic, legacy codebases). Parameters: path (required), encoding, offset (1-indexed start line), limit. Returns totalLines for pagination.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Read Text File",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "read_text_file", h.HandleReadTextFile))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_multiple_files",
		Description: "Read multiple files concurrently with encoding support. Individual failures don't stop operation. Parameters: paths (required array), encoding (optional, auto-detected per file).",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Read Multiple Files",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "read_multiple_files", h.HandleReadMultipleFiles))

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
		Description: "Auto-detect file encoding with confidence score (0-100) and BOM detection. ALWAYS use this first when encountering � characters or garbled text. Parameter: path (required).",
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
		Description: "Get file/directory metadata: size, created/modified/accessed times, permissions, type. Parameter: path (required).",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get File Info",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "get_file_info", h.HandleGetFileInfo))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "directory_tree",
		Description: "DEPRECATED: Use 'tree' instead (85% fewer tokens). Returns JSON tree structure for compatibility with mcp-js-servers. Parameters: path (required), excludePatterns (optional).",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Directory Tree (JSON)",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "directory_tree", h.HandleDirectoryTree))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tree",
		Description: "Compact indented tree view (85% fewer tokens than directory_tree). Directories end with /. Parameters: path (required), maxDepth (0=unlimited), maxFiles (default 1000), dirsOnly, exclude.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Tree (Compact)",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "tree", h.HandleTree))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_files",
		Description: "Recursively search for files matching a glob pattern (*.ext or **/*.ext). Returns full paths. Parameters: path (required), pattern (required), excludePatterns.",
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
		Description: "Write file with encoding conversion from UTF-8. USE THIS for non-UTF-8 files (Cyrillic, legacy codebases). Parameters: path (required), content (required), encoding (default: cp1251).",
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
		Description: "Move or rename files/directories. Fails if destination exists. Parameters: source (required), destination (required).",
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
		Description: "Replace text sequences in a file with whitespace-flexible matching. Returns unified diff. Parameters: path (required), edits (array of {oldText, newText}), dryRun (preview without writing), encoding.",
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
