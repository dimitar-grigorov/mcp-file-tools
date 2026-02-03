package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pmezard/go-difflib/difflib"
)

// HandleEditFile applies line-based edits to a text file.
// Supports non-UTF-8 encodings via auto-detection or explicit encoding parameter.
func (h *Handler) HandleEditFile(ctx context.Context, req *mcp.CallToolRequest, input EditFileInput) (*mcp.CallToolResult, EditFileOutput, error) {
	if len(input.Edits) == 0 {
		return errorResult(ErrEditsRequired.Error()), EditFileOutput{}, nil
	}

	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, EditFileOutput{}, nil
	}

	// Check file size - warn if large file will be loaded to memory
	if loadToMemory, size := h.shouldLoadEntireFile(v.Path); !loadToMemory {
		slog.Warn("loading large file into memory", "path", input.Path, "size", size, "threshold", h.config.MemoryThreshold)
	}

	// Preserve original file permissions
	originalMode := getFileMode(v.Path)

	// Handle read-only files
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

	// Detect line endings before any processing
	// TODO: Use DetectLineEndingsFromFile for streaming when file > MemoryThreshold
	lineEndings := DetectLineEndings(data)
	if lineEndings.Style == LineEndingMixed {
		slog.Warn("file has mixed line endings", "path", input.Path, "crlf", lineEndings.CRLFCount, "lf", lineEndings.LFCount)
	}

	// Determine encoding (auto-detect or use provided)
	encodingName := strings.ToLower(input.Encoding)
	if encodingName == "" {
		// Auto-detect encoding
		detected := encoding.Detect(data)
		encodingName = detected.Charset
		slog.Debug("edit_file: auto-detected encoding", "path", input.Path, "encoding", encodingName, "confidence", detected.Confidence)
	}

	// Decode file content to UTF-8 for matching
	var content string
	if encoding.IsUTF8(encodingName) {
		content = string(data)
	} else {
		enc, ok := encoding.Get(encodingName)
		if !ok {
			return errorResult(fmt.Sprintf("unsupported encoding: %s", encodingName)), EditFileOutput{}, nil
		}
		decoder := enc.NewDecoder()
		decoded, err := decoder.Bytes(data)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to decode file with %s: %v", encodingName, err)), EditFileOutput{}, nil
		}
		content = string(decoded)
		slog.Debug("edit_file: decoded content", "path", input.Path, "encoding", encodingName, "originalSize", len(data), "decodedSize", len(decoded))
	}

	// Normalize line endings (CRLF -> LF) for consistent processing
	content = ConvertLineEndings(content, LineEndingLF)

	// Apply edits sequentially
	modifiedContent, err := applyEdits(content, input.Edits)
	if err != nil {
		return errorResult(err.Error()), EditFileOutput{}, nil
	}

	// Generate unified diff
	diff := createUnifiedDiff(content, modifiedContent, input.Path)

	// Format diff with markdown code fence
	formattedDiff := formatDiffOutput(diff)

	// Write file if not dry run (atomic write with encoding and line ending preservation)
	if !input.DryRun {
		if err := atomicWriteFileWithEncoding(v.Path, modifiedContent, encodingName, lineEndings.Style, originalMode); err != nil {
			return errorResult(fmt.Sprintf("failed to write file: %v", err)), EditFileOutput{}, nil
		}
	}

	return &mcp.CallToolResult{}, EditFileOutput{Diff: formattedDiff, ReadOnlyCleared: readOnlyCleared}, nil
}

