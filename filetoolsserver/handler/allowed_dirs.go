// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleListAllowedDirectories lists all directories accessible to this server
func (h *Handler) HandleListAllowedDirectories(ctx context.Context, req *mcp.CallToolRequest, input ListAllowedDirectoriesInput) (*mcp.CallToolResult, ListAllowedDirectoriesOutput, error) {
	dirs := h.GetAllowedDirectories()
	output := ListAllowedDirectoriesOutput{Directories: dirs}

	slog.Debug("list_allowed_directories response", "count", len(dirs), "dirs", dirs)

	if len(dirs) == 0 {
		output.Message = "No allowed directories - file operations will fail. The working directory was " +
			"refused (drive root or home) or the fallback is off; the server's stderr says which. " +
			"Add paths as args in .mcp.json, or set MCP_FILE_TOOLS_ALLOWED_DIRS in its env block. " +
			"Example: {\"mcpServers\": {\"file-tools\": {\"type\": \"stdio\", \"command\": \"/path/to/mcp-file-tools\", \"args\": [\"D:\\\\Projects\"]}}}"
	}

	return &mcp.CallToolResult{}, output, nil
}
