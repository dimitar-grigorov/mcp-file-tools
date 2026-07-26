// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/filesystem"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultMaxResults = 10000

// HandleSearchFiles recursively searches for files matching a glob pattern.
func (h *Handler) HandleSearchFiles(ctx context.Context, req *mcp.CallToolRequest, input SearchFilesInput) (*mcp.CallToolResult, SearchFilesOutput, error) {
	if input.Pattern == "" {
		return errorResult(ErrPatternRequired.Error()), SearchFilesOutput{}, nil
	}
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, SearchFilesOutput{}, nil
	}
	stat, err := os.Stat(v.Path)
	if err != nil {
		return errorResult("failed to access path: " + err.Error()), SearchFilesOutput{}, nil
	}
	if !stat.IsDir() {
		return errorResult(ErrPathMustBeDirectory.Error()), SearchFilesOutput{}, nil
	}
	maxResults := input.MaxResults
	if maxResults <= 0 {
		maxResults = defaultMaxResults
	}
	results, truncated, err := searchFiles(ctx, v.Path, input.Pattern, input.ExcludePatterns, h.ResolvedAllowedDirs(), maxResults)
	if err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			return errorResult("search cancelled"), SearchFilesOutput{}, nil
		}
		return errorResult("search failed: " + err.Error()), SearchFilesOutput{}, nil
	}
	return &mcp.CallToolResult{}, SearchFilesOutput{Files: results, Truncated: truncated}, nil
}

// searchFiles recursively searches for files matching the pattern
func searchFiles(ctx context.Context, rootPath, pattern string, excludePatterns, allowedDirs []string, maxResults int) ([]string, bool, error) {
	var results []string
	truncated := false
	opts := filesystem.Options{
		AllowedDirs: allowedDirs,
		OnError: func(path string, _ int, err error) error {
			slog.Debug("skipping path due to error", "path", path, "error", err)
			return nil
		},
	}
	err := filesystem.Walk(ctx, rootPath, opts, func(e filesystem.Entry) (filesystem.Action, error) {
		if shouldExcludePath(e.RelPath, excludePatterns) {
			if e.IsDir() {
				return filesystem.SkipDir, nil
			}
			return filesystem.Continue, nil
		}
		if matchGlobPattern(e.RelPath, pattern) {
			results = append(results, e.Path)
			if len(results) >= maxResults {
				truncated = true
				return filesystem.Stop, nil
			}
		}
		return filesystem.Continue, nil
	})
	if err != nil {
		return nil, false, err
	}
	if results == nil {
		results = []string{}
	}
	return results, truncated, nil
}

// matchGlobPattern matches a path against a glob pattern, supporting ** for recursive matching
func matchGlobPattern(path, pattern string) bool {
	// Normalize pattern to use forward slashes
	pattern = filepath.ToSlash(pattern)

	// Handle ** patterns (recursive glob)
	if strings.Contains(pattern, "**") {
		return matchDoubleStarPattern(path, pattern)
	}

	// Standard glob match using filepath.Match
	matched, err := filepath.Match(pattern, path)
	if err == nil && matched {
		return true
	}

	// Also try matching just the filename for patterns without path separators
	if !strings.Contains(pattern, "/") {
		filename := filepath.Base(path)
		matched, err = filepath.Match(pattern, filename)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// matchDoubleStarPattern handles patterns containing **
func matchDoubleStarPattern(path, pattern string) bool {
	// Split pattern into parts around **
	parts := strings.Split(pattern, "**")

	if len(parts) == 2 {
		prefix := strings.TrimSuffix(parts[0], "/")
		suffix := strings.TrimPrefix(parts[1], "/")

		// Pattern like "**/*.ext" - match suffix against any subpath
		if prefix == "" {
			// Try matching the suffix against the path or any part of it
			if suffix != "" {
				// Match the suffix pattern against the filename or path ending
				return matchSuffix(path, suffix)
			}
			// Pattern is just "**" - matches everything
			return true
		}

		// Pattern like "dir/**" - match prefix then anything
		if suffix == "" {
			return strings.HasPrefix(path, prefix+"/") || path == prefix
		}

		// Pattern like "dir/**/file.ext"
		if strings.HasPrefix(path, prefix+"/") || prefix == "" {
			remaining := path
			if prefix != "" {
				remaining = strings.TrimPrefix(path, prefix+"/")
			}
			return matchSuffix(remaining, suffix)
		}
	}

	return false
}

// matchSuffix checks if the path ends with a pattern match
func matchSuffix(path, suffixPattern string) bool {
	// Try matching the entire path
	matched, err := filepath.Match(suffixPattern, path)
	if err == nil && matched {
		return true
	}

	// Try matching just the filename
	filename := filepath.Base(path)
	matched, err = filepath.Match(suffixPattern, filename)
	if err == nil && matched {
		return true
	}

	// Try matching the path with the suffix pattern at any depth
	parts := strings.Split(path, "/")
	for i := range parts {
		subpath := strings.Join(parts[i:], "/")
		matched, err = filepath.Match(suffixPattern, subpath)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// shouldExcludePath checks if a path matches any of the exclude patterns
func shouldExcludePath(path string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(pattern)

		// Try glob match
		if matchGlobPattern(path, pattern) {
			return true
		}

		// Also check if the path contains the pattern as a directory component
		if !containsGlobChars(pattern) {
			pathParts := strings.Split(path, "/")
			for _, part := range pathParts {
				if part == pattern {
					return true
				}
			}
		}
	}
	return false
}
