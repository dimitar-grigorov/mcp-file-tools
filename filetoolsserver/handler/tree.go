// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/filesystem"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultMaxFiles = 1000

// HandleTree returns a compact indented tree view optimized for AI consumption.
// Uses ~70-80% fewer tokens than JSON format.
func (h *Handler) HandleTree(ctx context.Context, req *mcp.CallToolRequest, input TreeInput) (*mcp.CallToolResult, TreeOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, TreeOutput{}, nil
	}
	stat, err := os.Stat(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to access path: %v", err)), TreeOutput{}, nil
	}
	if !stat.IsDir() {
		return errorResult(ErrPathMustBeDirectory.Error()), TreeOutput{}, nil
	}
	maxFiles := input.MaxFiles
	if maxFiles == 0 {
		maxFiles = defaultMaxFiles
	}
	state := &treeState{
		maxFiles:     maxFiles,
		dirsOnly:     input.DirsOnly,
		exclude:      input.Exclude,
		showEncoding: input.ShowEncoding,
		fileCount:    0,
		dirCount:     0,
		truncated:    false,
	}
	var sb strings.Builder
	opts := filesystem.Options{
		AllowedDirs:      h.ResolvedAllowedDirs(),
		MaxDepth:         input.MaxDepth,
		RespectGitignore: gitignoreDefault(input.RespectGitignore),
	}
	if err := filesystem.Walk(ctx, v.Path, opts, state.visit(&sb)); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			state.truncated = true
		}
	}
	return &mcp.CallToolResult{}, TreeOutput{
		Tree:      sb.String(),
		FileCount: state.fileCount,
		DirCount:  state.dirCount,
		Truncated: state.truncated,
	}, nil
}

type treeState struct {
	maxFiles     int
	dirsOnly     bool
	exclude      []string
	showEncoding bool
	fileCount    int
	dirCount     int
	truncated    bool
}

func (s *treeState) totalCount() int {
	return s.fileCount + s.dirCount
}

// visit renders one entry into the indented tree.
func (s *treeState) visit(sb *strings.Builder) filesystem.Visitor {
	return func(e filesystem.Entry) (filesystem.Action, error) {
		if s.totalCount() >= s.maxFiles {
			s.truncated = true
			return filesystem.Stop, nil
		}
		name := e.Name()
		if shouldExcludeTree(name, s.exclude) {
			if e.IsDir() {
				return filesystem.SkipDir, nil
			}
			return filesystem.Continue, nil
		}
		if !e.IsDir() && s.dirsOnly {
			return filesystem.Continue, nil
		}
		sb.WriteString(strings.Repeat("  ", e.Depth-1))
		sb.WriteString(name)
		if e.IsDir() {
			s.dirCount++
			sb.WriteString("/\n")
			return filesystem.Continue, nil
		}
		s.fileCount++
		if s.showEncoding {
			if enc := detectFileEncoding(e.Path); enc != "" {
				sb.WriteString("  [")
				sb.WriteString(enc)
				sb.WriteString("]")
			}
		}
		sb.WriteString("\n")
		return filesystem.Continue, nil
	}
}

// detectFileEncoding returns the detected encoding name for a file, or "" on error.
// Uses sample mode for speed since this is called per-file in tree traversal.
func detectFileEncoding(path string) string {
	result, err := encoding.DetectFromFile(path, "sample")
	if err != nil || result.Confidence < encoding.MinConfidenceThreshold {
		return ""
	}
	return result.Charset
}

func shouldExcludeTree(name string, patterns []string) bool {
	for _, pattern := range patterns {
		if name == pattern {
			return true
		}
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}
