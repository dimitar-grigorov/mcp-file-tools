package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleDirectoryTree returns a recursive tree view of files and directories as JSON.
func (h *Handler) HandleDirectoryTree(ctx context.Context, req *mcp.CallToolRequest, input DirectoryTreeInput) (*mcp.CallToolResult, DirectoryTreeOutput, error) {
	// Validate input
	if input.Path == "" {
		return errorResult("path is required and must be a non-empty string"), DirectoryTreeOutput{}, nil
	}

	// Validate path against allowed directories
	validatedPath, err := h.validatePath(input.Path)
	if err != nil {
		return errorResult(err.Error()), DirectoryTreeOutput{}, nil
	}

	// Check that the path is a directory
	stat, err := os.Stat(validatedPath)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to access path: %v", err)), DirectoryTreeOutput{}, nil
	}
	if !stat.IsDir() {
		return errorResult("path must be a directory"), DirectoryTreeOutput{}, nil
	}

	// Build the tree
	tree, err := buildTree(validatedPath, input.ExcludePatterns)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to build directory tree: %v", err)), DirectoryTreeOutput{}, nil
	}

	// Marshal to JSON with 2-space indentation for readability
	jsonBytes, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return errorResult(fmt.Sprintf("failed to marshal tree to JSON: %v", err)), DirectoryTreeOutput{}, nil
	}

	output := DirectoryTreeOutput{Tree: string(jsonBytes)}
	return &mcp.CallToolResult{}, output, nil
}

// buildTree recursively builds a tree of directory entries
func buildTree(dirPath string, excludePatterns []string) ([]TreeEntry, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var result []TreeEntry

	for _, entry := range entries {
		name := entry.Name()

		// Check if this entry should be excluded
		if shouldExclude(name, excludePatterns) {
			continue
		}

		treeEntry := TreeEntry{
			Name: name,
		}

		if entry.IsDir() {
			treeEntry.Type = "directory"

			// Recursively build subtree
			childPath := filepath.Join(dirPath, name)
			children, err := buildTree(childPath, excludePatterns)
			if err != nil {
				// Skip directories we can't read (permissions, etc.)
				continue
			}
			treeEntry.Children = &children
		} else {
			treeEntry.Type = "file"
			// Files don't have children (nil pointer = omitted in JSON)
		}

		result = append(result, treeEntry)
	}

	// Ensure we return empty array instead of nil for directories
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
