package handler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/filesystem"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/workpool"
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
	allowedDirs := h.ResolvedAllowedDirs()
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
			opts := filesystem.Options{
				AllowedDirs: allowedDirs,
				OnError: func(p string, _ int, err error) error {
					slog.Debug("skipping path due to error", "path", p, "error", err)
					return nil
				},
			}
			_ = filesystem.Walk(ctx, v.Path, opts, func(e filesystem.Entry) (filesystem.Action, error) {
				if e.IsDir() {
					return filesystem.Continue, nil
				}
				if shouldIncludeFile(e.Path, include, exclude) && !seen[e.Path] {
					seen[e.Path] = true
					files = append(files, e.Path)
				}
				return filesystem.Continue, nil
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

// searchFiles searches all files with bounded concurrency, committing results in file
// order so the truncated set is the same on every run, and stopping once maxMatches is hit.
func (h *Handler) searchFiles(ctx context.Context, files []string, re *regexp.Regexp, input GrepInput, maxMatches int, maxFileSize int64) ([]GrepMatch, int, bool) {
	var allMatches []GrepMatch
	filesMatched := 0
	truncated := false
	// One more than the cap, so a single file overflowing still reports truncation.
	perFileLimit := maxMatches + 1

	workpool.RunOrdered(ctx, files, workpool.Options{},
		func(ctx context.Context, _ int, path string) []GrepMatch {
			if ctx.Err() != nil {
				return nil
			}
			return searchSingleFile(path, re, input, maxFileSize, perFileLimit)
		},
		func(_ int, fileMatches []GrepMatch) bool {
			if len(fileMatches) > 0 {
				filesMatched++
			}
			for _, m := range fileMatches {
				if len(allMatches) >= maxMatches {
					truncated = true
					return false
				}
				allMatches = append(allMatches, m)
			}
			return true
		})

	return allMatches, filesMatched, truncated
}

// searchSingleFile searches for matches in a single file, stopping at limit matches.
func searchSingleFile(path string, re *regexp.Regexp, input GrepInput, maxFileSize int64, limit int) []GrepMatch {
	// Check file size - warn if large file will be loaded to memory
	if info, err := os.Stat(path); err == nil && info.Size() > maxFileSize {
		slog.Warn("loading large file into memory", "path", path, "size", info.Size(), "threshold", maxFileSize)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	// Decode first: UTF-16 text is full of NUL bytes, so classify the decoded text.
	content, detectedEncoding := decodeFileContent(data, input.Encoding)
	if content == "" || isBinaryText(content) {
		return nil
	}
	lines := splitGrepLines(content)
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
		if len(matches) >= limit {
			break
		}
	}
	return matches
}

// splitGrepLines splits on CRLF, lone CR and LF so no line keeps a trailing \r.
func splitGrepLines(content string) []string {
	if strings.ContainsRune(content, '\r') {
		content = strings.ReplaceAll(content, "\r\n", "\n")
		content = strings.ReplaceAll(content, "\r", "\n")
	}
	return strings.Split(content, "\n")
}

// isBinaryText classifies decoded text: a NUL rune, or dense control characters.
func isBinaryText(content string) bool {
	if !utf8.ValidString(content) {
		return true
	}
	controlCount, runeCount := 0, 0
	for i, r := range content {
		if i >= binaryCheckSize {
			break
		}
		runeCount++
		if r == 0 {
			return true
		}
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			controlCount++
		}
	}
	return runeCount > 0 && controlCount*10 >= runeCount
}

// detectGrepEncoding resolves the encoding to search with. Valid UTF-8 bytes beat an
// untrusted guess: chardet mislabels short mixed-script UTF-8 as a single-byte charset.
func detectGrepEncoding(data []byte) string {
	detection, trusted := encoding.DetectSample(data)
	if detection.Charset == "" {
		return "utf-8"
	}
	if trusted || !utf8.Valid(data) {
		return detection.Charset
	}
	return "utf-8"
}

// decodeFileContent decodes file data to UTF-8 string.
func decodeFileContent(data []byte, forcedEncoding string) (string, string) {
	var encodingName string
	if forcedEncoding != "" {
		encodingName, _ = encoding.Canonical(forcedEncoding)
		if encodingName == "" {
			encodingName = strings.ToLower(forcedEncoding)
		}
	} else {
		encodingName = detectGrepEncoding(data)
	}
	if encoding.IsUTF8(encodingName) {
		return trimBOM(string(data)), encodingName
	}
	enc, ok := encoding.Get(encodingName)
	if !ok {
		return trimBOM(string(data)), "utf-8"
	}
	decoder := enc.NewDecoder()
	decoded, err := decoder.Bytes(data)
	if err != nil {
		return trimBOM(string(data)), "utf-8"
	}
	return trimBOM(string(decoded)), encodingName
}

// trimBOM drops a leading BOM so it doesn't shift line 1 columns or break ^ anchors.
func trimBOM(content string) string {
	if r, size := utf8.DecodeRuneInString(content); r == 0xFEFF {
		return content[size:]
	}
	return content
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
