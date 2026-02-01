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
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, GetFileInfoOutput{}, nil
	}

	stat, err := os.Stat(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get file info: %v", err)), GetFileInfoOutput{}, nil
	}

	created, accessed, modified := getFileTimes(stat)
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

func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}
