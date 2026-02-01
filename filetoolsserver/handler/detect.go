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
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, DetectEncodingOutput{}, nil
	}

	data, err := os.ReadFile(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read file: %v", err)), DetectEncodingOutput{}, nil
	}

	result := encoding.Detect(data)
	if result.Charset == "" {
		return errorResult("Could not detect encoding"), DetectEncodingOutput{}, nil
	}

	return &mcp.CallToolResult{}, DetectEncodingOutput{
		Encoding:   result.Charset,
		Confidence: result.Confidence,
		HasBOM:     result.HasBOM,
	}, nil
}
