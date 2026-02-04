package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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
	results, err := searchFiles(ctx, v.Path, input.Pattern, input.ExcludePatterns, h.GetAllowedDirectories())
	if err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			return errorResult("search cancelled"), SearchFilesOutput{}, nil
		}
		return errorResult("search failed: " + err.Error()), SearchFilesOutput{}, nil
	}
	return &mcp.CallToolResult{}, SearchFilesOutput{Files: results}, nil
}

// searchFiles recursively searches for files matching the pattern
func searchFiles(ctx context.Context, rootPath, pattern string, excludePatterns, allowedDirs []string) ([]string, error) {
	var results []string
	err := filepath.Walk(rootPath, func(fullPath string, info os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err != nil {
			return nil
		}
		if info.IsDir() && fullPath != rootPath {
			if !security.IsPathSafe(fullPath, allowedDirs) {
				return filepath.SkipDir
			}
		}
		relativePath, err := filepath.Rel(rootPath, fullPath)
		if err != nil {
			return nil
		}
		if relativePath == "." {
			return nil
		}
		relativePathNorm := filepath.ToSlash(relativePath)
		if shouldExcludePath(relativePathNorm, excludePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if matchGlobPattern(relativePathNorm, pattern) {
			results = append(results, fullPath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if results == nil {
		results = []string{}
	}
	return results, nil
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
