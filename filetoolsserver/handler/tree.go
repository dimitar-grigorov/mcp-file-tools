package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
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
		maxDepth:     input.MaxDepth,
		dirsOnly:     input.DirsOnly,
		exclude:      input.Exclude,
		showEncoding: input.ShowEncoding,
		allowedDirs:  h.ResolvedAllowedDirs(),
		fileCount:    0,
		dirCount:     0,
		truncated:    false,
	}
	var sb strings.Builder
	buildCompactTree(ctx, &sb, v.Path, 0, state)
	return &mcp.CallToolResult{}, TreeOutput{
		Tree:      sb.String(),
		FileCount: state.fileCount,
		DirCount:  state.dirCount,
		Truncated: state.truncated,
	}, nil
}

type treeState struct {
	maxFiles     int
	maxDepth     int
	dirsOnly     bool
	exclude      []string
	showEncoding bool
	allowedDirs  []string
	fileCount    int
	dirCount     int
	truncated    bool
}

func (s *treeState) totalCount() int {
	return s.fileCount + s.dirCount
}

func buildCompactTree(ctx context.Context, sb *strings.Builder, dirPath string, depth int, state *treeState) {
	select {
	case <-ctx.Done():
		state.truncated = true
		return
	default:
	}
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
			subPath := filepath.Join(dirPath, name)
			if !security.IsPathSafeResolved(subPath, state.allowedDirs) {
				continue
			}
			state.dirCount++
			sb.WriteString(indent)
			sb.WriteString(name)
			sb.WriteString("/\n")
			buildCompactTree(ctx, sb, subPath, depth+1, state)
		} else if !state.dirsOnly {
			state.fileCount++
			sb.WriteString(indent)
			sb.WriteString(name)
			if state.showEncoding {
				filePath := filepath.Join(dirPath, name)
				if enc := detectFileEncoding(filePath); enc != "" {
					sb.WriteString("  [")
					sb.WriteString(enc)
					sb.WriteString("]")
				}
			}
			sb.WriteString("\n")
		}
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
