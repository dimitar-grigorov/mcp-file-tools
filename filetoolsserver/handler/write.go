package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *Handler) HandleWriteFile(ctx context.Context, req *mcp.CallToolRequest, input WriteFileInput) (*mcp.CallToolResult, WriteFileOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, WriteFileOutput{}, nil
	}

	encodingName := h.config.DefaultEncoding
	if input.Encoding != "" {
		encodingName = strings.ToLower(input.Encoding)
	}

	enc, ok := encoding.Get(encodingName)
	if !ok {
		return errorResult(fmt.Sprintf("unsupported encoding: %s. Use list_encodings to see available encodings.", encodingName)), WriteFileOutput{}, nil
	}

	var contentToWrite []byte
	if encoding.IsUTF8(encodingName) {
		contentToWrite = []byte(input.Content)
	} else {
		encoder := enc.NewEncoder()
		encoded, err := encoder.Bytes([]byte(input.Content))
		if err != nil {
			return errorResult(fmt.Sprintf("failed to encode content: %v", err)), WriteFileOutput{}, nil
		}
		contentToWrite = encoded
	}

	mode := getFileMode(v.Path)
	if err := atomicWriteFile(v.Path, contentToWrite, mode); err != nil {
		return errorResult(fmt.Sprintf("failed to write file: %v", err)), WriteFileOutput{}, nil
	}

	message := fmt.Sprintf("Successfully wrote %d bytes to %s (encoding: %s)", len(contentToWrite), input.Path, encodingName)
	return &mcp.CallToolResult{}, WriteFileOutput{Message: message}, nil
}
