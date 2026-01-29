package handler

import (
	"context"
	"fmt"
	"os"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleDetectEncoding detects the encoding of a file
func (h *Handler) HandleDetectEncoding(ctx context.Context, req *mcp.CallToolRequest, input DetectEncodingInput) (*mcp.CallToolResult, DetectEncodingOutput, error) {
	// Validate path
	if input.Path == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "path is required and must be a non-empty string"}},
			IsError: true,
		}, DetectEncodingOutput{}, nil
	}

	// Validate path against allowed directories
	validatedPath, err := h.validatePath(input.Path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, DetectEncodingOutput{}, nil
	}

	// Read file
	data, err := os.ReadFile(validatedPath)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to read file: %v", err)}},
			IsError: true,
		}, DetectEncodingOutput{}, nil
	}

	// Detect encoding
	result := encoding.Detect(data)

	// Handle unknown encoding
	if result.Charset == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "Could not detect encoding"}},
			IsError: true,
		}, DetectEncodingOutput{}, nil
	}

	return &mcp.CallToolResult{}, DetectEncodingOutput{
		Encoding:   result.Charset,
		Confidence: result.Confidence,
		HasBOM:     result.HasBOM,
	}, nil
}
