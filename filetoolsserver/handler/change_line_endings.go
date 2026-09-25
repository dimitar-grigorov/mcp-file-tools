// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// convertWideLineEndings rewrites line endings per code unit, 2 bytes wide for UTF-16 and 4 for UTF-32. Data must be BOM-free.
func convertWideLineEndings(data []byte, targetStyle string, width int, littleEndian bool) ([]byte, error) {
	if len(data)%width != 0 {
		return nil, fmt.Errorf("truncated data: %d bytes is not a whole number of %d-byte code units", len(data), width)
	}

	var order interface {
		binary.ByteOrder
		binary.AppendByteOrder
	} = binary.BigEndian
	if littleEndian {
		order = binary.LittleEndian
	}
	unitAt := func(i int) uint32 {
		if width == 2 {
			return uint32(order.Uint16(data[i:]))
		}
		return order.Uint32(data[i:])
	}
	putUnit := func(dst []byte, unit uint32) []byte {
		if width == 2 {
			return order.AppendUint16(dst, uint16(unit))
		}
		return order.AppendUint32(dst, unit)
	}

	converted := make([]byte, 0, len(data))
	for i := 0; i < len(data); i += width {
		unit := unitAt(i)
		if unit == '\r' && i+width < len(data) && unitAt(i+width) == '\n' {
			unit = '\n'
			i += width
		}
		if unit == '\n' {
			if targetStyle == LineEndingCRLF {
				converted = putUnit(converted, '\r')
			}
			converted = putUnit(converted, '\n')
			continue
		}
		converted = append(converted, data[i:i+width]...)
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

	if originalStyle == style || originalStyle == LineEndingNone {
		return &mcp.CallToolResult{}, ChangeLineEndingsOutput{
			Message:       fmt.Sprintf("File already uses %s line endings, no changes needed", style),
			OriginalStyle: originalStyle,
			NewStyle:      style,
			LinesChanged:  0,
		}, nil
	}

	var linesChanged int
	if style == LineEndingLF {
		linesChanged = info.CRLFCount
	} else {
		linesChanged = info.LFCount
	}

	// UTF-16 and UTF-32 need code units; every other registered encoding is ASCII-transparent.
	var converted []byte
	canonical, _ := encoding.Canonical(encResult.name)
	switch canonical {
	case "utf-16-le", "utf-16-be", "utf-32-le", "utf-32-be":
		width := 2
		if strings.HasPrefix(canonical, "utf-32") {
			width = 4
		}
		bom := bomPrefix(data, canonical)
		payload, err := convertWideLineEndings(data[len(bom):], style, width, strings.HasSuffix(canonical, "-le"))
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

	if err := rewriteFile(v.Path, converted); err != nil {
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
