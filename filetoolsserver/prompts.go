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
		return fmt.Sprintf(`Audit text encodings under %s (file-tools MCP, read-only).

1. tree with showEncoding=true.
2. Spot-check low-confidence files with detect_encoding mode="chunked".
3. Report a table: encoding -> count -> example files. Flag mixed directories,
   UTF-8 BOMs on scripts (break PHP/shell), and unreadable files.`, args["path"])
	}))

	server.AddPrompt(&mcp.Prompt{
		Name:        "fix_mojibake",
		Title:       "Fix garbled text",
		Description: "Diagnose and repair a file showing � characters or garbled Cyrillic/accented text.",
		Arguments: []*mcp.PromptArgument{
			{Name: "path", Description: "File that displays wrong", Required: true},
		},
	}, promptText(func(args map[string]string) string {
		return fmt.Sprintf(`%s displays garbled text. Diagnose and fix with the file-tools MCP.

1. detect_encoding mode="chunked".
2. read_text_file with the detected encoding. Reads fine? Then the *consumer*
   decodes it wrongly - report that and stop.
3. Still garbled? Try cp1251, cp1252, koi8-r until one renders real words.
4. Confirm the correct reading with the user BEFORE writing anything.
5. convert_encoding backup=true, read back to verify.
Never convert on a guess - worse than the mojibake.`, args["path"])
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
		return fmt.Sprintf(`Migrate %s files under %s to UTF-8 (file-tools MCP).

1. search_files with pattern %q to build the list.
2. convert_encoding paths=<list>, to="utf-8", dryRun=true - writes nothing.
3. Report: what converts, what is already utf-8, what would FAIL (failures name
   the characters that do not fit, with line/column).
4. Get an explicit go-ahead from the user.
5. Re-run with dryRun=false, backup=true. Verify with one more dry run: all
   should report "already utf-8". Remind the user about the .bak files.`, pattern, args["path"], pattern)
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
