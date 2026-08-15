// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pmezard/go-difflib/difflib"
)

// HandleEditFile applies line-based edits to a text file with encoding support.
func (h *Handler) HandleEditFile(ctx context.Context, req *mcp.CallToolRequest, input EditFileInput) (*mcp.CallToolResult, EditFileOutput, error) {
	if input.Patch != "" && input.Edits != nil {
		return errorResult("patch and edits are mutually exclusive"), EditFileOutput{}, nil
	}
	if input.Patch == "" && len(input.Edits) == 0 {
		return errorResult("provide either a non-empty edits array or patch"), EditFileOutput{}, nil
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
	forceWritable := input.ForceWritable != nil && *input.ForceWritable // default: false
	if isReadOnly(originalMode) {
		if !forceWritable {
			return errorResult("file is read-only — STOP, do NOT retry and do NOT attempt to change file attributes. Ask the user whether to proceed with forceWritable: true, or skip this file"), EditFileOutput{}, nil
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

	encodingName, err := h.resolveEncodingFromData(input.Encoding, data, input.Path)
	if err != nil {
		return errorResult(err.Error()), EditFileOutput{}, nil
	}

	content, err := encoding.Decode(data, encodingName) // name already validated above
	if err != nil {
		return errorResult(fmt.Sprintf("failed to decode file with %s: %v", encodingName, err)), EditFileOutput{}, nil
	}
	slog.Debug("edit_file: decoded content", "path", input.Path, "encoding", encodingName, "originalSize", len(data), "decodedSize", len(content))

	// Detect on decoded text: UTF-16 has a 00 between CR and LF.
	lineEndings := DetectLineEndings([]byte(content))
	if lineEndings.Style == LineEndingMixed {
		slog.Warn("file has mixed line endings", "path", input.Path, "crlf", lineEndings.CRLFCount, "lf", lineEndings.LFCount)
	}

	content = ConvertLineEndings(content, LineEndingLF)
	var modifiedContent string
	var replacements int
	if input.Patch != "" {
		modifiedContent, err = applyPatch(content, input.Patch)
	} else {
		modifiedContent, replacements, err = applyEdits(content, input.Edits)
	}
	if err != nil {
		return errorResult(err.Error()), EditFileOutput{}, nil
	}

	diff := createUnifiedDiff(content, modifiedContent, input.Path)

	if !input.DryRun {
		if r := cancelled(ctx); r != nil {
			return r, EditFileOutput{}, nil
		}
		if err := atomicWriteFileWithEncoding(v.Path, modifiedContent, encodingName, lineEndings.Style, originalMode); err != nil {
			return errorResult(fmt.Sprintf("failed to write file: %v", err)), EditFileOutput{}, nil
		}
	}

	text := diff
	if replacements > 1 {
		text += fmt.Sprintf("\nreplaceAll changed %d places — tell the user how many.", replacements)
	}
	if readOnlyCleared {
		text += "\nRead-only flag was cleared."
	}

	output := EditFileOutput{Diff: diff, ReadOnlyCleared: readOnlyCleared}
	if replacements > 1 {
		output.Replacements = replacements
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, output, nil
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

// atomicWriteFileWithEncoding encodes UTF-8 content to the target encoding and writes atomically.
// mode is passed in because the caller reads it before clearing any read-only flag.
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
		encoded, err := encoding.Encode(content, enc, encodingName)
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
