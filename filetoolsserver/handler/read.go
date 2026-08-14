// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *Handler) HandleReadTextFile(ctx context.Context, req *mcp.CallToolRequest, input ReadTextFileInput) (*mcp.CallToolResult, ReadTextFileOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, ReadTextFileOutput{}, nil
	}

	fileInfo, err := os.Stat(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to stat file: %v", err)), ReadTextFileOutput{}, nil
	}
	fileSizeBytes := fileInfo.Size()

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

	totalLines := countLines(content)

	var startLine, endLine int
	if input.Offset != nil || input.Limit != nil {
		content, startLine, endLine = applyOffsetLimit(content, input.Offset, input.Limit)
	} else {
		startLine = 1
		endLine = totalLines
	}

	if input.LineNumbers {
		content = addLineNumbers(content, startLine)
	}

	// maxCharacters counts runes, not bytes, and cuts on a rune boundary.
	truncated := false
	if input.MaxCharacters != nil && *input.MaxCharacters > 0 && utf8.RuneCountInString(content) > *input.MaxCharacters {
		runeCount := 0
		byteIdx := 0
		for byteIdx < len(content) && runeCount < *input.MaxCharacters {
			_, size := utf8.DecodeRuneInString(content[byteIdx:])
			byteIdx += size
			runeCount++
		}
		content = content[:byteIdx]
		content += fmt.Sprintf("\n\n[TRUNCATED at %d characters. File has %d lines, %d bytes. Use offset/limit for specific ranges.]",
			*input.MaxCharacters, totalLines, fileSizeBytes)
		truncated = true
	}

	output := ReadTextFileOutput{
		Content:       content,
		TotalLines:    totalLines,
		FileSizeBytes: fileSizeBytes,
		StartLine:     startLine,
		EndLine:       endLine,
		Truncated:     truncated,
	}
	if encResult.autoDetected {
		output.DetectedEncoding = encResult.detectedEncoding
		output.EncodingConfidence = encResult.encodingConfidence
	}
	var hints []string
	if le := DetectLineEndings(data); le.Style == LineEndingMixed {
		hints = append(hints, fmt.Sprintf(
			`This file has MIXED line endings (%d CRLF, %d LF) — tell the user, and use manage_line_endings action="convert" to normalise it.`,
			le.CRLFCount, le.LFCount))
	}
	if hint := h.plainUTF8HintFor(v.Path, encResult.name, existingBOM(v.Path).HasBOM); hint != "" {
		hints = append(hints, hint)
	}
	if encResult.fallbackHint != "" {
		hints = append(hints, encResult.fallbackHint)
	}
	output.Hint = strings.Join(hints, " ")

	return &mcp.CallToolResult{}, output, nil
}

// countLines counts lines; a trailing terminator doesn't open a new line, and a lone \r isn't a break.
func countLines(content string) int {
	if content == "" {
		return 0
	}
	n := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		n++
	}
	return n
}

// lineStarts returns the byte offset where each line begins.
func lineStarts(content string) []int {
	if content == "" {
		return nil
	}
	starts := make([]int, 0, countLines(content))
	starts = append(starts, 0)
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' && i+1 < len(content) {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// addLineNumbers prefixes each line with "N<tab>", numbering from start so a
// paged read shows absolute file line numbers. A trailing newline gets no number.
func addLineNumbers(content string, start int) string {
	if content == "" {
		return content
	}
	var b strings.Builder
	b.Grow(len(content) + len(content)/16)
	num := start
	for _, line := range strings.SplitAfter(content, "\n") {
		if line == "" { // after a trailing newline
			break
		}
		fmt.Fprintf(&b, "%d\t%s", num, line)
		num++
	}
	return b.String()
}

// applyOffsetLimit returns lines [offset, offset+limit) sliced from content (line endings preserved); offset is 1-indexed, negatives ignored.
func applyOffsetLimit(content string, offset, limit *int) (string, int, int) {
	starts := lineStarts(content)
	totalLines := len(starts)
	if totalLines == 0 {
		return "", 1, 0
	}

	startIdx := 0
	if offset != nil && *offset > 1 {
		startIdx = *offset - 1
		if startIdx >= totalLines {
			return "", totalLines + 1, totalLines // past the end
		}
	}

	endIdx := totalLines
	if limit != nil && *limit > 0 && startIdx+*limit < endIdx {
		endIdx = startIdx + *limit
	}

	// End at the next line's start, or at EOF for the last line (keeps its terminator).
	end := len(content)
	if endIdx < totalLines {
		end = starts[endIdx]
	}
	return content[starts[startIdx]:end], startIdx + 1, endIdx
}
