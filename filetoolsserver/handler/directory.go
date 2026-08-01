// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleListDirectory lists files in a directory with optional pattern filtering
func (h *Handler) HandleListDirectory(ctx context.Context, req *mcp.CallToolRequest, input ListDirectoryInput) (*mcp.CallToolResult, ListDirectoryOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, ListDirectoryOutput{}, nil
	}

	pattern := "*"
	if input.Pattern != "" {
		pattern = input.Pattern
	}
	sortBy, err := resolveSortBy(input.SortBy)
	if err != nil {
		return errorResult(err.Error()), ListDirectoryOutput{}, nil
	}

	entries, err := os.ReadDir(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read directory: %v", err)), ListDirectoryOutput{}, nil
	}

	files, err := listDirectoryEntries(entries, pattern, sortBy, input.Reverse, nil)
	if err != nil {
		return errorResult(fmt.Sprintf("invalid pattern: %v", err)), ListDirectoryOutput{}, nil
	}

	return &mcp.CallToolResult{}, ListDirectoryOutput{Files: files}, nil
}

// listDirectoryEntries filters and orders one directory. Name sorting makes no
// Info() call. stat is nil outside tests.
func listDirectoryEntries(entries []fs.DirEntry, pattern, sortBy string, reverse bool, stat statFunc) ([]string, error) {
	if stat == nil {
		stat = statEntry
	}
	readKeys := needsStat(sortBy)

	matched := make([]sortEntry, 0, len(entries))
	for _, entry := range entries {
		ok, err := filepath.Match(pattern, entry.Name())
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		prefix := ""
		if entry.IsDir() {
			prefix = "[DIR] "
		}
		e := sortEntry{key: entry.Name(), value: prefix + entry.Name()}
		if readKeys {
			e.mtime, e.size = stat(entry)
		}
		matched = append(matched, e)
	}
	sortEntries(matched, sortBy, reverse)

	files := make([]string, len(matched))
	for i, e := range matched {
		files[i] = e.value
	}
	return files, nil
}
