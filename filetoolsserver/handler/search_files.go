// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
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
	sortBy, err := resolveSortBy(input.SortBy)
	if err != nil {
		return errorResult(err.Error()), SearchFilesOutput{}, nil
	}
	results, truncated, err := searchFiles(ctx, v.Path, searchOptions{
		pattern:     input.Pattern,
		exclude:     input.ExcludePatterns,
		allowedDirs: h.ResolvedAllowedDirs(),
		maxResults:  maxResults,
		sortBy:      sortBy,
		reverse:     input.Reverse,
		gitignore:   gitignoreDefault(input.RespectGitignore),
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return errorResult("search cancelled"), SearchFilesOutput{}, nil
		}
		return errorResult("search failed: " + err.Error()), SearchFilesOutput{}, nil
	}
	return &mcp.CallToolResult{}, SearchFilesOutput{Files: results, Truncated: truncated}, nil
}

// searchOptions is the resolved search policy. stat is nil outside tests.
type searchOptions struct {
	pattern     string
	exclude     []string
	allowedDirs []string
	maxResults  int
	sortBy      string
	reverse     bool
	stat        statFunc
	gitignore   bool
}

// searchFiles recursively searches for files matching the pattern. Name keeps the
// walk's early stop; mtime and size rank the whole tree behind a bounded heap,
// since the newest file may be the last one visited.
func searchFiles(ctx context.Context, rootPath string, sOpts searchOptions) ([]string, bool, error) {
	opts := filesystem.Options{
		AllowedDirs:      sOpts.allowedDirs,
		RespectGitignore: sOpts.gitignore,
		OnError: func(path string, _ int, err error) error {
			slog.Debug("skipping path due to error", "path", path, "error", err)
			return nil
		},
	}
	stat := sOpts.stat
	if stat == nil {
		stat = statEntry
	}
	readKeys := needsStat(sOpts.sortBy)

	var capped []sortEntry
	top := newTopN(sOpts.maxResults, sOpts.sortBy, sOpts.reverse)
	truncated := false

	err := filesystem.Walk(ctx, rootPath, opts, func(e filesystem.Entry) (filesystem.Action, error) {
		if shouldExcludePath(e.RelPath, sOpts.exclude) {
			if e.IsDir() {
				return filesystem.SkipDir, nil
			}
			return filesystem.Continue, nil
		}
		if !matchGlobPattern(e.RelPath, sOpts.pattern) {
			return filesystem.Continue, nil
		}
		entry := sortEntry{key: e.Path, value: e.Path}
		if readKeys {
			entry.mtime, entry.size = stat(e.DirEntry)
			top.add(entry)
			return filesystem.Continue, nil
		}
		capped = append(capped, entry)
		if len(capped) >= sOpts.maxResults {
			truncated = true
			return filesystem.Stop, nil
		}
		return filesystem.Continue, nil
	})
	if err != nil {
		return nil, false, err
	}
	if readKeys {
		return top.values(), top.truncated(), nil
	}
	sortEntries(capped, sOpts.sortBy, sOpts.reverse)
	results := make([]string, len(capped))
	for i, entry := range capped {
		results[i] = entry.value
	}
	return results, truncated, nil
}

// matchGlobPattern matches a slash-separated path against a glob pattern.
// A pattern without a separator also matches on the basename alone, so "*.pas"
// finds files at any depth.
func matchGlobPattern(path, pattern string) bool {
	pattern = filepath.ToSlash(pattern)

	if strings.Contains(pattern, "**") {
		return matchDoubleStarPattern(path, pattern)
	}

	if matched, err := filepath.Match(pattern, path); err == nil && matched {
		return true
	}

	if !strings.Contains(pattern, "/") {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		return err == nil && matched
	}

	return false
}

// matchDoubleStarPattern handles a single "**" wildcard, which crosses directories.
// Two or more are not supported and match nothing.
func matchDoubleStarPattern(path, pattern string) bool {
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		return false
	}
	prefix := strings.TrimSuffix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[1], "/")

	switch {
	case prefix == "": // "**" alone, or "**/*.ext"
		return suffix == "" || matchSuffix(path, suffix)
	case suffix == "": // "dir/**"
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	case strings.HasPrefix(path, prefix+"/"): // "dir/**/file.ext"
		return matchSuffix(strings.TrimPrefix(path, prefix+"/"), suffix)
	}
	return false
}

// matchSuffix matches suffixPattern against the whole path, the basename, or any
// trailing run of path segments.
func matchSuffix(path, suffixPattern string) bool {
	if matched, err := filepath.Match(suffixPattern, path); err == nil && matched {
		return true
	}
	if matched, err := filepath.Match(suffixPattern, filepath.Base(path)); err == nil && matched {
		return true
	}

	parts := strings.Split(path, "/")
	for i := range parts {
		if matched, err := filepath.Match(suffixPattern, strings.Join(parts[i:], "/")); err == nil && matched {
			return true
		}
	}
	return false
}

func containsGlobChars(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// shouldExcludePath reports whether path matches any exclude pattern. A literal
// pattern also excludes anything under a directory of that name, so "vendor"
// drops the whole subtree rather than just a file called "vendor".
func shouldExcludePath(path string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(pattern)
		if matchGlobPattern(path, pattern) {
			return true
		}
		if !containsGlobChars(pattern) && slices.Contains(strings.Split(path, "/"), pattern) {
			return true
		}
	}
	return false
}
