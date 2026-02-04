package handler

import (
	"context"
	"fmt"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *Handler) HandleWriteFile(ctx context.Context, req *mcp.CallToolRequest, input WriteFileInput) (*mcp.CallToolResult, WriteFileOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, WriteFileOutput{}, nil
	}

	// Resolve encoding: explicit > preserve existing > configured default
	encodingName, err := h.resolveWriteEncoding(input.Encoding, v.Path)
	if err != nil {
		return errorResult(err.Error()), WriteFileOutput{}, nil
	}

	enc, _ := encoding.Get(encodingName) // Already validated by resolveWriteEncoding

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
