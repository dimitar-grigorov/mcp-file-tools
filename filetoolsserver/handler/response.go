// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import "github.com/modelcontextprotocol/go-sdk/mcp"

// errorResult creates an error CallToolResult with the given message.
func errorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}

// gitignoreDefault resolves the respectGitignore parameter: nil means true.
func gitignoreDefault(v *bool) bool {
	return v == nil || *v
}
