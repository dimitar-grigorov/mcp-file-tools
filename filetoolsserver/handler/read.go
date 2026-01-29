package handler

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleReadTextFile reads a file in the specified encoding and returns UTF-8 content
func (h *Handler) HandleReadTextFile(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ReadTextFileInput]) (*mcp.CallToolResultFor[ReadTextFileOutput], error) {
	input := params.Arguments

	// Validate path
	if input.Path == "" {
		return &mcp.CallToolResultFor[ReadTextFileOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: "path is required and must be a non-empty string"}},
			IsError: true,
		}, nil
	}

	// Validate head/tail - cannot specify both
	if input.Head != nil && input.Tail != nil {
		return &mcp.CallToolResultFor[ReadTextFileOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: "cannot specify both head and tail"}},
			IsError: true,
		}, nil
	}

	// Validate path against allowed directories
	validatedPath, err := h.validatePath(input.Path)
	if err != nil {
		return &mcp.CallToolResultFor[ReadTextFileOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
			IsError: true,
		}, nil
	}

	// Read file
	data, err := os.ReadFile(validatedPath)
	if err != nil {
		return &mcp.CallToolResultFor[ReadTextFileOutput]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to read file: %v", err)}},
			IsError: true,
		}, nil
	}

	// Determine encoding - default to UTF-8
	encodingName := strings.ToLower(input.Encoding)
	if encodingName == "" {
		encodingName = "utf-8"
	}

	// Validate encoding
	enc, ok := encoding.Get(encodingName)
	if !ok {
		return &mcp.CallToolResultFor[ReadTextFileOutput]{
			Content: []mcp.Content{&mcp.TextContent{
				Text: fmt.Sprintf("unsupported encoding: %s. Use list_encodings to see available encodings.", encodingName),
			}},
			IsError: true,
		}, nil
	}

	var content string

	// UTF-8: return content as-is (no conversion needed)
	if encoding.IsUTF8(encodingName) {
		content = string(data)
	} else {
		// Decode from specified encoding to UTF-8
		decoder := enc.NewDecoder()
		utf8Content, err := decoder.Bytes(data)
		if err != nil {
			return &mcp.CallToolResultFor[ReadTextFileOutput]{
				Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("failed to decode file content: %v", err)}},
				IsError: true,
			}, nil
		}
		content = string(utf8Content)
	}

	// Apply head/tail if specified
	if input.Head != nil || input.Tail != nil {
		lines := strings.Split(content, "\n")

		if input.Head != nil {
			n := *input.Head
			if n >= 0 && n < len(lines) {
				lines = lines[:n]
			}
		} else if input.Tail != nil {
			n := *input.Tail
			if n >= 0 && n < len(lines) {
				lines = lines[len(lines)-n:]
			}
		}

		content = strings.Join(lines, "\n")
	}

	return &mcp.CallToolResultFor[ReadTextFileOutput]{
		Content: []mcp.Content{&mcp.TextContent{Text: content}},
	}, nil
}
