// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

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
const serverInstructions = `MCP filesystem server with non-UTF-8 encoding support (24 encodings: CP1251, KOI8-R, ISO-8859-x, GBK/GB18030, etc).

PREFER THESE TOOLS over built-in Read/Write/Grep for file operations when encoding matters:
- read_text_file: auto-detects encoding, returns UTF-8. Use offset/limit for files >2000 lines.
- write_file: converts UTF-8 content to target encoding (keeps an existing file's encoding; new files default to utf-8)
- edit_file: in-place edits with encoding support, returns unified diff. Use dryRun=true to preview changes before applying.
- grep_text_files: encoding-aware regex search across files
- detect_encoding: diagnose encoding issues (garbled text, � characters)

Workflow:
1. Locate: tree, search_files (by name), grep_text_files (by content)
2. Read: read_text_file — encoding is auto-detected, no detect_encoding call needed first
3. Modify: edit_file, in place, keeps the encoding. Use write_file only for new files or a full rewrite; never read+write just to edit.

Diagnose only when output looks wrong: detect_encoding, detect_line_endings (mixed CRLF/LF), manage_bom (a UTF-8 BOM breaks PHP/shell scripts). convert_encoding re-encodes a whole file — pass backup=true.

"no allowed directories configured" — add paths as args in .mcp.json.

Call check_for_updates once per session; report an available update to the user.

Bugs and PRs: https://github.com/dimitar-grigorov/mcp-file-tools`

// Helper for bool pointers (DestructiveHint defaults to true, so we need explicit false)
func boolPtr(b bool) *bool {
	return &b
}

