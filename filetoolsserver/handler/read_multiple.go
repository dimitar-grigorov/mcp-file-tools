package handler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleReadMultipleFiles reads multiple files concurrently.
// Individual file failures don't stop the operation - errors are reported per file.
func (h *Handler) HandleReadMultipleFiles(ctx context.Context, req *mcp.CallToolRequest, input ReadMultipleFilesInput) (*mcp.CallToolResult, ReadMultipleFilesOutput, error) {
	if len(input.Paths) == 0 {
		return errorResult("paths array is required and must contain at least one path"), ReadMultipleFilesOutput{}, nil
	}
	results := make([]FileReadResult, len(input.Paths))
	var wg sync.WaitGroup
	for i, path := range input.Paths {
		wg.Add(1)
		go func(idx int, filePath string) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				results[idx] = FileReadResult{
					Path:      filePath,
					Error:     "operation cancelled",
					ErrorCode: ErrCodeOperationFailed,
				}
			default:
				results[idx] = h.readSingleFile(filePath, input.Encoding)
			}
		}(i, path)
	}
	wg.Wait()

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
		result.Error = v.Err.Error()
		result.ErrorCode = classifyPathError(v.Err)
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
		result.Error, result.ErrorCode = classifyReadError(err, v.Path)
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

// classifyPathError returns an error code based on the path validation error type.
func classifyPathError(err error) string {
	switch {
	case errors.Is(err, ErrPathRequired):
		return ErrCodeInvalidPath
	case errors.Is(err, security.ErrPathDenied):
		return ErrCodeAccessDenied
	case errors.Is(err, security.ErrSymlinkDenied):
		return ErrCodeSymlinkEscape
	case errors.Is(err, security.ErrNoAllowedDirs):
		return ErrCodeAccessDenied
	case errors.Is(err, security.ErrParentDirDenied):
		return ErrCodeAccessDenied
	default:
		return ErrCodeInvalidPath
	}
}

// classifyReadError returns a descriptive error message and code for file read errors.
func classifyReadError(err error, path string) (string, string) {
	if os.IsNotExist(err) {
		return fmt.Sprintf("file not found: %s", path), ErrCodeNotFound
	}
	if os.IsPermission(err) {
		return fmt.Sprintf("permission denied: %s", path), ErrCodePermission
	}
	return fmt.Sprintf("failed to read file: %v", err), ErrCodeIO
}
