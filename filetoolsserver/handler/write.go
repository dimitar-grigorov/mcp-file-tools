package handler

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleWriteFile writes UTF-8 content to a file with the specified encoding
func (h *Handler) HandleWriteFile(ctx context.Context, req *mcp.CallToolRequest, input WriteFileInput) (*mcp.CallToolResult, WriteFileOutput, error) {
	// Validate inputs
	if input.Path == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "path is required and must be a non-empty string"}},
			IsError: true,
		}, WriteFileOutput{}, nil
	}

	// Validate path against allowed directories
	validatedPath, err := h.validatePath(input.Path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, WriteFileOutput{}, nil
	}

	// Default encoding
	encodingName := h.defaultEncoding
	if input.Encoding != "" {
		encodingName = strings.ToLower(input.Encoding)
	}

	// Validate encoding
	enc, ok := encoding.Get(encodingName)
	if !ok {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("unsupported encoding: %s. Use list_encodings to see available encodings.", encodingName),
			}},
			IsError: true,
		}, WriteFileOutput{}, nil
	}

	var contentToWrite []byte

	// UTF-8: write content as-is (no conversion needed)
	if encoding.IsUTF8(encodingName) {
		contentToWrite = []byte(input.Content)
	} else {
		// Encode UTF-8 content to specified encoding
		encoder := enc.NewEncoder()
		encoded, err := encoder.Bytes([]byte(input.Content))
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to encode content: %v", err)}},
				IsError: true,
			}, WriteFileOutput{}, nil
		}
		contentToWrite = encoded
	}

	// Write file
	if err := os.WriteFile(validatedPath, contentToWrite, 0644); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to write file: %v", err)}},
			IsError: true,
		}, WriteFileOutput{}, nil
	}

	message := fmt.Sprintf("Successfully wrote %d bytes to %s (encoding: %s)", len(contentToWrite), input.Path, encodingName)
	return &mcp.CallToolResult{}, WriteFileOutput{Message: message}, nil
}
