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
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/encoding"
	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/filesystem"
	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/workpool"
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
	fallback      string // configured default, when detection decides nothing
	maxFileSize   int64
	perFileLimit  int // 0 = unlimited: count mode must see the whole file
}

// fileHits is one file's outcome; matches is filled in content mode only.
type fileHits struct {
	matches []GrepMatch
	count   int
}

// HandleGrep searches for a pattern in files with encoding support.
func (h *Handler) HandleGrep(ctx context.Context, req *mcp.CallToolRequest, input GrepInput) (*mcp.CallToolResult, GrepOutput, error) {
	if input.Pattern != "" && len(input.Patterns) > 0 {
		return errorResult("pattern and patterns cannot be used together"), GrepOutput{}, nil
	}
	if input.Pattern == "" && len(input.Patterns) == 0 {
		return errorResult("pattern is required"), GrepOutput{}, nil
	}
	if len(input.Paths) == 0 {
		return errorResult("paths is required"), GrepOutput{}, nil
	}
	if input.Include != "" && len(input.Includes) > 0 {
		return errorResult("include and includes cannot be used together"), GrepOutput{}, nil
	}
	if input.Exclude != "" && len(input.Excludes) > 0 {
		return errorResult("exclude and excludes cannot be used together"), GrepOutput{}, nil
	}
	includes := input.Includes
	if input.Include != "" {
		includes = []string{input.Include}
	}
	excludes := input.Excludes
	if input.Exclude != "" {
		excludes = []string{input.Exclude}
	}
	includes = expandBasenameGlobs(includes)
	excludes = expandBasenameGlobs(excludes)
	mode := input.OutputMode
	if mode == "" {
		mode = outputModeContent
	}
	if mode != outputModeContent && mode != outputModeFiles && mode != outputModeCount {
		return errorResult(fmt.Sprintf("invalid outputMode %q: use %q, %q or %q",
			input.OutputMode, outputModeContent, outputModeFiles, outputModeCount)), GrepOutput{}, nil
	}
	patterns := input.Patterns
	if input.Pattern != "" {
		patterns = []string{input.Pattern}
	}
	if slices.Contains(patterns, "") {
		return errorResult("patterns contains an empty pattern, which matches every line"), GrepOutput{}, nil
	}
	re, err := compilePattern(anyOf(patterns), input.CaseSensitive)
	if err != nil {
		return errorResult(namePatternError(patterns, input.CaseSensitive, err)), GrepOutput{}, nil
	}
	maxMatches := input.MaxMatches
	if maxMatches <= 0 {
		maxMatches = defaultMaxMatches
	}
	offset := max(input.Offset, 0)
	searchEncoding, encodingHint := resolveGrepEncoding(input.Encoding)
	opts := grepOptions{
		re:          re,
		mode:        mode,
		matchesOnly: input.MatchesOnly,
		encoding:    searchEncoding,
		fallback:    h.fallbackEncoding(),
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
	files := h.collectFiles(ctx, input.Paths, includes, excludes, gitignoreDefault(input.RespectGitignore))
	if len(files) == 0 {
		return &mcp.CallToolResult{}, GrepOutput{Matches: []GrepMatch{}, FilesSearched: 0, Hint: encodingHint}, nil
	}
	output := h.searchFiles(ctx, files, opts, maxMatches, offset)
	output.Hint = encodingHint
	if output.Truncated {
		output.NextOffset = offset + maxMatches
	}
	return &mcp.CallToolResult{}, output, nil
}

// resolveGrepEncoding falls back to per-file detection on an unknown name, rather than failing the search, and says so.
func resolveGrepEncoding(requested string) (string, string) {
	if _, ok := encoding.Canonical(requested); ok || requested == "" {
		return requested, ""
	}
	// Phrased as an instruction because models relay instructions and ignore trivia.
	return "", fmt.Sprintf(
		"Encoding %q is not supported, so each file's encoding was auto-detected instead — tell the user. "+
			"Call list_encodings for the supported names.", requested)
}

func compilePattern(pattern string, caseSensitive *bool) (*regexp.Regexp, error) {
	if caseSensitive != nil && !*caseSensitive {
		pattern = "(?i)" + pattern
	}
	return regexp.Compile(pattern)
}

// anyOf folds patterns into one alternation — the search stays single-pattern code, RE2 walks each line once, and grouping stops a pattern's own "a|b" swallowing the next.
func anyOf(patterns []string) string {
	if len(patterns) == 1 {
		return patterns[0]
	}
	return "(?:" + strings.Join(patterns, ")|(?:") + ")"
}

// namePatternError says which pattern is bad; the alternation's own error points into a string the caller never wrote.
func namePatternError(patterns []string, caseSensitive *bool, err error) string {
	for _, p := range patterns {
		if _, e := compilePattern(p, caseSensitive); e != nil {
			return fmt.Sprintf("invalid regex pattern %q: %v", p, e)
		}
	}
	return fmt.Sprintf("invalid regex pattern: %v", err)
}

func (h *Handler) collectFiles(ctx context.Context, paths, includes, excludes []string, gitignore bool) []string {
	var files []string
	seen := make(map[string]bool)
	allowedDirs := h.ResolvedAllowedDirs()
	for _, path := range paths {
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
				AllowedDirs:      allowedDirs,
				RespectGitignore: gitignore,
				OnError: func(p string, _ int, err error) error {
					slog.Debug("skipping path due to error", "path", p, "error", err)
					return nil
				},
			}
			_ = filesystem.Walk(ctx, v.Path, opts, func(e filesystem.Entry) (filesystem.Action, error) {
				if e.IsDir() {
					return filesystem.Continue, nil
				}
				if shouldIncludeFile(e.Path, includes, excludes) && !seen[e.Path] {
					seen[e.Path] = true
					files = append(files, e.Path)
				}
				return filesystem.Continue, nil
			})
		} else if shouldIncludeFile(v.Path, includes, excludes) && !seen[v.Path] {
			seen[v.Path] = true
			files = append(files, v.Path)
		}
	}
	return files
}

