package handler

import (
	"context"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleListEncodings returns a list of supported encodings
func (h *Handler) HandleListEncodings(ctx context.Context, req *mcp.CallToolRequest, input ListEncodingsInput) (*mcp.CallToolResult, ListEncodingsOutput, error) {
	items := encoding.ListEncodings()

	// Convert to handler types
	encodings := make([]EncodingItem, len(items))
	for i, item := range items {
		encodings[i] = EncodingItem{
			Name:        item.Name,
			DisplayName: item.DisplayName,
			Aliases:     item.Aliases,
			Description: item.Description,
		}
	}

	return &mcp.CallToolResult{}, ListEncodingsOutput{Encodings: encodings}, nil
}
