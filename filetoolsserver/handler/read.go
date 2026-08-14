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
	fallbackHint       string // set when a detected encoding was discarded
}

func (h *Handler) HandleReadTextFile(ctx context.Context, req *mcp.CallToolRequest, input ReadTextFileInput) (*mcp.CallToolResult, ReadTextFileOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, ReadTextFileOutput{}, nil
	}

	// Get file size early for the output
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

	// Apply maxCharacters truncation (counts Unicode runes, not bytes)
	truncated := false
	if input.MaxCharacters != nil && *input.MaxCharacters > 0 && utf8.RuneCountInString(content) > *input.MaxCharacters {
		// Truncate at rune boundary
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

// encodingSource says which branch of resolveWriteEncoding produced the name.
type encodingSource int

const (
	encodingFromRequest  encodingSource = iota // explicit encoding parameter
	encodingFromExisting                       // detected on the file being overwritten
	encodingFromDefault                        // configured default, file did not exist
	encodingFromFallback                       // configured default, existing file was inconclusive
)

// resolveWriteEncoding returns encoding for writes: explicit > existing file > config default.
func (h *Handler) resolveWriteEncoding(inputEncoding string, filePath string) (string, encodingSource, error) {
	// 1. Explicit encoding always wins
	if inputEncoding != "" {
		encodingName := strings.ToLower(inputEncoding)
		if _, ok := encoding.Get(encodingName); !ok {
			return "", encodingFromRequest, fmt.Errorf("%w: %s. Use list_encodings to see available encodings", ErrEncodingUnsupported, encodingName)
		}
		return encodingName, encodingFromRequest, nil
	}

	// 2. If file exists, detect and preserve its encoding
	fileExists := false
	if _, err := os.Stat(filePath); err == nil {
		fileExists = true
		detected, err := encoding.DetectFromFile(filePath, "sample")
		// "ascii" fits every encoding we support, so it's inconclusive, not a match.
		if err == nil && detected.Confidence >= encoding.MinConfidenceThreshold && detected.Charset != "ascii" {
			// Validate the detected encoding is supported
			if _, ok := encoding.Get(detected.Charset); ok {
				slog.Debug("preserving existing file encoding", "path", filePath, "encoding", detected.Charset, "confidence", detected.Confidence)
				return detected.Charset, encodingFromExisting, nil
			}
		}
		// Detection failed, ascii, or low confidence - fall through to default
		slog.Debug("encoding detection inconclusive, using default", "path", filePath, "detected", detected.Charset, "confidence", detected.Confidence)
	}

	// 3. New file or detection failed - use configured default
	if fileExists {
		return h.config.DefaultEncoding, encodingFromFallback, nil
	}
	return h.config.DefaultEncoding, encodingFromDefault, nil
}

// resolveEncodingFromData returns encoding from loaded data: explicit > auto-detect.
func (h *Handler) resolveEncodingFromData(inputEncoding string, data []byte, filePath string) (string, error) {
	// 1. Explicit encoding always wins
	if inputEncoding != "" {
		encodingName := strings.ToLower(inputEncoding)
		if _, ok := encoding.Get(encodingName); !ok {
			return "", fmt.Errorf("%w: %s. Use list_encodings to see available encodings", ErrEncodingUnsupported, encodingName)
		}
		return encodingName, nil
	}

	// 2. Auto-detect from loaded data ("ascii" is inconclusive - see resolveWriteEncoding)
	detected := encoding.Detect(data)
	if detected.Confidence >= encoding.MinConfidenceThreshold && detected.Charset != "ascii" {
		if _, ok := encoding.Get(detected.Charset); ok {
			slog.Debug("auto-detected encoding from data", "path", filePath, "encoding", detected.Charset, "confidence", detected.Confidence)
			return detected.Charset, nil
		}
	}

	// 3. Fall back to configured default, same as resolveWriteEncoding's existing-file branch
	slog.Debug("encoding detection inconclusive, using configured default", "path", filePath, "detected", detected.Charset, "confidence", detected.Confidence, "default", h.config.DefaultEncoding)
	return h.config.DefaultEncoding, nil
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
		slog.Warn("detected encoding not supported, reading as utf-8", "path", filePath, "detected", detection.Charset, "confidence", detection.Confidence)
		result.encoder = nil
		result.name = "utf-8"
		result.detectedEncoding = result.detectedEncoding + " (unsupported, using utf-8)"
		// Phrased as an instruction because models relay instructions and ignore trivia.
		result.fallbackHint = fmt.Sprintf(
			"Detected encoding %s is not supported; the file was read as raw utf-8, so non-ASCII text is likely garbled — tell the user.",
			detection.Charset)
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