// NewServer registers all file tools. Nil logger keeps recovery but drops
// logging middleware; nil cfg reads config from the environment.
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
		Instructions:       serverInstructions,
		Logger:             logger,
		InitializedHandler: createInitializedHandler(h),
		// Deprecated in 2026-07-28, kept for older clients.
		RootsListChangedHandler: createRootsListChangedHandler(h),
		// Explicit: unset, the SDK also advertises logging and listChanged.
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{},
		},
	}
	server := mcp.NewServer(impl, serverOpts)

	// Repair array/object args some MCP clients send as JSON-encoded strings.
	server.AddReceivingMiddleware(handler.RepairStringifiedArrayArgs)

	// Register all tools using the new AddTool API with annotations
	// All handlers are wrapped with recovery middleware (and logging if logger is provided)

	// Read-only tools
	mcp.AddTool(server, &mcp.Tool{
		Name: "read_text_file",
		Description: "Read file with encoding auto-detection, converts to UTF-8. PREFER THIS over built-in Read for non-UTF-8 files (Cyrillic, legacy codebases). For files >2000 lines, use offset/limit to paginate. Returns totalLines and fileSizeBytes for planning subsequent reads. Use maxCharacters to cap output size and prevent token overflow. Parameters: path (required), encoding (optional, auto-detected), offset (1-indexed start line), limit (max lines to return), maxCharacters (optional, truncates content). " +
			`Example — paging a 12000-line file: {"path": "D:\\src\\app.pas", "offset": 1, "limit": 2000}, then offset 2001, until offset exceeds totalLines.`,
		Meta: mcp.Meta{"anthropic/maxResultSizeChars": 200000},
		Annotations: &mcp.ToolAnnotations{
			Title:         "Read Text File",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "read_text_file", h.HandleReadTextFile))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_multiple_files",
		Description: "Read multiple files concurrently with encoding support. PREFER THIS when reading several non-UTF-8 files at once. Individual failures don't stop the batch — partial results are returned. Parameters: paths (required array), encoding (optional, auto-detected per file).",
		Meta:        mcp.Meta{"anthropic/maxResultSizeChars": 300000},
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
		Description: "List all 24 supported encodings with name, aliases, and description. Use this to find the correct encoding name for read/write/convert operations.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Encodings",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "list_encodings", h.HandleListEncodings))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "detect_encoding",
		Description: "Auto-detect file encoding with confidence score (0-100) and BOM detection. ALWAYS use this first when encountering garbled text or � characters. Use before read_text_file to determine the correct encoding. Parameters: path (required), mode (sample=fast default, chunked=thorough, full=entire file).",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Detect Encoding",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "detect_encoding", h.HandleDetectEncoding))

	mcp.AddTool(server, &mcp.Tool{
		Name: "grep_text_files",
		Description: "Regex search in file contents with encoding support. PREFER THIS over built-in Grep when searching non-UTF-8 files or when encoding-aware matching is needed. Parameters: pattern (required regex), paths (required array of files/dirs), caseSensitive (default: true), contextBefore/After (lines), maxMatches (default 1000), include/exclude (globs), encoding. " +
			"Entries in paths may be directories (searched recursively) or individual files. include/exclude are single glob STRINGS matched against the file name, not arrays: use \"*.pas\", not [\"*.pas\"]. Brace sets (\"*.{pas,dfm}\") and directory-qualified patterns (\"src/*.pas\") do NOT match. " +
			`Example: {"pattern": "TCustomer", "paths": ["D:\\proj\\src"], "include": "*.pas", "contextAfter": 2}`,
		Meta: mcp.Meta{"anthropic/maxResultSizeChars": 300000},
		Annotations: &mcp.ToolAnnotations{
			Title:         "Grep Text Files",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "grep_text_files", h.HandleGrep))

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
		Description: "Get file/directory metadata: size, timestamps, permissions, type. Use this to check file size before reading large files with read_text_file. Parameter: path (required).",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Get File Info",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "get_file_info", h.HandleGetFileInfo))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tree",
		Description: "Compact indented tree view of directory structure. PREFER THIS for directory visualization. Set showEncoding=true to detect and display file encodings (e.g., for auditing legacy codebases). Parameters: path (required), maxDepth (0=unlimited), maxFiles (default 1000), dirsOnly (bool), exclude (array of patterns), showEncoding (bool, shows detected encoding per file).",
		Meta:        mcp.Meta{"anthropic/maxResultSizeChars": 200000},
		Annotations: &mcp.ToolAnnotations{
			Title:         "Tree (Compact)",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "tree", h.HandleTree))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_files",
		Description: "Recursively search for files matching a glob pattern (*.ext or **/*.ext). Returns full paths. Parameters: path (required), pattern (required), excludePatterns, maxResults (default 10000).",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Search Files",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "search_files", h.HandleSearchFiles))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "detect_line_endings",
		Description: "Detect line ending style (crlf/lf/mixed/none) and find inconsistent lines. Useful for diagnosing mixed line ending issues in cross-platform legacy codebases. Returns dominant style, total lines, and line numbers with minority endings. Parameters: path (required), encoding (optional, auto-detected including most BOM-less UTF-16 text — pass utf-16-le or utf-16-be if a very short or unusual file is misread).",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Detect Line Endings",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "detect_line_endings", h.HandleDetectLineEndings))

	// Write tools
	mcp.AddTool(server, &mcp.Tool{
		Name:        "manage_bom",
		Description: "Detect, strip, or add Unicode BOM (Byte Order Mark). UTF-8 BOM breaks PHP/shell scripts; UTF-16 files need BOMs. Parameters: path (required), action (required: \"detect\"|\"strip\"|\"add\"), encoding (required for \"add\": utf-8, utf-16-le, utf-16-be, utf-32-le, utf-32-be).",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Manage BOM",
			ReadOnlyHint:    false,
			IdempotentHint:  true,
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
		},
	}, handler.Wrap(logger, "manage_bom", h.HandleManageBom))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "change_line_endings",
		Description: "Convert line endings in a file to LF or CRLF. Use after detect_line_endings to fix mixed or wrong line endings. UTF-16 is converted per code unit and the BOM is preserved. Returns original style, new style, and number of lines changed. No-op if file already uses the target style. Parameters: path (required), style (required: \"lf\" or \"crlf\"), encoding (optional, auto-detected including most BOM-less UTF-16 text — pass utf-16-le or utf-16-be if a very short or unusual file is misread).",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Change Line Endings",
			ReadOnlyHint:    false,
			IdempotentHint:  true,
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
		},
	}, handler.Wrap(logger, "change_line_endings", h.HandleChangeLineEndings))

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
		Name: "write_file",
		Description: "Write file with encoding conversion from UTF-8. PREFER THIS over built-in Write for non-UTF-8 files — converts UTF-8 content to target encoding, preserving legacy compatibility. Parameters: path (required), content (required), encoding (optional: defaults to an existing file's detected encoding, else utf-8), bom (optional: \"auto\" default — BOM for UTF-16 targets and keeps a BOM the file already had; \"always\", \"never\", \"preserve\"). Use after read_text_file to preserve original encoding. " +
			"bom modes: \"auto\" writes a BOM for utf-16-* targets and otherwise keeps one only if the file already had a BOM of the same encoding; \"preserve\" keeps the existing BOM even when the encoding changed; \"never\" strips it; \"always\" fails on encodings that have no BOM (e.g. cp1251). " +
			`Example — strip a UTF-8 BOM that breaks PHP: {"path": "D:\\www\\index.php", "content": "<?php ...", "bom": "never"}`,
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
		Name:        "copy_file",
		Description: "Copy a file. Fails if destination exists. Parameters: source (required), destination (required).",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Copy File",
			ReadOnlyHint:    false,
			IdempotentHint:  true,
			DestructiveHint: boolPtr(false),
			OpenWorldHint:   boolPtr(false),
		},
	}, handler.Wrap(logger, "copy_file", h.HandleCopyFile))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_file",
		Description: "Delete a file. Does not delete directories. Parameter: path (required).",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Delete File",
			ReadOnlyHint:    false,
			IdempotentHint:  false,
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
		},
	}, handler.Wrap(logger, "delete_file", h.HandleDeleteFile))

	// WrapContentOnly: returns readable diff text instead of StructuredContent JSON.
	mcp.AddTool(server, &mcp.Tool{
		Name: "edit_file",
		Description: "Replace text in a file with whitespace-flexible matching. Returns unified diff. Supports non-UTF-8 via encoding param. " +
			"In 'ask before edits' mode: ALWAYS call with dryRun=true first, show the diff, then dryRun=false after user confirms. " +
			"With auto-edit permissions: call directly with dryRun=false. " +
			"On no match, the error hints the closest matching content — use it to fix oldText and retry. " +
			"Parameters: path, edits [{oldText, newText}], dryRun (false), encoding (auto). " +
			"Edits apply in order and each replaces only the FIRST match. Matching ignores per-line leading/trailing whitespace and CRLF/LF differences, but interior spacing must match; newText is re-indented to the file. " +
			`Example: {"path": "D:\\src\\unit1.pas", "edits": [{"oldText": "i: Integer;", "newText": "i: NativeInt;"}, {"oldText": "for i := 0 to 10 do", "newText": "for i := 0 to 20 do"}], "dryRun": true}`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Edit File",
			ReadOnlyHint:    false,
			IdempotentHint:  false,
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
		},
	}, handler.WrapContentOnly(logger, "edit_file", h.HandleEditFile))

	mcp.AddTool(server, &mcp.Tool{
		Name: "convert_encoding",
		Description: "Convert file from one encoding to another. Use after detect_encoding to identify the source. A source BOM is stripped before decoding, and a conflict between the BOM and an explicit 'from' is an error. No-op if the file already holds the target bytes. Parameters: path (required), from (source encoding, auto-detected if omitted), to (target encoding, required), backup (create .bak file before converting, default: false), bom (optional: \"auto\" default — BOM for UTF-16 targets and keeps a same-encoding source BOM; \"always\", \"never\", \"preserve\"). IMPORTANT: Use backup=true for irreversible conversions. " +
			"Omit 'from' to auto-detect; pass it only to override detection on a file that misdetects. A narrowing conversion (e.g. utf-8 to cp1251) fails rather than corrupting text if the content has characters the target lacks. " +
			`Example: {"path": "D:\\legacy\\data.txt", "to": "utf-8", "backup": true} detects the source and leaves data.txt.bak.`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Convert Encoding",
			ReadOnlyHint:    false,
			IdempotentHint:  true,
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
		},
	}, handler.Wrap(logger, "convert_encoding", h.HandleConvertEncoding))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "check_for_updates",
		Description: "Check if a newer version of mcp-file-tools is available. Returns current version, latest version, and update instructions if outdated. Uses cached result (max 1 GitHub API call per 2h). Call once at the start of each session.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Check for Updates",
			ReadOnlyHint: true,
		},
	}, handler.Wrap(logger, "check_for_updates", handler.NewCheckUpdateHandler(Version)))

	return server
}
