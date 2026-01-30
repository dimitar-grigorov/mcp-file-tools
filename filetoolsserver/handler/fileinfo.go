package handler

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleGetFileInfo retrieves detailed metadata about a file or directory
func (h *Handler) HandleGetFileInfo(ctx context.Context, req *mcp.CallToolRequest, input GetFileInfoInput) (*mcp.CallToolResult, GetFileInfoOutput, error) {
	// Validate path
	if input.Path == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "path is required and must be a non-empty string"}},
			IsError: true,
		}, GetFileInfoOutput{}, nil
	}

	// Validate path against allowed directories
	validatedPath, err := h.validatePath(input.Path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, GetFileInfoOutput{}, nil
	}

	// Get file info
	stat, err := os.Stat(validatedPath)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to get file info: %v", err)}},
			IsError: true,
		}, GetFileInfoOutput{}, nil
	}

	// Get file times (platform-specific)
	created, accessed, modified := getFileTimes(stat)

	// Get permissions in octal format (last 3 digits)
	permissions := fmt.Sprintf("%03o", stat.Mode().Perm())

	output := GetFileInfoOutput{
		Size:        stat.Size(),
		Created:     formatTime(created),
		Modified:    formatTime(modified),
		Accessed:    formatTime(accessed),
		IsDirectory: stat.IsDir(),
		IsFile:      stat.Mode().IsRegular(),
		Permissions: permissions,
	}

	return &mcp.CallToolResult{}, output, nil
}

// formatTime formats a time.Time to ISO 8601 format
func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}
