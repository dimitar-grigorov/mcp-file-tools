package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleListDirectory lists files in a directory with optional pattern filtering
func (h *Handler) HandleListDirectory(ctx context.Context, req *mcp.CallToolRequest, input ListDirectoryInput) (*mcp.CallToolResult, ListDirectoryOutput, error) {
	// Validate path
	if input.Path == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "path is required and must be a non-empty string"}},
			IsError: true,
		}, ListDirectoryOutput{}, nil
	}

	// Validate path against allowed directories
	validatedPath, err := h.validatePath(input.Path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, ListDirectoryOutput{}, nil
	}

	// Default pattern
	pattern := "*"
	if input.Pattern != "" {
		pattern = input.Pattern
	}

	// Read directory
	entries, err := os.ReadDir(validatedPath)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to read directory: %v", err)}},
			IsError: true,
		}, ListDirectoryOutput{}, nil
	}

	// Filter by pattern and build result
	var files []string
	for _, entry := range entries {
		matched, err := filepath.Match(pattern, entry.Name())
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("invalid pattern: %v", err)}},
				IsError: true,
			}, ListDirectoryOutput{}, nil
		}
		if matched {
			prefix := ""
			if entry.IsDir() {
				prefix = "[DIR] "
			}
			files = append(files, prefix+entry.Name())
		}
	}

	return &mcp.CallToolResult{}, ListDirectoryOutput{Files: files}, nil
}
