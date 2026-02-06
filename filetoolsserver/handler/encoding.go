package handler

import (
	"context"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleListEncodings returns a list of supported encodings
func (h *Handler) HandleListEncodings(ctx context.Context, req *mcp.CallToolRequest, input ListEncodingsInput) (*mcp.CallToolResult, ListEncodingsOutput, error) {
	return &mcp.CallToolResult{}, ListEncodingsOutput{Encodings: encoding.ListEncodings()}, nil
}
