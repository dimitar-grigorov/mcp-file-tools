// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

// Unified-diff patches for edit_file: parse the ---/+++/@@ format, replay hunks through the normal matcher.

import (
	"fmt"
	"strconv"
	"strings"
)

type patchHunk struct {
	oldStart int
	oldLines []string
	newLines []string
}

func applyPatch(content, patch string) (string, error) {
	hunks, err := parseUnifiedPatch(patch)
	if err != nil {
		return "", err
	}

	modified := content
	lineOffset := 0
	for i, hunk := range hunks {
		oldText := strings.Join(hunk.oldLines, "\n")
		newText := strings.Join(hunk.newLines, "\n")
		if len(hunk.oldLines) == 0 {
			modified, err = insertPatchLines(modified, newText, hunk.oldStart+lineOffset)
		} else {
			modified, _, err = applyEdits(modified, []EditOperation{{OldText: oldText, NewText: newText}})
		}
		if err != nil {
			return "", fmt.Errorf("patch hunk %d failed: %w", i+1, err)
		}
		lineOffset += len(hunk.newLines) - len(hunk.oldLines)
	}
	return modified, nil
}

func parseUnifiedPatch(patch string) ([]patchHunk, error) {
	lines := strings.Split(ConvertLineEndings(patch, LineEndingLF), "\n")
	i := 0
	for i < len(lines) && lines[i] == "" {
		i++
	}
	if i >= len(lines) || !strings.HasPrefix(lines[i], "--- ") {
		return nil, fmt.Errorf("patch must start with --- and +++ file headers")
	}
	i++
	if i >= len(lines) || !strings.HasPrefix(lines[i], "+++ ") {
		return nil, fmt.Errorf("patch must start with --- and +++ file headers")
	}
	i++

	var hunks []patchHunk
	for i < len(lines) {
		if lines[i] == "" && i == len(lines)-1 {
			break
		}
		if strings.HasPrefix(lines[i], "--- ") {
			return nil, fmt.Errorf("patch contains multiple files")
		}
		oldStart, oldCount, newCount, err := parseHunkHeader(lines[i])
		if err != nil {
			return nil, fmt.Errorf("invalid patch hunk %d: %w", len(hunks)+1, err)
		}
		i++

		hunk := patchHunk{oldStart: oldStart}
		oldSeen, newSeen := 0, 0
		for oldSeen < oldCount || newSeen < newCount {
			if i >= len(lines) {
				return nil, fmt.Errorf("invalid patch hunk %d: unexpected end", len(hunks)+1)
			}
			line := lines[i]
			i++
			if line == `\ No newline at end of file` {
				continue
			}
			if line == "" {
				return nil, fmt.Errorf("invalid patch hunk %d: unprefixed line", len(hunks)+1)
			}
			switch line[0] {
			case ' ':
				hunk.oldLines = append(hunk.oldLines, line[1:])
				hunk.newLines = append(hunk.newLines, line[1:])
				oldSeen++
				newSeen++
			case '-':
				hunk.oldLines = append(hunk.oldLines, line[1:])
				oldSeen++
			case '+':
				hunk.newLines = append(hunk.newLines, line[1:])
				newSeen++
			default:
				return nil, fmt.Errorf("invalid patch hunk %d: line lacks a diff prefix", len(hunks)+1)
			}
			if oldSeen > oldCount || newSeen > newCount {
				return nil, fmt.Errorf("invalid patch hunk %d: line counts exceed header", len(hunks)+1)
			}
		}
		if i < len(lines) && lines[i] == `\ No newline at end of file` {
			i++
		}
		hunks = append(hunks, hunk)
	}
	if len(hunks) == 0 {
		return nil, fmt.Errorf("patch contains no hunks")
	}
	return hunks, nil
}

func parseHunkHeader(header string) (oldStart, oldCount, newCount int, err error) {
	fields := strings.Fields(header)
	if len(fields) < 4 || fields[0] != "@@" || fields[3] != "@@" {
		return 0, 0, 0, fmt.Errorf("expected @@ -old +new @@ header")
	}
	oldStart, oldCount, err = parseHunkRange(fields[1], '-')
	if err != nil {
		return 0, 0, 0, err
	}
	_, newCount, err = parseHunkRange(fields[2], '+')
	return oldStart, oldCount, newCount, err
}

func parseHunkRange(value string, prefix byte) (start, count int, err error) {
	if len(value) < 2 || value[0] != prefix {
		return 0, 0, fmt.Errorf("invalid range %q", value)
	}
	parts := strings.SplitN(value[1:], ",", 2)
	start, err = strconv.Atoi(parts[0])
	if err != nil || start < 0 {
		return 0, 0, fmt.Errorf("invalid range %q", value)
	}
	count = 1
	if len(parts) == 2 {
		count, err = strconv.Atoi(parts[1])
		if err != nil || count < 0 {
			return 0, 0, fmt.Errorf("invalid range %q", value)
		}
	}
	return start, count, nil
}

func insertPatchLines(content, newText string, line int) (string, error) {
	lines := strings.Split(content, "\n")
	if content == "" {
		lines = nil
	}
	if line < 0 || line > len(lines) {
		return "", fmt.Errorf("insertion line %d is outside the file", line)
	}
	newLines := strings.Split(newText, "\n")
	result := make([]string, 0, len(lines)+len(newLines))
	result = append(result, lines[:line]...)
	result = append(result, newLines...)
	result = append(result, lines[line:]...)
	return strings.Join(result, "\n"), nil
}
