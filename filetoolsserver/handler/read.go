package handler

import (
	"context"
	"fmt"
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

// HandleReadTextFile reads a file in the specified encoding and returns UTF-8 content.
// If encoding is not specified, it auto-detects the encoding using chunked sampling.
// Supports offset/limit for reading specific line ranges (context-efficient).
func (h *Handler) HandleReadTextFile(ctx context.Context, req *mcp.CallToolRequest, input ReadTextFileInput) (*mcp.CallToolResult, ReadTextFileOutput, error) {
	// Validate conflicting parameters
	if input.Head != nil && input.Tail != nil {
		return errorResult(ErrHeadTailConflict.Error()), ReadTextFileOutput{}, nil
	}
	if (input.Offset != nil || input.Limit != nil) && (input.Head != nil || input.Tail != nil) {
		return errorResult("cannot use offset/limit with head/tail"), ReadTextFileOutput{}, nil
	}

	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, ReadTextFileOutput{}, nil
	}

	// Resolve encoding using streaming detection (only reads ~384KB max for detection)
	encResult, err := resolveEncoding(input.Encoding, v.Path)
	if err != nil {
		return errorResult(err.Error()), ReadTextFileOutput{}, nil
	}

	// Read file content for decoding
	// TODO: This still loads the entire file - optimize with streaming in future
	data, err := os.ReadFile(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read file: %v", err)), ReadTextFileOutput{}, nil
	}

	// Decode content to UTF-8
	content, err := decodeContent(data, encResult)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to decode file content: %v", err)), ReadTextFileOutput{}, nil
	}

	// Count total lines
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// Apply line selection (offset/limit or head/tail)
	var startLine, endLine int
	if input.Offset != nil || input.Limit != nil {
		content, startLine, endLine = applyOffsetLimit(lines, input.Offset, input.Limit)
	} else if input.Head != nil || input.Tail != nil {
		content, startLine, endLine = applyHeadTailWithRange(lines, input.Head, input.Tail)
	} else {
		content = strings.Join(lines, "\n")
		startLine = 1
		endLine = totalLines
	}

	// Build output
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

// resolveEncoding determines the encoding to use, either from explicit input or auto-detection.
// Uses streaming detection to avoid loading the entire file for encoding detection.
func resolveEncoding(inputEncoding string, filePath string) (encodingResult, error) {
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

	// Auto-detect encoding using streaming (sample mode - reads only ~384KB max)
	result.autoDetected = true
	detection, err := encoding.DetectFromFile(filePath, "sample")
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

// applyHeadTailWithRange applies head or tail and returns range info.
// Returns content, startLine, endLine.
func applyHeadTailWithRange(lines []string, head, tail *int) (string, int, int) {
	totalLines := len(lines)

	if head != nil {
		n := *head
		if n >= 0 && n < totalLines {
			return strings.Join(lines[:n], "\n"), 1, n
		}
		return strings.Join(lines, "\n"), 1, totalLines
	}

	if tail != nil {
		n := *tail
		if n >= 0 && n < totalLines {
			startLine := totalLines - n + 1
			return strings.Join(lines[totalLines-n:], "\n"), startLine, totalLines
		}
		return strings.Join(lines, "\n"), 1, totalLines
	}

	return strings.Join(lines, "\n"), 1, totalLines
}
