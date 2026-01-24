package handler

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleReadFile reads a file in the specified encoding and returns UTF-8 content
func (h *Handler) HandleReadFile(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ReadFileInput]) (*mcp.CallToolResultFor[ReadFileOutput], error) {
	input := params.Arguments

	// Validate path
	if input.Path == "" {
		return &mcp.CallToolResultFor[ReadFileOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: "path is required and must be a non-empty string"}},
			IsError: true,
		}, nil
	}

	// Read file first (needed for auto-detection)
	data, err := os.ReadFile(input.Path)
	if err != nil {
		return &mcp.CallToolResultFor[ReadFileOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to read file: %v", err)}},
			IsError: true,
		}, nil
	}

	// Determine encoding
	encodingName := strings.ToLower(input.Encoding)
	var detection encoding.DetectionResult

	if encodingName == "" {
		// No encoding specified - auto-detect using chardet
		detection = encoding.Detect(data)
		if detection.Charset != "" {
			encodingName = detection.Charset
		} else {
			// Fallback to default if detection failed
			encodingName = h.defaultEncoding
		}
	}

	// Validate encoding
	enc, ok := encoding.Get(encodingName)
	if !ok {
		return &mcp.CallToolResultFor[ReadFileOutput]{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("unsupported encoding: %s. Use list_encodings to see available encodings.", encodingName),
			}},
			IsError: true,
		}, nil
	}

	// UTF-8: return content as-is (no conversion needed)
	if encoding.IsUTF8(encodingName) {
		text := string(data)
		if detection.Charset != "" {
			if detection.HasBOM {
				text = fmt.Sprintf("[Auto-detected: UTF-8 with BOM, confidence: %d%%]\n\n%s", detection.Confidence, text)
			} else {
				text = fmt.Sprintf("[Auto-detected: UTF-8, confidence: %d%%]\n\n%s", detection.Confidence, text)
			}
		}
		return &mcp.CallToolResultFor[ReadFileOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: text}},
		}, nil
	}

	// Decode from specified encoding to UTF-8
	decoder := enc.NewDecoder()
	utf8Content, err := decoder.Bytes(data)
	if err != nil {
		return &mcp.CallToolResultFor[ReadFileOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to decode file content: %v", err)}},
			IsError: true,
		}, nil
	}

	// Add detection info if auto-detected
	text := string(utf8Content)
	if detection.Charset != "" {
		text = fmt.Sprintf("[Auto-detected: %s, confidence: %d%%]\n\n%s", encodingName, detection.Confidence, text)
	}

	return &mcp.CallToolResultFor[ReadFileOutput]{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, nil
}
