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
func (h *Handler) HandleReadTextFile(ctx context.Context, req *mcp.CallToolRequest, input ReadTextFileInput) (*mcp.CallToolResult, ReadTextFileOutput, error) {
	// Validate input
	if err := validateReadInput(input); err != nil {
		return errorResult(err.Error()), ReadTextFileOutput{}, nil
	}

	// Validate path against allowed directories
	validatedPath, err := h.validatePath(input.Path)
	if err != nil {
		return errorResult(err.Error()), ReadTextFileOutput{}, nil
	}

	// Read file
	data, err := os.ReadFile(validatedPath)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read file: %v", err)), ReadTextFileOutput{}, nil
	}

	// Resolve encoding (explicit or auto-detect)
	encResult, err := resolveEncoding(input.Encoding, data)
	if err != nil {
		return errorResult(err.Error()), ReadTextFileOutput{}, nil
	}

	// Decode content to UTF-8
	content, err := decodeContent(data, encResult)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to decode file content: %v", err)), ReadTextFileOutput{}, nil
	}

	// Apply head/tail if specified
	content = applyHeadTail(content, input.Head, input.Tail)

	// Build output
	output := ReadTextFileOutput{Content: content}
	if encResult.autoDetected {
		output.DetectedEncoding = encResult.detectedEncoding
		output.EncodingConfidence = encResult.encodingConfidence
	}

	return &mcp.CallToolResult{}, output, nil
}

// validateReadInput validates the input parameters for reading a file
func validateReadInput(input ReadTextFileInput) error {
	if input.Path == "" {
		return ErrPathRequired
	}
	if input.Head != nil && input.Tail != nil {
		return ErrHeadTailConflict
	}
	return nil
}

// resolveEncoding determines the encoding to use, either from explicit input or auto-detection
func resolveEncoding(inputEncoding string, data []byte) (encodingResult, error) {
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

	// Auto-detect encoding
	result.autoDetected = true
	detection, trusted := encoding.DetectFromChunks(data)
	result.detectedEncoding = detection.Charset
	result.encodingConfidence = detection.Confidence

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

// applyHeadTail applies head or tail line limiting to the content
func applyHeadTail(content string, head, tail *int) string {
	if head == nil && tail == nil {
		return content
	}

	lines := strings.Split(content, "\n")

	if head != nil {
		n := *head
		if n >= 0 && n < len(lines) {
			lines = lines[:n]
		}
	} else if tail != nil {
		n := *tail
		if n >= 0 && n < len(lines) {
			lines = lines[len(lines)-n:]
		}
	}

	return strings.Join(lines, "\n")
}
