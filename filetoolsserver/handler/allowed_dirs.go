package handler

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleListAllowedDirectories lists all directories accessible to this server
func (h *Handler) HandleListAllowedDirectories(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ListAllowedDirectoriesInput]) (*mcp.CallToolResultFor[ListAllowedDirectoriesOutput], error) {
	dirs := h.GetAllowedDirectories()

	var text string
	if len(dirs) == 0 {
		text = "No allowed directories configured.\nDirectories will be provided via MCP roots protocol or CLI arguments."
	} else {
		text = "Allowed directories:\n" + strings.Join(dirs, "\n")
	}

	return &mcp.CallToolResultFor[ListAllowedDirectoriesOutput]{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil
}
