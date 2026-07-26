// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/filesystem"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleDirectoryTree returns a recursive tree view of files and directories as JSON.
func (h *Handler) HandleDirectoryTree(ctx context.Context, req *mcp.CallToolRequest, input DirectoryTreeInput) (*mcp.CallToolResult, DirectoryTreeOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, DirectoryTreeOutput{}, nil
	}
	stat, err := os.Stat(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to access path: %v", err)), DirectoryTreeOutput{}, nil
	}
	if !stat.IsDir() {
		return errorResult(ErrPathMustBeDirectory.Error()), DirectoryTreeOutput{}, nil
	}
	resolvedDirs := h.ResolvedAllowedDirs()
	tree, err := buildTree(ctx, v.Path, input.ExcludePatterns, resolvedDirs)
	if err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			return errorResult("operation cancelled"), DirectoryTreeOutput{}, nil
		}
		return errorResult(fmt.Sprintf("failed to build directory tree: %v", err)), DirectoryTreeOutput{}, nil
	}
	jsonBytes, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return errorResult(fmt.Sprintf("failed to marshal tree to JSON: %v", err)), DirectoryTreeOutput{}, nil
	}
	output := DirectoryTreeOutput{Tree: string(jsonBytes)}
	return &mcp.CallToolResult{}, output, nil
}

// buildTree nests the flat walk: levels[d] takes entries at depth d+1.
func buildTree(ctx context.Context, dirPath string, excludePatterns []string, allowedDirs []string) ([]TreeEntry, error) {
	top := &[]TreeEntry{}
	levels := []*[]TreeEntry{top}
	opts := filesystem.Options{
		AllowedDirs: allowedDirs,
		OnError: func(_ string, depth int, err error) error {
			if depth == 0 {
				return err
			}
			// Drop an unreadable directory, as the old traversal did.
			if parent := levels[depth-1]; len(*parent) > 0 {
				*parent = (*parent)[:len(*parent)-1]
			}
			return nil
		},
	}
	err := filesystem.Walk(ctx, dirPath, opts, func(e filesystem.Entry) (filesystem.Action, error) {
		if shouldExclude(e.Name(), excludePatterns) {
			if e.IsDir() {
				return filesystem.SkipDir, nil
			}
			return filesystem.Continue, nil
		}
		levels = levels[:e.Depth]
		parent := levels[e.Depth-1]
		treeEntry := TreeEntry{Name: e.Name(), Type: "file"}
		if e.IsDir() {
			treeEntry.Type = "directory"
			treeEntry.Children = &[]TreeEntry{}
			levels = append(levels, treeEntry.Children)
		}
		*parent = append(*parent, treeEntry)
		return filesystem.Continue, nil
	})
	if err != nil {
		return nil, err
	}
	return *top, nil
}

// shouldExclude checks if a name matches any of the exclude patterns
func shouldExclude(name string, patterns []string) bool {
	for _, pattern := range patterns {
		// Try exact match first
		if name == pattern {
			return true
		}

		// Try glob pattern match
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}

		// For patterns without wildcards, also try as substring/prefix
		// This mimics the JS behavior for patterns like "node_modules"
		if !containsGlobChars(pattern) {
			if strings.Contains(name, pattern) {
				return true
			}
		}
	}
	return false
}

// containsGlobChars checks if pattern contains glob metacharacters
func containsGlobChars(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}
