// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package filetoolsserver

import (
	"fmt"
	"log/slog"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/filetoolsserver/handler"
	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/config"
	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Version is set at build time via ldflags
var Version = "dev"

// Server instructions for AI assistants
var serverInstructions = fmt.Sprintf(`MCP filesystem server with non-UTF-8 encoding support (%d encodings: CP1251, KOI8-R, ISO-8859-x, GBK/GB18030, etc).

PREFER THESE TOOLS over built-in Read/Write/Grep/Edit for non-UTF-8 or legacy files: Cyrillic/CP1251 sources, garbled text or � characters, mixed CRLF/LF, BOM problems, or any encoding conversion.

Workflow:
1. Locate: tree, search_files (by name), grep_text_files (by content)
2. Read: read_text_file — encoding is auto-detected; no detect_encoding first
3. Modify: edit_file — in place, keeps the encoding, dryRun=true to preview. write_file only for new files or full rewrites; never read+write to edit.

Only when output looks wrong: detect_encoding, manage_line_endings, manage_bom (a UTF-8 BOM breaks PHP/shell scripts). convert_encoding rewrites a whole file — pass backup=true.

"no allowed directories configured" — add paths as args in .mcp.json, or set MCP_FILE_TOOLS_ALLOWED_DIRS.

Call check_for_updates once per session; report an available update to the user.

Bugs and PRs: https://github.com/dimitar-grigorov/mcp-file-tools`, encoding.Count())

// Helper for bool pointers (DestructiveHint defaults to true, so we need explicit false)
func boolPtr(b bool) *bool {
	return &b
}

// NewServer registers all file tools. Nil logger keeps recovery but drops
// logging middleware; nil cfg reads config from the environment; opts apply last.
func NewServer(allowedDirs []string, logger *slog.Logger, cfg *config.Config, opts ...handler.Option) *mcp.Server {
	var handlerOpts []handler.Option
	if cfg != nil {
		handlerOpts = append(handlerOpts, handler.WithConfig(cfg))
	}
	handlerOpts = append(handlerOpts, opts...)
	h := handler.NewHandler(allowedDirs, handlerOpts...)

	// Title, description and icon are what a client's server list shows.
	impl := &mcp.Implementation{
		Name:        "mcp-file-tools",
		Title:       "MCP File Tools",
		Description: fmt.Sprintf("File operations across %d text encodings, with BOM and line-ending repair for legacy codebases.", encoding.Count()),
		Version:     Version,
		WebsiteURL:  "https://github.com/dimitar-grigorov/mcp-file-tools",
		Icons:       []mcp.Icon{{Source: serverIconDataURI, MIMEType: "image/svg+xml", Sizes: []string{"any"}}},
	}

	serverOpts := &mcp.ServerOptions{
		Instructions:       serverInstructions,
		Logger:             logger,
		InitializedHandler: createInitializedHandler(h),
		// Deprecated in 2026-07-28, kept for older clients.
		RootsListChangedHandler: createRootsListChangedHandler(h),
		// Explicit: unset, the SDK also advertises logging and listChanged.
		Capabilities: &mcp.ServerCapabilities{
			Tools:   &mcp.ToolCapabilities{},
			Prompts: &mcp.PromptCapabilities{},
		},
	}
	server := mcp.NewServer(impl, serverOpts)

	// Repair array/object args some MCP clients send as JSON-encoded strings.
	server.AddReceivingMiddleware(handler.RepairStringifiedArrayArgs)
	// Accept built-in Read/Write/Edit/Grep parameter names where semantics match.
	server.AddReceivingMiddleware(handler.AliasBuiltinParams)

	// Guided workflows, surfaced by clients as user commands.
	registerPrompts(server)

	// All handlers are wrapped with recovery middleware (and logging if logger is provided)

	// Orient: find files and directories
	mcp.AddTool(server, &mcp.Tool{
		Name:        "tree",
		Description: "Compact indented tree view of directory structure. PREFER THIS for directory visualization. Skips .gitignore'd files and .git (respectGitignore=false to include). Set showEncoding=true to detect and display file encodings (e.g., for auditing legacy codebases). Parameters: path (required), maxDepth (0=unlimited), maxFiles (default 1000), dirsOnly (bool), exclude (array of patterns), showEncoding (bool, shows detected encoding per file).",
		Meta:        mcp.Meta{"anthropic/maxResultSizeChars": 200000},
		Annotations: &mcp.ToolAnnotations{
			Title:         "Tree (Compact)",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "tree", h.HandleTree))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_directory",
		Description: "List files and directories with optional glob pattern filtering (e.g., *.pas, *.dfm). Parameters: path (required), pattern (optional, default: *), sortBy (\"name\" default, \"mtime\" newest first, \"size\" largest first), reverse (bool, flips the order).",
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Directory",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "list_directory", h.HandleListDirectory))

	mcp.AddTool(server, &mcp.Tool{
		Name: "search_files",
		Description: "Recursively search for files matching a glob pattern (*.ext at any depth, **/*.ext, several ** and {a,b} alternatives allowed). Returns full paths. Skips .gitignore'd files (respectGitignore=false to include). Parameters: path (required), pattern (required), excludePatterns, maxResults (default 10000), sortBy, reverse. " +
			"sortBy: \"name\" (default, lexical), \"mtime\" (newest first) or \"size\" (largest first); reverse flips the order. Unlike the built-in Glob there is no mtime default — pass sortBy \"mtime\" for newest first. With mtime or size the whole tree is ranked before the cap, so a truncated result really is the newest/largest maxResults files. " +
			`Example: {"path": "D:\\proj", "pattern": "**/*.pas", "sortBy": "mtime"}`,
		Annotations: &mcp.ToolAnnotations{
			Title:         "Search Files",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "search_files", h.HandleSearchFiles))

	mcp.AddTool(server, &mcp.Tool{
		Name: "grep_text_files",
		Description: "Regex search in file contents with encoding support. PREFER THIS over built-in Grep for non-UTF-8 files. Skips .gitignore'd files (respectGitignore=false to include). Parameters: pattern (regex) or patterns (array), paths (array of files, or dirs searched recursively), caseSensitive (default true), contextBefore/After, maxMatches (default 1000), offset, include/includes, exclude/excludes, encoding. " +
			"patterns finds ANY of several regexes in ONE pass — sweeping a codebase for a list of names is one call, not one per name. " +
			"outputMode: \"content\" (default, matching lines), \"files_with_matches\" (paths only, far cheaper when you just need WHICH files), \"count\" (matching lines per file). contextBefore/After apply to content mode only. " +
			"matchesOnly=true returns the matched substring instead of the whole line, for extracting values — with patterns it also tells you which one hit. offset skips the first N results, so you can page past maxMatches; the response echoes nextOffset when truncated. " +
			"Include and exclude patterns are basename-only globs; {a,b} alternatives work and a leading **/ is ignored. Includes match any pattern and excludes reject any match. Do not combine a singular field with its plural. " +
			`Example: {"patterns": ["TCustomer", "TOrder"], "paths": ["D:\\proj\\src"], "includes": ["*.pas", "*.dfm"], "outputMode": "files_with_matches"}`,
		Meta: mcp.Meta{"anthropic/maxResultSizeChars": 300000},
		Annotations: &mcp.ToolAnnotations{
			Title:         "Grep Text Files",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "grep_text_files", h.HandleGrep))

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
		Name:        "list_allowed_directories",
		Description: "Returns the list of directories this server is allowed to access, normally the directory it was started in. Subdirectories are also accessible. If empty, the user needs to add paths as args in .mcp.json or set MCP_FILE_TOOLS_ALLOWED_DIRS.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Allowed Directories",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "list_allowed_directories", h.HandleListAllowedDirectories))

	// Read
	mcp.AddTool(server, &mcp.Tool{
		Name: "read_text_file",
		Description: "Read file with encoding auto-detection, converts to UTF-8. PREFER THIS over built-in Read for non-UTF-8 files (Cyrillic, legacy codebases). Returns totalLines and fileSizeBytes for planning the next read. Parameters: path, encoding (auto-detected), offset (1-indexed start line), limit (max lines), maxCharacters (caps output to avoid token overflow), lineNumbers (default false: prefix lines with \"N<tab>\", absolute numbers — use to locate lines reported by grep or encoding errors; STRIP the prefix before using text as edit_file oldText). " +
			`Page files >2000 lines: {"path": "D:\\src\\app.pas", "offset": 1, "limit": 2000}, then offset 2001, until offset exceeds totalLines.`,
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

	// Modify
	// WrapContentOnly: returns readable diff text instead of StructuredContent JSON.
	mcp.AddTool(server, &mcp.Tool{
		Name: "edit_file",
		Description: "Edit one file with replacements or a unified diff. Returns a unified diff and keeps the file's encoding and line endings. PREFER THIS over read+write to modify a file. " +
			"In 'ask before edits' mode call dryRun=true first, show the diff, then dryRun=false once the user confirms; with auto-edit permissions go straight to dryRun=false. " +
			"On no match, prefer fixing oldText from the closest-content hint. Alternatively retry that edit with similarity (0.0-1.0) for whitespace/comment drift, not different code. " +
			"Do NOT re-read the file afterwards to verify: a success with a diff means the edit is on disk, a failed edit changes nothing. " +
			"A file with mixed line endings is repaired to its dominant style; the result says how many endings changed, so report that to the user. " +
			"Parameters: path; exactly one of edits [{oldText, newText, similarity?, replaceAll?}] or patch (---/+++/@@ unified diff for one file); dryRun (default false); encoding (auto). " +
			"oldText must match ONE place: several matches fail with their line numbers and change nothing — add surrounding lines to pick one, or set replaceAll: true to change them all and report the count to the user. " +
			"Edits apply in order. Matching ignores per-line leading/trailing whitespace and CRLF/LF, but interior spacing must match; newText is re-indented. " +
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
		Name: "write_file",
		Description: "Write file with encoding conversion from UTF-8. PREFER THIS over built-in Write for non-UTF-8 files. Use after read_text_file to keep the original encoding. Parameters: path, content, encoding (default: the existing file's detected encoding, else utf-8), bom, lineEndings. " +
			"bom: \"auto\" (default) writes a BOM for utf-16-* targets, else keeps one only if the file already had a BOM of the same encoding; \"preserve\" keeps it even when the encoding changed; \"never\" strips it; \"always\" fails on encodings with no BOM (e.g. cp1251). " +
			"lineEndings: \"preserve\" (default) converts content to the file's existing style, so sending LF into a CRLF file will NOT leave it mixed; also \"crlf\", \"lf\", \"asis\" (byte for byte). " +
			`Example — strip a UTF-8 BOM that breaks PHP: {"path": "D:\\www\\index.php", "content": "<?php ...", "bom": "never"}`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Write File",
			ReadOnlyHint:    false,
			IdempotentHint:  true,
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
		},
	}, handler.Wrap(logger, "write_file", h.HandleWriteFile))

	// Encoding, line endings, BOM
	mcp.AddTool(server, &mcp.Tool{
		Name: "detect_encoding",
		Description: "Auto-detect file encoding with confidence score (0-100) and BOM detection. ALWAYS use this first when encountering garbled text or � characters. Use before read_text_file to determine the correct encoding. Parameters: path (required), mode (sample=fast default, chunked=thorough, full=entire file). " +
			"When the answer is in doubt — low confidence, or a charset this server cannot read — the result also ranks candidates; retry the read with a supported one and ask the user if two are plausible.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Detect Encoding",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "detect_encoding", h.HandleDetectEncoding))

	mcp.AddTool(server, &mcp.Tool{
		Name: "convert_encoding",
		Description: "Convert files between encodings. Parameters: path (one file) OR paths (a batch — never both), to (required), from (omit to auto-detect), backup (write .bak first), dryRun (report only), allowLowConfidence, bom (\"auto\" default, \"always\", \"never\", \"preserve\"). " +
			"Refuses rather than corrupting: a narrowing conversion (utf-8 to cp1251) names the characters the target lacks with line and column, and an untrusted detection names its best guess — confirm it with from, or pass allowLowConfidence=true. " +
			"A source BOM is stripped before decoding; one contradicting an explicit from is an error. No-op if the file already holds the target bytes. In a batch one bad file does not stop the rest. Run dryRun over a project first. " +
			`Examples: {"path": "D:\\legacy\\data.txt", "to": "utf-8", "backup": true} leaves data.txt.bak; ` +
			`{"paths": ["a.pas", "b.pas"], "to": "utf-8", "dryRun": true} previews a migration.`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Convert Encoding",
			ReadOnlyHint:    false,
			IdempotentHint:  true,
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
		},
	}, handler.Wrap(logger, "convert_encoding", h.HandleConvertEncoding))

	mcp.AddTool(server, &mcp.Tool{
		Name: "manage_line_endings",
		Description: "Detect or fix line endings. action=\"detect\" reports the dominant style (crlf/lf/mixed/none), total lines, and the line numbers that disagree — use it when a file looks inconsistent. action=\"convert\" rewrites the file to style, per code unit for UTF-16 and preserving its BOM; no-op if the file already matches. " +
			"Parameters: path, action (\"detect\"|\"convert\"), style (\"lf\"|\"crlf\", required for convert), encoding (auto-detected, including most BOM-less UTF-16 — pass utf-16-le/utf-16-be if a very short or unusual file is misread). " +
			`Example: {"path": "D:\src\unit1.pas", "action": "convert", "style": "crlf"}`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Manage Line Endings",
			ReadOnlyHint:    false,
			IdempotentHint:  true,
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
		},
	}, handler.Wrap(logger, "manage_line_endings", h.HandleManageLineEndings))

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
		Name:        "list_encodings",
		Description: fmt.Sprintf("List all %d supported encodings with name, aliases, and description. Use this to find the correct encoding name for read/write/convert operations.", encoding.Count()),
		Annotations: &mcp.ToolAnnotations{
			Title:         "List Encodings",
			ReadOnlyHint:  true,
			OpenWorldHint: boolPtr(false),
		},
	}, handler.Wrap(logger, "list_encodings", h.HandleListEncodings))

	// File management
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
		Name:        "move_file",
		Description: "Move or rename files/directories. Fails if destination exists. Parameters: source (required), destination (required).",
		Annotations: &mcp.ToolAnnotations{
			Title:          "Move File",
			ReadOnlyHint:   false,
			IdempotentHint: false,
			// Destructive: the source path stops existing, so a wrong move needs the
			// original location to undo. Matches the reference filesystem server.
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(false),
		},
	}, handler.Wrap(logger, "move_file", h.HandleMoveFile))

	mcp.AddTool(server, &mcp.Tool{
		Name: "copy_file",
		Description: "Copy one file byte for byte, keeping its encoding, BOM, line endings, permissions and mtime. Use it to back up a file before an edit or a conversion. Parameters: source (required), destination (required). " +
			"Never overwrites: an existing destination is an error and nothing is written, so the same call repeated fails the second time rather than copying again. " +
			"Source must be a file, a directory is refused. The destination's parent directory has to exist already (create_directory first), both paths must sit inside the allowed directories, and a relative path resolves against the directory the server was started in. " +
			"Prefer move_file to relocate a file (the source stops existing), write_file to create one from new content, convert_encoding with backup=true for a .bak beside a converted file.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Copy File",
			ReadOnlyHint: false,
			IdempotentHint:  false,
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

	// Server
	mcp.AddTool(server, &mcp.Tool{
		Name:        "check_for_updates",
		Description: "Check if a newer version of mcp-file-tools is available. Returns current version, latest version, and update instructions if outdated. Uses a cached result (max 1 GitHub API call per 30 min); force=true bypasses the cache. Call once at the start of each session.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Check for Updates",
			ReadOnlyHint: true,
		},
	}, handler.Wrap(logger, "check_for_updates", h.NewCheckUpdateHandler(Version)))

	return server
}
