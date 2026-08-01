// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

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

	outputModeContent = "content"
	outputModeFiles   = "files_with_matches"
	outputModeCount   = "count"
)

// grepOptions is the resolved per-search policy handed to each worker.
type grepOptions struct {
	re            *regexp.Regexp
	mode          string
	matchesOnly   bool
	contextBefore int
	contextAfter  int
	encoding      string
	maxFileSize   int64
	perFileLimit  int // 0 means no limit: count mode must see the whole file
}

// fileHits is one file's outcome. matches is filled in content mode only;
// count is the number of matching lines, capped at 1 in files_with_matches.
type fileHits struct {
	matches []GrepMatch
	count   int
}

// HandleGrep searches for a pattern in files with encoding support.
func (h *Handler) HandleGrep(ctx context.Context, req *mcp.CallToolRequest, input GrepInput) (*mcp.CallToolResult, GrepOutput, error) {
	if input.Pattern == "" {
		return errorResult("pattern is required"), GrepOutput{}, nil
	}
	if len(input.Paths) == 0 {
		return errorResult("paths is required"), GrepOutput{}, nil
	}
	mode := input.OutputMode
	if mode == "" {
		mode = outputModeContent
	}
	if mode != outputModeContent && mode != outputModeFiles && mode != outputModeCount {
		return errorResult(fmt.Sprintf("invalid outputMode %q: use %q, %q or %q",
			input.OutputMode, outputModeContent, outputModeFiles, outputModeCount)), GrepOutput{}, nil
	}
	re, err := compilePattern(input.Pattern, input.CaseSensitive)
	if err != nil {
		return errorResult(fmt.Sprintf("invalid regex pattern: %v", err)), GrepOutput{}, nil
	}
	maxMatches := input.MaxMatches
	if maxMatches <= 0 {
		maxMatches = defaultMaxMatches
	}
	offset := input.Offset
	if offset < 0 {
		offset = 0
	}
	opts := grepOptions{
		re:          re,
		mode:        mode,
		matchesOnly: input.MatchesOnly,
		encoding:    input.Encoding,
		maxFileSize: h.config.MemoryThreshold,
	}
	switch mode {
	case outputModeContent:
		opts.contextBefore = input.ContextBefore
		opts.contextAfter = input.ContextAfter
		// One past the page end, so a single file overflowing still reports truncation.
		opts.perFileLimit = offset + maxMatches + 1
	case outputModeFiles:
		opts.perFileLimit = 1 // stop reading the moment the file qualifies
	}
	files := h.collectFiles(ctx, input.Paths, input.Include, input.Exclude)
	if len(files) == 0 {
		return &mcp.CallToolResult{}, GrepOutput{Matches: []GrepMatch{}, FilesSearched: 0}, nil
	}
	output := h.searchFiles(ctx, files, opts, maxMatches, offset)
	output.FilesSearched = len(files)
	if output.Truncated {
		output.NextOffset = offset + maxMatches
	}
	return &mcp.CallToolResult{}, output, nil
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
// order so the truncated set is the same on every run, and stopping once the page is full.
// Results before offset are counted and dropped; the mode decides what gets collected.
func (h *Handler) searchFiles(ctx context.Context, files []string, opts grepOptions, maxMatches, offset int) GrepOutput {
	out := GrepOutput{Matches: []GrepMatch{}}
	skip, taken := offset, 0

	// take reports whether the page is still open after admitting one more result.
	take := func(admit func()) bool {
		if skip > 0 {
			skip--
			return true
		}
		if taken >= maxMatches {
			out.Truncated = true
			return false
		}
		admit()
		taken++
		return true
	}

	workpool.RunOrdered(ctx, files, workpool.Options{},
		func(ctx context.Context, _ int, path string) fileHits {
			if ctx.Err() != nil {
				return fileHits{}
			}
			return searchSingleFile(path, opts)
		},
		func(_ int, hits fileHits) bool {
			if hits.count == 0 {
				return true
			}
			out.FilesMatched++
			switch opts.mode {
			case outputModeFiles:
				path := hits.matches[0].Path
				return take(func() { out.Files = append(out.Files, path) })
			case outputModeCount:
				c := GrepFileCount{Path: hits.matches[0].Path, Count: hits.count}
				return take(func() { out.Counts = append(out.Counts, c) })
			}
			for _, m := range hits.matches {
				if !take(func() { out.Matches = append(out.Matches, m) }) {
					return false
				}
			}
			return true
		})

	// totalMatches counts what the page actually reports: matching lines in content
	// mode, paths in files_with_matches (one hit each, the search stopped there),
	// and the summed per-file counts in count mode.
	out.TotalMatches = taken
	if opts.mode == outputModeCount {
		out.TotalMatches = 0
		for _, c := range out.Counts {
			out.TotalMatches += c.Count
		}
	}
	return out
}

// searchSingleFile searches one file under the resolved options. In content mode it
// collects matches; otherwise it only counts, so nothing is built to be thrown away.
func searchSingleFile(path string, opts grepOptions) fileHits {
	// Check file size - warn if large file will be loaded to memory
	if info, err := os.Stat(path); err == nil && info.Size() > opts.maxFileSize {
		slog.Warn("loading large file into memory", "path", path, "size", info.Size(), "threshold", opts.maxFileSize)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fileHits{}
	}
	// Decode first: UTF-16 text is full of NUL bytes, so classify the decoded text.
	content, detectedEncoding := decodeFileContent(data, opts.encoding)
	if content == "" || isBinaryText(content) {
		return fileHits{}
	}
	lines := splitGrepLines(content)
	var hits fileHits
	for lineNum, line := range lines {
		locs := findLineMatches(opts, line)
		if len(locs) == 0 {
			continue
		}
		if opts.mode != outputModeContent {
			// The path is all the caller needs; carry it on a bare match.
			if hits.count == 0 {
				hits.matches = []GrepMatch{{Path: path}}
			}
			hits.count += len(locs)
			if opts.perFileLimit > 0 && hits.count >= opts.perFileLimit {
				break
			}
			continue
		}
		for _, loc := range locs {
			text := line
			if opts.matchesOnly {
				text = line[loc[0]:loc[1]]
			}
			match := GrepMatch{
				Path:     path,
				Line:     lineNum + 1,
				Column:   loc[0] + 1,
				Text:     text,
				Encoding: detectedEncoding,
			}
			if opts.contextBefore > 0 {
				match.Before = getContextBefore(lines, lineNum, opts.contextBefore)
			}
			if opts.contextAfter > 0 {
				match.After = getContextAfter(lines, lineNum, opts.contextAfter)
			}
			hits.matches = append(hits.matches, match)
			hits.count++
		}
		if opts.perFileLimit > 0 && hits.count >= opts.perFileLimit {
			break
		}
	}
	return hits
}

// findLineMatches returns the match spans on a line: every occurrence when
// matchesOnly extracts substrings, otherwise just the first — one match per line.
func findLineMatches(opts grepOptions, line string) [][]int {
	if opts.matchesOnly && opts.mode == outputModeContent {
		return opts.re.FindAllStringIndex(line, -1)
	}
	if loc := opts.re.FindStringIndex(line); loc != nil {
		return [][]int{loc}
	}
	return nil
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
