package handler

import "github.com/modelcontextprotocol/go-sdk/mcp"

// Response helpers for handler operations.
// These provide consistent error and success response formatting across all handlers.

// errorResult creates an error CallToolResult with the given message.
// Use this for all error responses to maintain consistency.
func errorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: true,
	}
}
