package handler

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultMaxMatches = 1000
	binaryCheckSize   = 8192 // 8KB to catch files with text header but binary payload
)

// HandleGrep searches for a pattern in files with encoding support.
func (h *Handler) HandleGrep(ctx context.Context, req *mcp.CallToolRequest, input GrepInput) (*mcp.CallToolResult, GrepOutput, error) {
	if input.Pattern == "" {
		return errorResult("pattern is required"), GrepOutput{}, nil
	}
	if len(input.Paths) == 0 {
		return errorResult("paths is required"), GrepOutput{}, nil
	}
	re, err := compilePattern(input.Pattern, input.CaseSensitive)
	if err != nil {
		return errorResult(fmt.Sprintf("invalid regex pattern: %v", err)), GrepOutput{}, nil
	}
	maxMatches := input.MaxMatches
	if maxMatches <= 0 {
		maxMatches = defaultMaxMatches
	}
	files := h.collectFiles(ctx, input.Paths, input.Include, input.Exclude)
	if len(files) == 0 {
		return &mcp.CallToolResult{}, GrepOutput{Matches: []GrepMatch{}, FilesSearched: 0}, nil
	}
	matches, filesMatched, truncated := h.searchFiles(ctx, files, re, input, maxMatches, h.config.MemoryThreshold)
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
	if caseSensitive != nil && !*caseSensitive {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

// collectFiles gathers all files to search from the given paths.
func (h *Handler) collectFiles(ctx context.Context, paths []string, include, exclude string) []string {
	var files []string
	seen := make(map[string]bool)
	allowedDirs := h.GetAllowedDirectories()
	for _, path := range paths {
		// Check for cancellation between paths
		select {
		case <-ctx.Done():
			return files
		default:
		}
		v := h.ValidatePath(path)
		if !v.Ok() {
			continue
		}
		info, err := os.Stat(v.Path)
		if err != nil {
			continue
		}
		if info.IsDir() {
			filepath.WalkDir(v.Path, func(p string, d fs.DirEntry, err error) error {
				// Check for cancellation during walk
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				if err != nil {
					slog.Debug("skipping path due to error", "path", p, "error", err)
					return nil
				}
				if d.IsDir() {
					if !security.IsPathSafe(p, allowedDirs) {
						return filepath.SkipDir
					}
					return nil
				}
				if shouldIncludeFile(p, include, exclude) && !seen[p] {
					seen[p] = true
					files = append(files, p)
				}
				return nil
			})
		} else if shouldIncludeFile(v.Path, include, exclude) && !seen[v.Path] {
			seen[v.Path] = true
			files = append(files, v.Path)
		}
	}
	return files
}

// shouldIncludeFile checks if a file matches include/exclude patterns.
// Matches against both full path (with forward slashes) and basename.
func shouldIncludeFile(path string, include, exclude string) bool {
	base := filepath.Base(path)
	normalized := filepath.ToSlash(path)
	if exclude != "" {
		if matchedBase, _ := filepath.Match(exclude, base); matchedBase {
			return false
		}
		if matchedPath, _ := filepath.Match(exclude, normalized); matchedPath {
			return false
		}
	}
	if include != "" {
		if matchedBase, _ := filepath.Match(include, base); matchedBase {
			return true
		}
		if matchedPath, _ := filepath.Match(include, normalized); matchedPath {
			return true
		}
		return false
	}
	return true
}

// searchFiles searches all files concurrently using a worker pool.
func (h *Handler) searchFiles(ctx context.Context, files []string, re *regexp.Regexp, input GrepInput, maxMatches int, maxFileSize int64) ([]GrepMatch, int, bool) {
	numWorkers := runtime.NumCPU()
	if numWorkers > len(files) {
		numWorkers = len(files)
	}
	jobs := make(chan string, len(files))
	results := make(chan []GrepMatch, len(files))
	var filesMatched int
	var mu sync.Mutex
	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				select {
				case <-ctx.Done():
					results <- nil
				default:
					matches := searchSingleFile(path, re, input, maxFileSize)
					if len(matches) > 0 {
						mu.Lock()
						filesMatched++
						mu.Unlock()
					}
					results <- matches
				}
			}
		}()
	}
	// Send all jobs
	for _, file := range files {
		jobs <- file
	}
	close(jobs)
	// Close results when workers done
	go func() {
		wg.Wait()
		close(results)
	}()
	// Collect results
	var allMatches []GrepMatch
	truncated := false
	for fileMatches := range results {
		for _, m := range fileMatches {
			if len(allMatches) >= maxMatches {
				truncated = true
				break
			}
			allMatches = append(allMatches, m)
		}
	}
	return allMatches, filesMatched, truncated
}

// searchSingleFile searches for matches in a single file.
func searchSingleFile(path string, re *regexp.Regexp, input GrepInput, maxFileSize int64) []GrepMatch {
	// Check file size - warn if large file will be loaded to memory
	if info, err := os.Stat(path); err == nil && info.Size() > maxFileSize {
		slog.Warn("loading large file into memory", "path", path, "size", info.Size(), "threshold", maxFileSize)
	}
	data, err := os.ReadFile(path)
	if err != nil || isBinaryFile(data) {
		return nil
	}
	content, detectedEncoding := decodeFileContent(data, input.Encoding)
	if content == "" {
		return nil
	}
	lines := strings.Split(content, "\n")
	var matches []GrepMatch
	for lineNum, line := range lines {
		loc := re.FindStringIndex(line)
		if loc == nil {
			continue
		}
		match := GrepMatch{
			Path:     path,
			Line:     lineNum + 1,
			Column:   loc[0] + 1,
			Text:     line,
			Encoding: detectedEncoding,
		}
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

// isBinaryFile checks if the data appears to be binary (contains null bytes).
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
		detection, _ := encoding.DetectSample(data)
		if detection.Charset != "" {
			encodingName = detection.Charset
		} else {
			encodingName = "utf-8"
		}
	}
	if encoding.IsUTF8(encodingName) {
		return string(data), encodingName
	}
	enc, ok := encoding.Get(encodingName)
	if !ok {
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
