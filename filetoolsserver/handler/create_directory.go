package handler

import (
	"context"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleCreateDirectory creates a new directory or ensures a directory exists.
// Can create multiple nested directories in one operation (like mkdir -p).
func (h *Handler) HandleCreateDirectory(ctx context.Context, req *mcp.CallToolRequest, input CreateDirectoryInput) (*mcp.CallToolResult, CreateDirectoryOutput, error) {
	// Validate path
	if input.Path == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "path is required and must be a non-empty string"}},
			IsError: true,
		}, CreateDirectoryOutput{}, nil
	}

	// Validate path against allowed directories
	validatedPath, err := h.validatePath(input.Path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, CreateDirectoryOutput{}, nil
	}

	// Create directory (and any necessary parents)
	err = os.MkdirAll(validatedPath, 0755)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to create directory: %v", err)}},
			IsError: true,
		}, CreateDirectoryOutput{}, nil
	}

	message := fmt.Sprintf("Successfully created directory %s", input.Path)
	return &mcp.CallToolResult{}, CreateDirectoryOutput{Message: message}, nil
}
