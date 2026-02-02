package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

	// Set defaults
	maxFiles := input.MaxFiles
	if maxFiles == 0 {
		maxFiles = 1000
	}

	state := &treeState{
		maxFiles:  maxFiles,
		maxDepth:  input.MaxDepth,
		dirsOnly:  input.DirsOnly,
		exclude:   input.Exclude,
		fileCount: 0,
		dirCount:  0,
		truncated: false,
	}

	var sb strings.Builder
	buildCompactTree(&sb, v.Path, 0, state)

	return &mcp.CallToolResult{}, TreeOutput{
		Tree:      sb.String(),
		FileCount: state.fileCount,
		DirCount:  state.dirCount,
		Truncated: state.truncated,
	}, nil
}

type treeState struct {
	maxFiles  int
	maxDepth  int
	dirsOnly  bool
	exclude   []string
	fileCount int
	dirCount  int
	truncated bool
}

func (s *treeState) totalCount() int {
	return s.fileCount + s.dirCount
}

func buildCompactTree(sb *strings.Builder, dirPath string, depth int, state *treeState) {
	if state.truncated {
		return
	}
	if state.maxDepth > 0 && depth >= state.maxDepth {
		return
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return
	}

	indent := strings.Repeat("  ", depth)

	for _, entry := range entries {
		if state.totalCount() >= state.maxFiles {
			state.truncated = true
			return
		}

		name := entry.Name()
		if shouldExcludeTree(name, state.exclude) {
			continue
		}

		if entry.IsDir() {
			state.dirCount++
			sb.WriteString(indent)
			sb.WriteString(name)
			sb.WriteString("/\n")
			buildCompactTree(sb, filepath.Join(dirPath, name), depth+1, state)
		} else if !state.dirsOnly {
			state.fileCount++
			sb.WriteString(indent)
			sb.WriteString(name)
			sb.WriteString("\n")
		}
	}
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
