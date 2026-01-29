package handler

import (
	"context"
	"fmt"
	"os"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleDetectEncoding detects the encoding of a file
func (h *Handler) HandleDetectEncoding(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[DetectEncodingInput]) (*mcp.CallToolResultFor[DetectEncodingOutput], error) {
	input := params.Arguments

	// Validate path
	if input.Path == "" {
		return &mcp.CallToolResultFor[DetectEncodingOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: "path is required and must be a non-empty string"}},
			IsError: true,
		}, nil
	}

	// Validate path against allowed directories
	validatedPath, err := h.validatePath(input.Path)
	if err != nil {
		return &mcp.CallToolResultFor[DetectEncodingOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, nil
	}

	// Read file
	data, err := os.ReadFile(validatedPath)
	if err != nil {
		return &mcp.CallToolResultFor[DetectEncodingOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to read file: %v", err)}},
			IsError: true,
		}, nil
	}

	// Detect encoding
	result := encoding.Detect(data)

	// Handle unknown encoding
	if result.Charset == "" {
		return &mcp.CallToolResultFor[DetectEncodingOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: "Could not detect encoding"}},
			IsError: true,
		}, nil
	}

	// Build response message
	message := fmt.Sprintf("Detected encoding: %s (confidence: %d%%)", result.Charset, result.Confidence)
	if result.HasBOM {
		message += " [has BOM]"
	}

	return &mcp.CallToolResultFor[DetectEncodingOutput]{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
	}, nil
}
