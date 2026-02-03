package handler

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleListAllowedDirectories lists all directories accessible to this server
func (h *Handler) HandleListAllowedDirectories(ctx context.Context, req *mcp.CallToolRequest, input ListAllowedDirectoriesInput) (*mcp.CallToolResult, ListAllowedDirectoriesOutput, error) {
	dirs := h.GetAllowedDirectories()
	output := ListAllowedDirectoriesOutput{Directories: dirs}

	if len(dirs) == 0 {
		output.Message = "No allowed directories configured. File operations will fail. " +
			"Add directory paths as args in .mcp.json (project) or ~/.claude.json (global). " +
			"Example: {\"mcpServers\": {\"file-tools\": {\"type\": \"stdio\", \"command\": \"/path/to/mcp-file-tools\", \"args\": [\"D:\\\\Projects\"]}}}"
	}

	return &mcp.CallToolResult{}, output, nil
}
