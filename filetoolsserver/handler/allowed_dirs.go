package handler

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleListAllowedDirectories lists all directories accessible to this server
func (h *Handler) HandleListAllowedDirectories(ctx context.Context, req *mcp.CallToolRequest, input ListAllowedDirectoriesInput) (*mcp.CallToolResult, ListAllowedDirectoriesOutput, error) {
	dirs := h.GetAllowedDirectories()
	return &mcp.CallToolResult{}, ListAllowedDirectoriesOutput{Directories: dirs}, nil
}
