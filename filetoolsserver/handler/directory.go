package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleListDirectory lists files in a directory with optional pattern filtering
func (h *Handler) HandleListDirectory(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ListDirectoryInput]) (*mcp.CallToolResultFor[ListDirectoryOutput], error) {
	input := params.Arguments

	// Validate path
	if input.Path == "" {
		return &mcp.CallToolResultFor[ListDirectoryOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: "path is required and must be a non-empty string"}},
			IsError: true,
		}, nil
	}

	// Validate path against allowed directories
	validatedPath, err := h.validatePath(input.Path)
	if err != nil {
		return &mcp.CallToolResultFor[ListDirectoryOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, nil
	}

	// Default pattern
	pattern := "*"
	if input.Pattern != "" {
		pattern = input.Pattern
	}

	// Read directory
	entries, err := os.ReadDir(validatedPath)
	if err != nil {
		return &mcp.CallToolResultFor[ListDirectoryOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to read directory: %v", err)}},
			IsError: true,
		}, nil
	}

	// Filter by pattern and build result
	var files []string
	for _, entry := range entries {
		matched, err := filepath.Match(pattern, entry.Name())
		if err != nil {
			return &mcp.CallToolResultFor[ListDirectoryOutput]{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("invalid pattern: %v", err)}},
				IsError: true,
			}, nil
		}
		if matched {
			prefix := ""
			if entry.IsDir() {
				prefix = "[DIR] "
			}
			files = append(files, prefix+entry.Name())
		}
	}

	// Return result
	if len(files) == 0 {
		return &mcp.CallToolResultFor[ListDirectoryOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: "No files found matching the pattern."}},
		}, nil
	}

	return &mcp.CallToolResultFor[ListDirectoryOutput]{
		Content: []mcp.Content{&mcp.TextContent{Text: strings.Join(files, "\n")}},
	}, nil
}
