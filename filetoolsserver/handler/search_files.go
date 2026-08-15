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

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/filesystem"
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
		// One past the cap, so an exact fit is not reported as truncated.
		if len(capped) > sOpts.maxResults {
			capped = capped[:sOpts.maxResults]
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

// "**" spans any number of segments and "{a,b}" tries each alternative; a pattern
// without a separator matches on the basename alone, so "*.pas" finds files at any depth.
func matchGlobPattern(path, pattern string) bool {
	pattern = strings.TrimSuffix(filepath.ToSlash(pattern), "/")
	if !strings.Contains(pattern, "{") {
		return matchOnePattern(path, pattern)
	}
	for _, p := range expandBraces(pattern) {
		if matchOnePattern(path, p) {
			return true
		}
	}
	return false
}

func matchOnePattern(path, pattern string) bool {
	if !strings.Contains(pattern, "/") && !strings.Contains(pattern, "**") {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		return err == nil && matched
	}
	return matchSegments(strings.Split(path, "/"), strings.Split(pattern, "/"))
}

// expandBraces expands the first {a,b} group and recurses, so "*.{ts,tsx}"
// tries both alternatives. Nesting is not supported; expansion is capped.
func expandBraces(pattern string) []string {
	open := strings.IndexByte(pattern, '{')
	if open < 0 {
		return []string{pattern}
	}
	end := strings.IndexByte(pattern[open:], '}')
	if end < 0 {
		return []string{pattern}
	}
	end += open
	var out []string
	for _, alt := range strings.Split(pattern[open+1:end], ",") {
		out = append(out, expandBraces(pattern[:open]+alt+pattern[end+1:])...)
		if len(out) >= 64 {
			break
		}
	}
	return out
}

// matchSegments: "**" spans any number of path segments, including none.
func matchSegments(path, pat []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			if len(pat) == 1 {
				return true
			}
			for i := 0; i <= len(path); i++ {
				if matchSegments(path[i:], pat[1:]) {
					return true
				}
			}
			return false
		}
		if len(path) == 0 {
			return false
		}
		if matched, err := filepath.Match(pat[0], path[0]); err != nil || !matched {
			return false
		}
		path, pat = path[1:], pat[1:]
	}
	return len(path) == 0
}

func containsGlobChars(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// A literal exclude pattern also excludes anything under a directory of that name,
// so "vendor" drops the whole subtree rather than just a file called "vendor".
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
