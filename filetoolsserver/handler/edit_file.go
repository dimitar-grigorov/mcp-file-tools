package handler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pmezard/go-difflib/difflib"
)

// HandleEditFile applies line-based edits to a text file with encoding support.
func (h *Handler) HandleEditFile(ctx context.Context, req *mcp.CallToolRequest, input EditFileInput) (*mcp.CallToolResult, EditFileOutput, error) {
	if len(input.Edits) == 0 {
		return errorResult(ErrEditsRequired.Error()), EditFileOutput{}, nil
	}

	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, EditFileOutput{}, nil
	}

	if loadToMemory, size := h.shouldLoadEntireFile(v.Path); !loadToMemory {
		slog.Warn("loading large file into memory", "path", input.Path, "size", size, "threshold", h.config.MemoryThreshold)
	}

	originalMode := getFileMode(v.Path)

	readOnlyCleared := false
	forceWritable := input.ForceWritable == nil || *input.ForceWritable // default: true
	if isReadOnly(originalMode) {
		if !forceWritable {
			return errorResult("file is read-only; set forceWritable: true to clear the read-only flag"), EditFileOutput{}, nil
		}
		if !input.DryRun {
			if err := clearReadOnly(v.Path, originalMode); err != nil {
				return errorResult(fmt.Sprintf("failed to clear read-only flag: %v", err)), EditFileOutput{}, nil
			}
			readOnlyCleared = true
			slog.Info("cleared read-only flag", "path", input.Path)
		}
	}

	data, err := os.ReadFile(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read file: %v", err)), EditFileOutput{}, nil
	}

	// TODO: Use DetectLineEndingsFromFile for streaming when file > MemoryThreshold
	lineEndings := DetectLineEndings(data)
	if lineEndings.Style == LineEndingMixed {
		slog.Warn("file has mixed line endings", "path", input.Path, "crlf", lineEndings.CRLFCount, "lf", lineEndings.LFCount)
	}

	encodingName, err := h.resolveEncodingFromData(input.Encoding, data, input.Path)
	if err != nil {
		return errorResult(err.Error()), EditFileOutput{}, nil
	}

	var content string
	if encoding.IsUTF8(encodingName) {
		content = string(data)
	} else {
		enc, _ := encoding.Get(encodingName) // Already validated by resolveEncodingFromData
		decoder := enc.NewDecoder()
		decoded, err := decoder.Bytes(data)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to decode file with %s: %v", encodingName, err)), EditFileOutput{}, nil
		}
		content = string(decoded)
		slog.Debug("edit_file: decoded content", "path", input.Path, "encoding", encodingName, "originalSize", len(data), "decodedSize", len(decoded))
	}

	content = ConvertLineEndings(content, LineEndingLF)
	modifiedContent, err := applyEdits(content, input.Edits)
	if err != nil {
		return errorResult(err.Error()), EditFileOutput{}, nil
	}

	diff := createUnifiedDiff(content, modifiedContent, input.Path)
	formattedDiff := formatDiffOutput(diff)

	if !input.DryRun {
		if err := atomicWriteFileWithEncoding(v.Path, modifiedContent, encodingName, lineEndings.Style, originalMode); err != nil {
			return errorResult(fmt.Sprintf("failed to write file: %v", err)), EditFileOutput{}, nil
		}
	}

	return &mcp.CallToolResult{}, EditFileOutput{Diff: formattedDiff, ReadOnlyCleared: readOnlyCleared}, nil
}

// applyEdits applies edits sequentially, trying exact match then whitespace-flexible match.
func applyEdits(content string, edits []EditOperation) (string, error) {
	modifiedContent := content

	for _, edit := range edits {
		if edit.OldText == "" {
			return "", ErrOldTextEmpty
		}

		normalizedOld := ConvertLineEndings(edit.OldText, LineEndingLF)
		normalizedNew := ConvertLineEndings(edit.NewText, LineEndingLF)

		// Try exact match first
		if strings.Contains(modifiedContent, normalizedOld) {
			modifiedContent = strings.Replace(modifiedContent, normalizedOld, normalizedNew, 1)
			continue
		}

		// Try whitespace-flexible line matching
		matched, result := tryFlexibleMatch(modifiedContent, normalizedOld, normalizedNew)
		if matched {
			modifiedContent = result
			continue
		}

		return "", fmt.Errorf("%w:\n%s", ErrEditNoMatch, edit.OldText)
	}

	return modifiedContent, nil
}

