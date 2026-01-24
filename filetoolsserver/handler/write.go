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
func (h *Handler) HandleWriteFile(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[WriteFileInput]) (*mcp.CallToolResultFor[WriteFileOutput], error) {
	input := params.Arguments

	// Validate inputs
	if input.Path == "" {
		return &mcp.CallToolResultFor[WriteFileOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: "path is required and must be a non-empty string"}},
			IsError: true,
		}, nil
	}

	// Default encoding
	encodingName := h.defaultEncoding
	if input.Encoding != "" {
		encodingName = strings.ToLower(input.Encoding)
	}

	// Validate encoding
	enc, ok := encoding.Get(encodingName)
	if !ok {
		return &mcp.CallToolResultFor[WriteFileOutput]{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("unsupported encoding: %s. Use list_encodings to see available encodings.", encodingName),
			}},
			IsError: true,
		}, nil
	}

	// Check if existing file is UTF-8 but we're writing non-UTF-8
	var warning string
	if !encoding.IsUTF8(encodingName) {
		if existingData, err := os.ReadFile(input.Path); err == nil {
			if encoding.IsValidUTF8(existingData) {
				warning = fmt.Sprintf("\n\nWarning: Existing file was UTF-8, but writing with %s encoding. This may cause encoding issues.", encodingName)
			}
		}
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
			return &mcp.CallToolResultFor[WriteFileOutput]{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to encode content: %v", err)}},
				IsError: true,
			}, nil
		}
		contentToWrite = encoded
	}

	// Write file
	if err := os.WriteFile(input.Path, contentToWrite, 0644); err != nil {
		return &mcp.CallToolResultFor[WriteFileOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to write file: %v", err)}},
			IsError: true,
		}, nil
	}

	message := fmt.Sprintf("Successfully wrote %d bytes to %s (encoding: %s)%s", len(contentToWrite), input.Path, encodingName, warning)
	return &mcp.CallToolResultFor[WriteFileOutput]{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
	}, nil
}
