package handler

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// convertUTF16LineEndings rewrites line endings per code unit. Data must be BOM-free.
func convertUTF16LineEndings(data []byte, targetStyle string, littleEndian bool) ([]byte, error) {
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("truncated UTF-16 data: %d bytes", len(data))
	}

	unitAt := func(i int) uint16 {
		if littleEndian {
			return uint16(data[i]) | uint16(data[i+1])<<8
		}
		return uint16(data[i])<<8 | uint16(data[i+1])
	}
	putUnit := func(dst []byte, unit uint16) []byte {
		if littleEndian {
			return append(dst, byte(unit), byte(unit>>8))
		}
		return append(dst, byte(unit>>8), byte(unit))
	}

	converted := make([]byte, 0, len(data))
	for i := 0; i < len(data); i += 2 {
		unit := unitAt(i)
		if unit == '\r' && i+2 < len(data) && unitAt(i+2) == '\n' {
			unit = '\n'
			i += 2
		}
		if unit == '\n' {
			if targetStyle == LineEndingCRLF {
				converted = putUnit(converted, '\r')
			}
			converted = putUnit(converted, '\n')
			continue
		}
		converted = append(converted, data[i], data[i+1])
	}
	return converted, nil
}

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

	encResult, err := h.resolveEncoding(input.Encoding, v.Path)
	if err != nil {
		return errorResult(err.Error()), ChangeLineEndingsOutput{}, nil
	}

	data, err := os.ReadFile(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read file: %v", err)), ChangeLineEndingsOutput{}, nil
	}

	// Detect on decoded text: UTF-16 has a 00 between CR and LF.
	content, err := decodeContent(data, encResult)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to decode file content: %v", err)), ChangeLineEndingsOutput{}, nil
	}
	info := DetectLineEndings([]byte(content))
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

	// UTF-16 needs code units; every other registered encoding is ASCII-transparent.
	var converted []byte
	canonical, _ := encoding.Canonical(encResult.name)
	switch canonical {
	case "utf-16-le", "utf-16-be":
		bom := bomPrefix(data, canonical)
		payload, err := convertUTF16LineEndings(data[len(bom):], style, canonical == "utf-16-le")
		if err != nil {
			return errorResult(fmt.Sprintf("failed to convert %s line endings: %v", canonical, err)), ChangeLineEndingsOutput{}, nil
		}
		converted = make([]byte, 0, len(bom)+len(payload))
		converted = append(converted, bom...)
		converted = append(converted, payload...)
	default:
		converted = []byte(ConvertLineEndings(string(data), style))
	}

	if r := cancelled(ctx); r != nil {
		return r, ChangeLineEndingsOutput{}, nil
	}

	mode := getFileMode(v.Path)
	if err := atomicWriteFile(v.Path, converted, mode); err != nil {
		return errorResult(fmt.Sprintf("failed to write file: %v", err)), ChangeLineEndingsOutput{}, nil
	}

	return &mcp.CallToolResult{}, ChangeLineEndingsOutput{
		Message:       fmt.Sprintf("Converted %s from %s to %s (%d lines changed)", input.Path, originalStyle, style, linesChanged),
		OriginalStyle: originalStyle,
		NewStyle:      style,
		LinesChanged:  linesChanged,
	}, nil
}

// bomPrefix returns the leading BOM bytes when they match the given encoding.
func bomPrefix(data []byte, canonical string) []byte {
	result, found := encoding.DetectBOM(data)
	if !found || result.Charset != canonical {
		return nil
	}
	return data[:encoding.BOMSize(result.Charset)]
}
