// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package filetoolsserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerPrompts adds the guided workflows. Clients surface these as user
// commands (Claude Code: /mcp__file-tools__<name>), so each one is a complete
// task brief the model can execute with the tools alone.
func registerPrompts(server *mcp.Server) {
	server.AddPrompt(&mcp.Prompt{
		Name:        "audit_encodings",
		Title:       "Audit project encodings",
		Description: "Survey a directory tree and report the encoding, BOM and line-ending situation.",
		Arguments: []*mcp.PromptArgument{
			{Name: "path", Description: "Directory to audit", Required: true},
		},
	}, promptText(func(args map[string]string) string {
		return fmt.Sprintf(`Audit the text encodings under %s using the file-tools MCP server.

1. Run tree with showEncoding=true (add exclude patterns for build output and VCS dirs).
2. Summarise: files per encoding, files with a BOM, anything detected with low confidence.
3. Spot-check one low-confidence file with detect_encoding mode="chunked" before trusting it.
4. Report as a short table: encoding -> count -> example files. Flag mixed-encoding directories,
   UTF-8 BOMs on script files (they break PHP/shell), and any file the detector could not read.
Do not modify anything - this is a read-only audit.`, args["path"])
	}))

	server.AddPrompt(&mcp.Prompt{
		Name:        "fix_mojibake",
		Title:       "Fix garbled text",
		Description: "Diagnose and repair a file showing � characters or garbled Cyrillic/accented text.",
		Arguments: []*mcp.PromptArgument{
			{Name: "path", Description: "File that displays wrong", Required: true},
		},
	}, promptText(func(args map[string]string) string {
		return fmt.Sprintf(`The file %s displays garbled text. Diagnose and fix it with the file-tools MCP server.

1. detect_encoding mode="chunked" - note the encoding and confidence.
2. read_text_file with the detected encoding. If the text now reads correctly, the file is fine
   and the *consumer* is decoding it wrongly - report that and stop.
3. If it is still garbled, the bytes are double-encoded or mis-saved. Try read_text_file with
   the likely original encodings (cp1251, cp1252, koi8-r) until one renders real words.
4. Confirm the correct reading with the user BEFORE writing anything.
5. Fix with convert_encoding backup=true, then read the file back to verify.
Never convert on a guess - a wrong conversion is worse than the mojibake.`, args["path"])
	}))

	server.AddPrompt(&mcp.Prompt{
		Name:        "migrate_to_utf8",
		Title:       "Migrate project to UTF-8",
		Description: "Convert a legacy-encoded tree to UTF-8 with a dry run first.",
		Arguments: []*mcp.PromptArgument{
			{Name: "path", Description: "Project directory", Required: true},
			{Name: "pattern", Description: "File glob to convert (default *.pas)", Required: false},
		},
	}, promptText(func(args map[string]string) string {
		pattern := args["pattern"]
		if pattern == "" {
			pattern = "*.pas"
		}
		return fmt.Sprintf(`Migrate %s files under %s to UTF-8 using the file-tools MCP server.

1. Build the file list with search_files (pattern %q), excluding build output and VCS dirs.
2. convert_encoding with paths=<the list>, to="utf-8", dryRun=true. Nothing is written.
3. Read the report: which files would convert, which are already utf-8, and which would FAIL -
   failures name the exact characters that do not fit, with line and column.
4. Show the user the summary and get an explicit go-ahead.
5. Re-run with dryRun=false and backup=true. Then re-run the dry run: everything should now
   report "already utf-8".
6. Remind the user the .bak files exist and how the compiler/IDE must be told about the new
   encoding (e.g. Delphi needs no change for UTF-8 with BOM, but check bom handling).
Batch calls do not stop at the first failure - every file gets a result entry.`, pattern, args["path"], pattern)
	}))
}

// promptText adapts a string builder to a PromptHandler.
func promptText(build func(map[string]string) string) mcp.PromptHandler {
	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Messages: []*mcp.PromptMessage{{
				Role:    "user",
				Content: &mcp.TextContent{Text: build(req.Params.Arguments)},
			}},
		}, nil
	}
}
