package handler

import (
	"context"
	"fmt"
	"os"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/workpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleReadMultipleFiles reads multiple files concurrently.
// Individual file failures don't stop the operation - errors are reported per file.
func (h *Handler) HandleReadMultipleFiles(ctx context.Context, req *mcp.CallToolRequest, input ReadMultipleFilesInput) (*mcp.CallToolResult, ReadMultipleFilesOutput, error) {
	if len(input.Paths) == 0 {
		return errorResult("paths array is required and must contain at least one path"), ReadMultipleFilesOutput{}, nil
	}
	results := make([]FileReadResult, len(input.Paths))

	// Every path gets a result even under cancellation, hence DispatchAfterCancel.
	workpool.RunOrdered(ctx, input.Paths, workpool.Options{DispatchAfterCancel: true},
		func(ctx context.Context, _ int, path string) FileReadResult {
			if ctx.Err() != nil {
				return FileReadResult{
					Path:      path,
					Error:     "operation cancelled",
					ErrorCode: ErrCodeOperationFailed,
				}
			}
			return h.readSingleFile(path, input.Encoding)
		},
		func(idx int, result FileReadResult) bool {
			results[idx] = result
			return true
		})

	var successCount, errorCount int
	var errorSummary []string
	for _, r := range results {
		if r.Error != "" {
			errorCount++
			errorSummary = append(errorSummary, fmt.Sprintf("%s: %s", r.Path, r.Error))
		} else {
			successCount++
		}
	}

	return &mcp.CallToolResult{}, ReadMultipleFilesOutput{
		Results:      results,
		SuccessCount: successCount,
		ErrorCount:   errorCount,
		Errors:       errorSummary,
	}, nil
}

// readSingleFile reads a single file with optional encoding.
func (h *Handler) readSingleFile(path, requestedEncoding string) FileReadResult {
	result := FileReadResult{Path: path}

	v := h.ValidatePath(path)
	if !v.Ok() {
		// No path here: a validation error's own message is the useful one.
		result.Error, result.ErrorCode = mapOperationError(v.Err, "", ErrCodeInvalidPath)
		return result
	}

	// Resolve encoding (detection mode based on file size vs MemoryThreshold)
	encResult, err := h.resolveEncoding(requestedEncoding, v.Path)
	if err != nil {
		result.Error = err.Error()
		result.ErrorCode = ErrCodeEncoding
		return result
	}

	// Read file content for decoding
	data, err := os.ReadFile(v.Path)
	if err != nil {
		result.Error, result.ErrorCode = mapOperationError(
			fmt.Errorf("failed to read file: %w", err), v.Path, ErrCodeIO)
		return result
	}

	content, err := decodeContent(data, encResult)
	if err != nil {
		result.Error = fmt.Sprintf("failed to decode file content: %v", err)
		result.ErrorCode = ErrCodeEncoding
		return result
	}

	result.Content = content
	if encResult.autoDetected {
		result.DetectedEncoding = encResult.detectedEncoding
		result.EncodingConfidence = encResult.encodingConfidence
	}

	return result
}
