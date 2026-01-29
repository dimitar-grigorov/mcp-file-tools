package handler

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleListAllowedDirectories lists all directories accessible to the server
func (h *Handler) HandleListAllowedDirectories(
	ctx context.Context,
	ss *mcp.ServerSession,
	params *mcp.CallToolParamsFor[ListAllowedDirectoriesInput],
) (*mcp.CallToolResultFor[ListAllowedDirectoriesOutput], error) {

	dirs := h.GetAllowedDirectories()

	if len(dirs) == 0 {
		return &mcp.CallToolResultFor[ListAllowedDirectoriesOutput]{
			Content: []mcp.Content{&mcp.TextContent{
				Text: "No allowed directories configured",
			}},
		}, nil
	}

	text := "Allowed directories:\n"
	for _, dir := range dirs {
		text += fmt.Sprintf("- %s\n", dir)
	}

	return &mcp.CallToolResultFor[ListAllowedDirectoriesOutput]{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil
}
