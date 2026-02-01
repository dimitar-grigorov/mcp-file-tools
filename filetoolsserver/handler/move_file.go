package handler

import (
	"context"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleMoveFile moves or renames a file or directory.
// Can move files between directories and rename them in a single operation.
// Fails if the destination already exists.
func (h *Handler) HandleMoveFile(ctx context.Context, req *mcp.CallToolRequest, input MoveFileInput) (*mcp.CallToolResult, MoveFileOutput, error) {
	// Validate source path
	if input.Source == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "source is required and must be a non-empty string"}},
			IsError: true,
		}, MoveFileOutput{}, nil
	}

	// Validate destination path
	if input.Destination == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "destination is required and must be a non-empty string"}},
			IsError: true,
		}, MoveFileOutput{}, nil
	}

	// Validate source path against allowed directories
	validatedSource, err := h.validatePath(input.Source)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, MoveFileOutput{}, nil
	}

	// Validate destination path against allowed directories
	validatedDest, err := h.validatePath(input.Destination)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, MoveFileOutput{}, nil
	}

	// Check if source exists
	if _, err := os.Stat(validatedSource); os.IsNotExist(err) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("source does not exist: %s", input.Source)}},
			IsError: true,
		}, MoveFileOutput{}, nil
	}

	// Check if destination already exists
	if _, err := os.Stat(validatedDest); err == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("destination already exists: %s", input.Destination)}},
			IsError: true,
		}, MoveFileOutput{}, nil
	}

	// Perform the move/rename operation
	err = os.Rename(validatedSource, validatedDest)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to move file: %v", err)}},
			IsError: true,
		}, MoveFileOutput{}, nil
	}

	message := fmt.Sprintf("Successfully moved %s to %s", input.Source, input.Destination)
	return &mcp.CallToolResult{}, MoveFileOutput{Message: message}, nil
}