// applyEdits applies a sequence of edit operations to content.
// Each edit is tried first as an exact match, then with whitespace-flexible matching.
func applyEdits(content string, edits []EditOperation) (string, error) {
	modifiedContent := content

	for _, edit := range edits {
		// Validate that oldText is not empty
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

// tryFlexibleMatch attempts to match oldText in content with whitespace flexibility.
// It compares lines with trimmed whitespace and preserves original indentation.
func tryFlexibleMatch(content, oldText, newText string) (bool, string) {
	oldLines := strings.Split(oldText, "\n")
	contentLines := strings.Split(content, "\n")

	// Need at least as many content lines as old lines to match
	if len(contentLines) < len(oldLines) {
		return false, ""
	}

	for i := 0; i <= len(contentLines)-len(oldLines); i++ {
		potentialMatch := contentLines[i : i+len(oldLines)]

		// Compare lines with trimmed whitespace
		isMatch := true
		for j, oldLine := range oldLines {
			if strings.TrimSpace(oldLine) != strings.TrimSpace(potentialMatch[j]) {
				isMatch = false
				break
			}
		}

		if isMatch {
			// Preserve original indentation from the file
			originalIndent := getLeadingWhitespace(contentLines[i])
			newLines := strings.Split(newText, "\n")

			// Apply indentation to replacement lines
			for j := range newLines {
				if j == 0 {
					// First line: use original file's indentation + trimmed new content
					newLines[j] = originalIndent + strings.TrimLeft(newLines[j], " \t")
				} else {
					// Subsequent lines: preserve relative indentation
					newLines[j] = adjustRelativeIndent(oldLines, newLines[j], j, originalIndent)
				}
			}

			// Replace in content
			result := make([]string, 0, len(contentLines)-len(oldLines)+len(newLines))
			result = append(result, contentLines[:i]...)
			result = append(result, newLines...)
			result = append(result, contentLines[i+len(oldLines):]...)

			return true, strings.Join(result, "\n")
		}
	}

	return false, ""
}

// adjustRelativeIndent calculates and applies relative indentation for a replacement line.
// It compares the indentation of the new line to the old line at the same position,
// then applies the base indentation plus any relative change.
func adjustRelativeIndent(oldLines []string, newLine string, lineIndex int, baseIndent string) string {
	// If we don't have a corresponding old line, return the line as-is
	if lineIndex >= len(oldLines) {
		return newLine
	}

	oldIndent := getLeadingWhitespace(oldLines[lineIndex])
	newIndent := getLeadingWhitespace(newLine)

	// Calculate the relative indentation change
	relativeIndent := len(newIndent) - len(oldIndent)

	// Apply base indentation + relative change
	if relativeIndent > 0 {
		return baseIndent + strings.Repeat(" ", relativeIndent) + strings.TrimLeft(newLine, " \t")
	}
	return baseIndent + strings.TrimLeft(newLine, " \t")
}

// getLeadingWhitespace extracts leading spaces and tabs from a string
func getLeadingWhitespace(s string) string {
	for i, c := range s {
		if c != ' ' && c != '\t' {
			return s[:i]
		}
	}
	return s // entire string is whitespace
}

// createUnifiedDiff generates a git-style unified diff between original and modified content
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

// formatDiffOutput wraps the diff in a markdown code fence with 'diff' syntax highlighting.
// It handles cases where the diff itself contains backticks by using more backticks.
func formatDiffOutput(diff string) string {
	numBackticks := 3
	for strings.Contains(diff, strings.Repeat("`", numBackticks)) {
		numBackticks++
	}
	fence := strings.Repeat("`", numBackticks)
	return fmt.Sprintf("%sdiff\n%s%s\n\n", fence, diff, fence)
}

// atomicWriteFileWithEncoding writes content to a file atomically with encoding conversion.
// Content is expected to be UTF-8 (with LF line endings) and will be encoded to the specified encoding.
// Line endings are restored to the original style before writing.
// File permissions are preserved from the original file.
func atomicWriteFileWithEncoding(filepath, content, encodingName, lineEndingStyle string, mode os.FileMode) (err error) {
	// Generate random temp filename
	randBytes := make([]byte, 16)
	if _, err := rand.Read(randBytes); err != nil {
		return fmt.Errorf("failed to generate temp filename: %w", err)
	}
	tempPath := fmt.Sprintf("%s.%s.tmp", filepath, hex.EncodeToString(randBytes))

	// Ensure cleanup on any error or panic
	defer func() {
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	// Restore original line endings before encoding
	content = ConvertLineEndings(content, lineEndingStyle)

	// Encode content if not UTF-8
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

	// Write to temp file with original permissions
	if err = os.WriteFile(tempPath, dataToWrite, mode); err != nil {
		return err
	}

	// Atomic rename
	if err = os.Rename(tempPath, filepath); err != nil {
		return err
	}

	return nil
}

// isReadOnly checks if the file mode indicates the file is read-only.
// On Unix, checks if the owner write bit is not set.
// On Windows, this also works as Go maps the read-only attribute to mode bits.
func isReadOnly(mode os.FileMode) bool {
	return mode&0200 == 0 // owner write bit not set
}

// clearReadOnly removes the read-only flag from a file.
// Adds owner write permission (0200) to the existing mode.
func clearReadOnly(path string, currentMode os.FileMode) error {
	newMode := currentMode | 0200 // add owner write permission
	return os.Chmod(path, newMode)
}