// tryFlexibleMatch matches oldText ignoring whitespace differences, preserving file indentation.
func tryFlexibleMatch(content, oldText, newText string) (bool, string) {
	oldLines := strings.Split(oldText, "\n")
	contentLines := strings.Split(content, "\n")

	if len(contentLines) < len(oldLines) {
		return false, ""
	}

	for i := 0; i <= len(contentLines)-len(oldLines); i++ {
		potentialMatch := contentLines[i : i+len(oldLines)]

		isMatch := true
		for j, oldLine := range oldLines {
			if strings.TrimSpace(oldLine) != strings.TrimSpace(potentialMatch[j]) {
				isMatch = false
				break
			}
		}

		if isMatch {
			originalIndent := getLeadingWhitespace(contentLines[i])
			newLines := strings.Split(newText, "\n")

			for j := range newLines {
				if j == 0 {
					newLines[j] = originalIndent + strings.TrimLeft(newLines[j], " \t")
				} else {
					newLines[j] = adjustRelativeIndent(oldLines, newLines[j], j, originalIndent)
				}
			}

			result := make([]string, 0, len(contentLines)-len(oldLines)+len(newLines))
			result = append(result, contentLines[:i]...)
			result = append(result, newLines...)
			result = append(result, contentLines[i+len(oldLines):]...)

			return true, strings.Join(result, "\n")
		}
	}

	return false, ""
}

// adjustRelativeIndent applies baseIndent plus the indentation delta between old and new lines.
func adjustRelativeIndent(oldLines []string, newLine string, lineIndex int, baseIndent string) string {
	if lineIndex >= len(oldLines) {
		return newLine
	}

	oldIndent := getLeadingWhitespace(oldLines[lineIndex])
	newIndent := getLeadingWhitespace(newLine)

	relativeIndent := len(newIndent) - len(oldIndent)
	trimmedContent := strings.TrimLeft(newLine, " \t")
	switch {
	case relativeIndent > 0:
		return baseIndent + strings.Repeat(" ", relativeIndent) + trimmedContent
	case relativeIndent < 0:
		// Negative indent: trim characters from the end of baseIndent
		trim := -relativeIndent
		if trim >= len(baseIndent) {
			return trimmedContent
		}
		return baseIndent[:len(baseIndent)-trim] + trimmedContent
	default:
		return baseIndent + trimmedContent
	}
}

func getLeadingWhitespace(s string) string {
	for i, c := range s {
		if c != ' ' && c != '\t' {
			return s[:i]
		}
	}
	return s // entire string is whitespace
}

func createUnifiedDiff(original, modified, filepath string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(original),
		B:        difflib.SplitLines(modified),
		FromFile: filepath,
		ToFile:   filepath,
		Context:  3,
	}
	text, _ := difflib.GetUnifiedDiffString(diff)
	return text
}

// formatDiffOutput wraps diff in a markdown code fence, escaping backticks if needed.
func formatDiffOutput(diff string) string {
	numBackticks := 3
	for strings.Contains(diff, strings.Repeat("`", numBackticks)) {
		numBackticks++
	}
	fence := strings.Repeat("`", numBackticks)
	return fmt.Sprintf("%sdiff\n%s%s\n\n", fence, diff, fence)
}

// atomicWriteFileWithEncoding encodes UTF-8 content to the target encoding and writes atomically.
func atomicWriteFileWithEncoding(path, content, encodingName, lineEndingStyle string, mode os.FileMode) error {
	content = ConvertLineEndings(content, lineEndingStyle)

	var dataToWrite []byte
	if encoding.IsUTF8(encodingName) {
		dataToWrite = []byte(content)
	} else {
		enc, ok := encoding.Get(encodingName)
		if !ok {
			return fmt.Errorf("unsupported encoding: %s", encodingName)
		}
		encoder := enc.NewEncoder()
		encoded, err := encoder.Bytes([]byte(content))
		if err != nil {
			return fmt.Errorf("failed to encode content to %s: %w", encodingName, err)
		}
		dataToWrite = encoded
		slog.Debug("edit_file: encoded content for write", "encoding", encodingName, "utf8Size", len(content), "encodedSize", len(encoded))
	}

	return atomicWriteFile(path, dataToWrite, mode)
}

func isReadOnly(mode os.FileMode) bool {
	return mode&0200 == 0
}

// clearReadOnly adds owner write permission to the file.
func clearReadOnly(path string, currentMode os.FileMode) error {
	newMode := currentMode | 0200
	return os.Chmod(path, newMode)
}
