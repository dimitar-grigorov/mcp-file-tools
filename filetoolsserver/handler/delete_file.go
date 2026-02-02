package handler

import (
	"context"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleDeleteFile deletes a file.
func (h *Handler) HandleDeleteFile(ctx context.Context, req *mcp.CallToolRequest, input DeleteFileInput) (*mcp.CallToolResult, DeleteFileOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, DeleteFileOutput{}, nil
	}

	info, err := os.Stat(v.Path)
	if os.IsNotExist(err) {
		return errorResult(fmt.Sprintf("file does not exist: %s", input.Path)), DeleteFileOutput{}, nil
	}
	if err != nil {
		return errorResult(fmt.Sprintf("failed to access file: %v", err)), DeleteFileOutput{}, nil
	}

	if info.IsDir() {
		return errorResult("path is a directory, use a different tool to delete directories"), DeleteFileOutput{}, nil
	}

	if err := os.Remove(v.Path); err != nil {
		return errorResult(fmt.Sprintf("failed to delete file: %v", err)), DeleteFileOutput{}, nil
	}

	message := fmt.Sprintf("Successfully deleted %s", input.Path)
	return &mcp.CallToolResult{}, DeleteFileOutput{Message: message}, nil
}
