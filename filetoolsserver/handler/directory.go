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
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, ListDirectoryOutput{}, nil
	}

	pattern := "*"
	if input.Pattern != "" {
		pattern = input.Pattern
	}

	entries, err := os.ReadDir(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read directory: %v", err)), ListDirectoryOutput{}, nil
	}

	var files []string
	for _, entry := range entries {
		matched, err := filepath.Match(pattern, entry.Name())
		if err != nil {
			return errorResult(fmt.Sprintf("invalid pattern: %v", err)), ListDirectoryOutput{}, nil
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
