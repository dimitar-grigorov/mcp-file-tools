package handler

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultMaxMatches = 1000
	maxWorkers        = 0 // 0 means use runtime.NumCPU()
	binaryCheckSize   = 1024
)

// HandleGrep searches for a pattern in files with encoding support.
func (h *Handler) HandleGrep(ctx context.Context, req *mcp.CallToolRequest, input GrepInput) (*mcp.CallToolResult, GrepOutput, error) {
	// Validate input
	if input.Pattern == "" {
		return errorResult("pattern is required"), GrepOutput{}, nil
	}
	if len(input.Paths) == 0 {
		return errorResult("paths is required"), GrepOutput{}, nil
	}

	// Compile regex
	re, err := compilePattern(input.Pattern, input.CaseSensitive)
	if err != nil {
		return errorResult(fmt.Sprintf("invalid regex pattern: %v", err)), GrepOutput{}, nil
	}

	// Set defaults
	maxMatches := input.MaxMatches
	if maxMatches <= 0 {
		maxMatches = defaultMaxMatches
	}

	// Collect files to search
	files, err := h.collectFiles(input.Paths, input.Include, input.Exclude)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to collect files: %v", err)), GrepOutput{}, nil
	}

	if len(files) == 0 {
		return &mcp.CallToolResult{}, GrepOutput{
			Matches:       []GrepMatch{},
			FilesSearched: 0,
		}, nil
	}

	// Search files concurrently
	matches, filesMatched, truncated := h.searchFiles(ctx, files, re, input, maxMatches)

	return &mcp.CallToolResult{}, GrepOutput{
		Matches:       matches,
		TotalMatches:  len(matches),
		FilesSearched: len(files),
		FilesMatched:  filesMatched,
		Truncated:     truncated,
	}, nil
}

// compilePattern compiles the regex pattern with optional case sensitivity.
func compilePattern(pattern string, caseSensitive *bool) (*regexp.Regexp, error) {
	// Default to case-sensitive
	if caseSensitive != nil && !*caseSensitive {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

// collectFiles gathers all files to search from the given paths.
func (h *Handler) collectFiles(paths []string, include, exclude string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	for _, path := range paths {
		// Validate path
		v := h.ValidatePath(path)
		if !v.Ok() {
			continue // Skip invalid paths
		}

		info, err := os.Stat(v.Path)
		if err != nil {
			continue // Skip non-existent paths
		}

		if info.IsDir() {
			// Walk directory
			err := filepath.WalkDir(v.Path, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil // Skip errors, continue walking
				}
				if d.IsDir() {
					return nil
				}
				if shouldIncludeFile(p, include, exclude) && !seen[p] {
					seen[p] = true
					files = append(files, p)
				}
				return nil
			})
			if err != nil {
				continue
			}
		} else {
			// Single file
			if shouldIncludeFile(v.Path, include, exclude) && !seen[v.Path] {
				seen[v.Path] = true
				files = append(files, v.Path)
			}
		}
	}

	return files, nil
}

// shouldIncludeFile checks if a file matches include/exclude patterns.
func shouldIncludeFile(path string, include, exclude string) bool {
	base := filepath.Base(path)

	// Check exclude first
	if exclude != "" {
		matched, _ := filepath.Match(exclude, base)
		if matched {
			return false
		}
	}

	// Check include
	if include != "" {
		matched, _ := filepath.Match(include, base)
		return matched
	}

	return true
}

// searchFiles searches all files concurrently using a worker pool.
func (h *Handler) searchFiles(ctx context.Context, files []string, re *regexp.Regexp, input GrepInput, maxMatches int) ([]GrepMatch, int, bool) {
	numWorkers := maxWorkers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	if numWorkers > len(files) {
		numWorkers = len(files)
	}

	// Channels
	jobs := make(chan string, len(files))
	results := make(chan []GrepMatch, len(files))

	// Track files with matches
	var filesMatched int
	var filesMatchedMu sync.Mutex

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				matches := searchSingleFile(path, re, input)
				if len(matches) > 0 {
					filesMatchedMu.Lock()
					filesMatched++
					filesMatchedMu.Unlock()
					results <- matches
				} else {
					results <- nil
				}
			}
		}()
	}

	// Send jobs
	for _, file := range files {
		jobs <- file
	}
	close(jobs)

	// Wait for workers and close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allMatches []GrepMatch
	truncated := false

	for fileMatches := range results {
		if fileMatches == nil {
			continue
		}
		for _, m := range fileMatches {
			if len(allMatches) >= maxMatches {
				truncated = true
				break
			}
			allMatches = append(allMatches, m)
		}
		if truncated {
			break
		}
	}

	return allMatches, filesMatched, truncated
}

// searchSingleFile searches for matches in a single file.
func searchSingleFile(path string, re *regexp.Regexp, input GrepInput) []GrepMatch {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	// Skip binary files
	if isBinaryFile(data) {
		return nil
	}

	// Detect and decode encoding
	content, detectedEncoding := decodeFileContent(data, input.Encoding)
	if content == "" {
		return nil
	}

	// Split into lines
	lines := strings.Split(content, "\n")

	// Find matches
	var matches []GrepMatch
	for lineNum, line := range lines {
		loc := re.FindStringIndex(line)
		if loc == nil {
			continue
		}

		match := GrepMatch{
			Path:     path,
			Line:     lineNum + 1, // 1-indexed
			Column:   loc[0] + 1,  // 1-indexed
			Text:     line,
			Encoding: detectedEncoding,
		}

		// Add context
		if input.ContextBefore > 0 {
			match.Before = getContextBefore(lines, lineNum, input.ContextBefore)
		}
		if input.ContextAfter > 0 {
			match.After = getContextAfter(lines, lineNum, input.ContextAfter)
		}

		matches = append(matches, match)
	}

	return matches
}

// isBinaryFile checks if the data appears to be binary.
func isBinaryFile(data []byte) bool {
	checkSize := binaryCheckSize
	if len(data) < checkSize {
		checkSize = len(data)
	}

	for i := 0; i < checkSize; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

// decodeFileContent decodes file data to UTF-8 string.
func decodeFileContent(data []byte, forcedEncoding string) (string, string) {
	var encodingName string

	if forcedEncoding != "" {
		encodingName = strings.ToLower(forcedEncoding)
	} else {
		// Auto-detect encoding
		detection, _ := encoding.DetectSample(data)
		if detection.Charset != "" {
			encodingName = detection.Charset
		} else {
			encodingName = "utf-8"
		}
	}

	// Decode to UTF-8
	if encoding.IsUTF8(encodingName) {
		return string(data), encodingName
	}

	enc, ok := encoding.Get(encodingName)
	if !ok {
		// Fallback to treating as UTF-8
		return string(data), "utf-8"
	}

	decoder := enc.NewDecoder()
	decoded, err := decoder.Bytes(data)
	if err != nil {
		return string(data), "utf-8"
	}

	return string(decoded), encodingName
}

// getContextBefore returns N lines before the given line index.
func getContextBefore(lines []string, lineIdx, count int) []string {
	start := lineIdx - count
	if start < 0 {
		start = 0
	}
	if start >= lineIdx {
		return nil
	}
	return lines[start:lineIdx]
}

// getContextAfter returns N lines after the given line index.
func getContextAfter(lines []string, lineIdx, count int) []string {
	end := lineIdx + count + 1
	if end > len(lines) {
		end = len(lines)
	}
	if lineIdx+1 >= end {
		return nil
	}
	return lines[lineIdx+1 : end]
}