// expandBasenameGlobs expands {a,b} alternatives and drops a leading "**/", which models send out of habit.
func expandBasenameGlobs(patterns []string) []string {
	if len(patterns) == 0 {
		return patterns
	}
	out := make([]string, 0, len(patterns))
	for _, p := range patterns {
		for _, e := range expandBraces(p) {
			out = append(out, strings.TrimPrefix(e, "**/"))
		}
	}
	return out
}

// shouldIncludeFile matches patterns against the basename.
func shouldIncludeFile(path string, includes, excludes []string) bool {
	base := filepath.Base(path)
	for _, pattern := range excludes {
		if matched, _ := filepath.Match(pattern, base); matched {
			return false
		}
	}
	if len(includes) == 0 {
		return true
	}
	for _, pattern := range includes {
		if matched, _ := filepath.Match(pattern, base); matched {
			return true
		}
	}
	return false
}

// searchFiles runs bounded-concurrent, committing in file order so a truncated result is the same on every run.
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

	stats := workpool.RunOrdered(ctx, files, workpool.Options{},
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

	// Only the committed files were accounted for; a full page stops the rest.
	out.FilesSearched = stats.Committed

	// totalMatches is what the page holds: lines, or paths, or the summed counts.
	out.TotalMatches = taken
	if opts.mode == outputModeCount {
		out.TotalMatches = 0
		for _, c := range out.Counts {
			out.TotalMatches += c.Count
		}
	}
	return out
}

// searchSingleFile only counts outside content mode, building nothing to discard.
func searchSingleFile(path string, opts grepOptions) fileHits {
	if info, err := os.Stat(path); err == nil && info.Size() > opts.maxFileSize {
		slog.Warn("loading large file into memory", "path", path, "size", info.Size(), "threshold", opts.maxFileSize)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fileHits{}
	}
	// Decode first: UTF-16 text is full of NUL bytes, so classify the decoded text.
	content, detectedEncoding := decodeFileContent(data, opts.encoding, opts.fallback)
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
			// The path is all the caller needs, so carry it on a bare match.
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

// findLineMatches returns every occurrence for matchesOnly, else the first only.
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

// detectGrepEncoding picks the search encoding; valid UTF-8 beats a weak guess.
func detectGrepEncoding(data []byte, fallback string) string {
	detection, trusted := encoding.DetectSample(data)
	if detection.Charset == "" {
		return fallback
	}
	if trusted || !utf8.Valid(data) {
		return detection.Charset
	}
	return fallback
}

// decodeFileContent decodes file data to UTF-8, falling back the same way reads do.
func decodeFileContent(data []byte, forcedEncoding, fallback string) (string, string) {
	if fallback == "" {
		fallback = "utf-8"
	}
	if forcedEncoding == "" {
		return decodeWithFallback(data, detectGrepEncoding(data, fallback), fallback)
	}
	return decodeWithFallback(data, canonicalCharset(forcedEncoding), fallback)
}

// decodeWithFallback decodes as name, retrying once as fallback; the BOM is dropped so it can't shift line 1 columns or break ^ anchors.
func decodeWithFallback(data []byte, name, fallback string) (string, string) {
	if decoded, err := encoding.Decode(data, name); err == nil {
		content, _ := trimContentBOM(decoded)
		return content, name
	}
	if name == fallback || encoding.IsUTF8(fallback) {
		content, _ := trimContentBOM(string(data))
		return content, fallback
	}
	return decodeWithFallback(data, fallback, fallback)
}

func getContextBefore(lines []string, lineIdx, count int) []string {
	start := max(lineIdx-count, 0)
	if start >= lineIdx {
		return nil
	}
	return lines[start:lineIdx]
}

func getContextAfter(lines []string, lineIdx, count int) []string {
	end := min(lineIdx+count+1, len(lines))
	if lineIdx+1 >= end {
		return nil
	}
	return lines[lineIdx+1 : end]
}
