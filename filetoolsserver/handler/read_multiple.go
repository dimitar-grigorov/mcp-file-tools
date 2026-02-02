package handler

import (
	"context"
	"fmt"
	"os"
	"sync"

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
			results[idx] = h.readSingleFile(filePath, input.Encoding)
		}(i, path)
	}

	wg.Wait()

	var successCount, errorCount int
	for _, r := range results {
		if r.Error != "" {
			errorCount++
		} else {
			successCount++
		}
	}

	return &mcp.CallToolResult{}, ReadMultipleFilesOutput{
		Results:      results,
		SuccessCount: successCount,
		ErrorCount:   errorCount,
	}, nil
}

// readSingleFile reads a single file with optional encoding.
func (h *Handler) readSingleFile(path, requestedEncoding string) FileReadResult {
	result := FileReadResult{Path: path}

	v := h.ValidatePath(path)
	if !v.Ok() {
		result.Error = v.Err.Error()
		return result
	}

	data, err := os.ReadFile(v.Path)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read file: %v", err)
		return result
	}

	encResult, err := resolveEncoding(requestedEncoding, data)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	content, err := decodeContent(data, encResult)
	if err != nil {
		result.Error = fmt.Sprintf("failed to decode file content: %v", err)
		return result
	}

	result.Content = content
	if encResult.autoDetected {
		result.DetectedEncoding = encResult.detectedEncoding
		result.EncodingConfidence = encResult.encodingConfidence
	}

	return result
}
