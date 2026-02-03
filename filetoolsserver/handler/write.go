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
	// Validate path
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, WriteFileOutput{}, nil
	}
	validatedPath := v.Path

	// Default encoding from config
	encodingName := h.config.DefaultEncoding
	if input.Encoding != "" {
		encodingName = strings.ToLower(input.Encoding)
	}

	// Validate encoding
	enc, ok := encoding.Get(encodingName)
	if !ok {
		return errorResult(fmt.Sprintf("unsupported encoding: %s. Use list_encodings to see available encodings.", encodingName)), WriteFileOutput{}, nil
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
			return errorResult(fmt.Sprintf("failed to encode content: %v", err)), WriteFileOutput{}, nil
		}
		contentToWrite = encoded
	}

	// Preserve original permissions if file exists, otherwise use default
	mode := getFileMode(validatedPath)

	// Write file
	if err := os.WriteFile(validatedPath, contentToWrite, mode); err != nil {
		return errorResult(fmt.Sprintf("failed to write file: %v", err)), WriteFileOutput{}, nil
	}

	message := fmt.Sprintf("Successfully wrote %d bytes to %s (encoding: %s)", len(contentToWrite), input.Path, encodingName)
	return &mcp.CallToolResult{}, WriteFileOutput{Message: message}, nil
}
