package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
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

// buildTree recursively builds a tree of directory entries
func buildTree(ctx context.Context, dirPath string, excludePatterns []string, allowedDirs []string) ([]TreeEntry, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	var result []TreeEntry
	for _, entry := range entries {
		name := entry.Name()
		if shouldExclude(name, excludePatterns) {
			continue
		}
		treeEntry := TreeEntry{Name: name}
		if entry.IsDir() {
			treeEntry.Type = "directory"
			childPath := filepath.Join(dirPath, name)
			if !security.IsPathSafeResolved(childPath, allowedDirs) {
				continue
			}
			children, err := buildTree(ctx, childPath, excludePatterns, allowedDirs)
			if err != nil {
				if err == context.Canceled || err == context.DeadlineExceeded {
					return nil, err
				}
				continue
			}
			treeEntry.Children = &children
		} else {
			treeEntry.Type = "file"
		}
		result = append(result, treeEntry)
	}
	if result == nil {
		result = []TreeEntry{}
	}
	return result, nil
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
