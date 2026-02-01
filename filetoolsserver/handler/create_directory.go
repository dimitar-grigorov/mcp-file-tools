package handler

import (
	"context"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleCreateDirectory creates a new directory or ensures a directory exists.
func (h *Handler) HandleCreateDirectory(ctx context.Context, req *mcp.CallToolRequest, input CreateDirectoryInput) (*mcp.CallToolResult, CreateDirectoryOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, CreateDirectoryOutput{}, nil
	}

	if err := os.MkdirAll(v.Path, 0755); err != nil {
		return errorResult(fmt.Sprintf("failed to create directory: %v", err)), CreateDirectoryOutput{}, nil
	}

	message := fmt.Sprintf("Successfully created directory %s", input.Path)
	return &mcp.CallToolResult{}, CreateDirectoryOutput{Message: message}, nil
}
