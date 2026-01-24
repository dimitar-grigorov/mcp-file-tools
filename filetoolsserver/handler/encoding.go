package handler

import (
	"context"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleListEncodings returns a list of supported encodings
func (h *Handler) HandleListEncodings(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ListEncodingsInput]) (*mcp.CallToolResultFor[ListEncodingsOutput], error) {
	return &mcp.CallToolResultFor[ListEncodingsOutput]{
		Content: []mcp.Content{&mcp.TextContent{Text: encoding.List()}},
	}, nil
}
