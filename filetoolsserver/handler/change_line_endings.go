package handler

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleChangeLineEndings converts line endings in a file to the specified style.
func (h *Handler) HandleChangeLineEndings(ctx context.Context, req *mcp.CallToolRequest, input ChangeLineEndingsInput) (*mcp.CallToolResult, ChangeLineEndingsOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, ChangeLineEndingsOutput{}, nil
	}

	style := strings.ToLower(input.Style)
	if style != LineEndingLF && style != LineEndingCRLF {
		return errorResult("style must be \"lf\" or \"crlf\""), ChangeLineEndingsOutput{}, nil
	}

	data, err := os.ReadFile(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read file: %v", err)), ChangeLineEndingsOutput{}, nil
	}

	// Detect current line endings
	info := DetectLineEndings(data)
	originalStyle := info.Style

	// Already in target style — no-op
	if originalStyle == style || originalStyle == LineEndingNone {
		return &mcp.CallToolResult{}, ChangeLineEndingsOutput{
			Message:       fmt.Sprintf("File already uses %s line endings, no changes needed", style),
			OriginalStyle: originalStyle,
			NewStyle:      style,
			LinesChanged:  0,
		}, nil
	}

	// Count lines that will change
	var linesChanged int
	if style == LineEndingLF {
		linesChanged = info.CRLFCount
	} else {
		linesChanged = info.LFCount
	}

	// Convert
	content := string(data)
	converted := ConvertLineEndings(content, style)

	mode := getFileMode(v.Path)
	if err := atomicWriteFile(v.Path, []byte(converted), mode); err != nil {
		return errorResult(fmt.Sprintf("failed to write file: %v", err)), ChangeLineEndingsOutput{}, nil
	}

	return &mcp.CallToolResult{}, ChangeLineEndingsOutput{
		Message:       fmt.Sprintf("Converted %s from %s to %s (%d lines changed)", input.Path, originalStyle, style, linesChanged),
		OriginalStyle: originalStyle,
		NewStyle:      style,
		LinesChanged:  linesChanged,
	}, nil
}
