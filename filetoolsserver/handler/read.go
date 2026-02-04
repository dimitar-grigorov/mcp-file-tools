package handler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	textEncoding "golang.org/x/text/encoding"
)

// encodingResult holds the result of encoding resolution
type encodingResult struct {
	encoder            textEncoding.Encoding
	name               string
	detectedEncoding   string
	encodingConfidence int
	autoDetected       bool
}

func (h *Handler) HandleReadTextFile(ctx context.Context, req *mcp.CallToolRequest, input ReadTextFileInput) (*mcp.CallToolResult, ReadTextFileOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, ReadTextFileOutput{}, nil
	}

	if loadToMemory, size := h.shouldLoadEntireFile(v.Path); !loadToMemory {
		slog.Warn("loading large file into memory", "path", input.Path, "size", size, "threshold", h.config.MemoryThreshold)
	}

	encResult, err := h.resolveEncoding(input.Encoding, v.Path)
	if err != nil {
		return errorResult(err.Error()), ReadTextFileOutput{}, nil
	}

	data, err := os.ReadFile(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read file: %v", err)), ReadTextFileOutput{}, nil
	}

	content, err := decodeContent(data, encResult)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to decode file content: %v", err)), ReadTextFileOutput{}, nil
	}

	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	var startLine, endLine int
	if input.Offset != nil || input.Limit != nil {
		content, startLine, endLine = applyOffsetLimit(lines, input.Offset, input.Limit)
	} else {
		content = strings.Join(lines, "\n")
		startLine = 1
		endLine = totalLines
	}

	output := ReadTextFileOutput{
		Content:    content,
		TotalLines: totalLines,
		StartLine:  startLine,
		EndLine:    endLine,
	}
	if encResult.autoDetected {
		output.DetectedEncoding = encResult.detectedEncoding
		output.EncodingConfidence = encResult.encodingConfidence
	}

	return &mcp.CallToolResult{}, output, nil
}

// resolveEncoding returns explicit encoding or auto-detects based on file size.
func (h *Handler) resolveEncoding(inputEncoding string, filePath string) (encodingResult, error) {
	result := encodingResult{}

	if inputEncoding != "" {
		// Use explicitly specified encoding
		result.name = strings.ToLower(inputEncoding)
		enc, ok := encoding.Get(result.name)
		if !ok {
			return result, fmt.Errorf("%w: %s. Use list_encodings to see available encodings", ErrEncodingUnsupported, result.name)
		}
		result.encoder = enc
		return result, nil
	}

	// Determine detection mode based on file size
	detectionMode := "full"
	if loadToMemory, _ := h.shouldLoadEntireFile(filePath); !loadToMemory {
		detectionMode = "sample"
	}

	// Auto-detect encoding
	result.autoDetected = true
	detection, err := encoding.DetectFromFile(filePath, detectionMode)
	if err != nil {
		// Detection failed, fall back to UTF-8
		result.name = "utf-8"
		result.detectedEncoding = "detection failed, using utf-8"
		result.encoder = nil
		return result, nil
	}
	result.detectedEncoding = detection.Charset
	result.encodingConfidence = detection.Confidence

	trusted := detection.Confidence >= encoding.MinConfidenceThreshold
	if trusted && detection.Charset != "" {
		result.name = detection.Charset
	} else {
		// Fall back to UTF-8 if detection is not confident enough
		result.name = "utf-8"
		if detection.Charset != "" {
			result.detectedEncoding = detection.Charset + " (low confidence, using utf-8)"
		}
	}

	// Validate the detected/fallback encoding
	enc, ok := encoding.Get(result.name)
	if !ok {
		// Unsupported detected encoding, fall back to UTF-8
		result.encoder = nil
		result.name = "utf-8"
		result.detectedEncoding = result.detectedEncoding + " (unsupported, using utf-8)"
	} else {
		result.encoder = enc
	}

	return result, nil
}

// decodeContent decodes the file data to UTF-8 using the resolved encoding
func decodeContent(data []byte, encResult encodingResult) (string, error) {
	if encoding.IsUTF8(encResult.name) {
		return string(data), nil
	}

	decoder := encResult.encoder.NewDecoder()
	utf8Content, err := decoder.Bytes(data)
	if err != nil {
		return "", err
	}
	return string(utf8Content), nil
}

// applyOffsetLimit applies offset and limit to select a range of lines.
// Offset is 1-indexed (like line numbers). Returns content, startLine, endLine.
// Negative values are treated as not provided.
func applyOffsetLimit(lines []string, offset, limit *int) (string, int, int) {
	totalLines := len(lines)

	// Default offset is 1 (first line)
	startIdx := 0
	if offset != nil && *offset > 1 {
		startIdx = *offset - 1 // Convert 1-indexed to 0-indexed
		if startIdx >= totalLines {
			return "", totalLines + 1, totalLines // Empty result, past end
		}
	}

	// Default limit is all remaining lines
	endIdx := totalLines
	if limit != nil && *limit > 0 {
		endIdx = startIdx + *limit
		if endIdx > totalLines {
			endIdx = totalLines
		}
	}

	selectedLines := lines[startIdx:endIdx]
	return strings.Join(selectedLines, "\n"), startIdx + 1, endIdx
}
