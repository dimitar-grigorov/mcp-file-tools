// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pmezard/go-difflib/difflib"
)

// HandleEditFile applies line-based edits to a text file with encoding support.
func (h *Handler) HandleEditFile(ctx context.Context, req *mcp.CallToolRequest, input EditFileInput) (*mcp.CallToolResult, EditFileOutput, error) {
	if input.Patch != "" && input.Edits != nil {
		return errorResult("patch and edits are mutually exclusive"), EditFileOutput{}, nil
	}
	if input.Patch == "" && len(input.Edits) == 0 {
		return errorResult("provide either a non-empty edits array or patch"), EditFileOutput{}, nil
	}

	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, EditFileOutput{}, nil
	}

	if loadToMemory, size := h.shouldLoadEntireFile(v.Path); !loadToMemory {
		slog.Warn("loading large file into memory", "path", input.Path, "size", size, "threshold", h.config.MemoryThreshold)
	}

	originalMode := getFileMode(v.Path)

	readOnlyCleared := false
	forceWritable := input.ForceWritable != nil && *input.ForceWritable // default: false
	if isReadOnly(originalMode) {
		if !forceWritable {
			return errorResult("file is read-only — STOP, do NOT retry and do NOT attempt to change file attributes. Ask the user whether to proceed with forceWritable: true, or skip this file"), EditFileOutput{}, nil
		}
		if !input.DryRun {
			if err := clearReadOnly(v.Path, originalMode); err != nil {
				return errorResult(fmt.Sprintf("failed to clear read-only flag: %v", err)), EditFileOutput{}, nil
			}
			readOnlyCleared = true
			slog.Info("cleared read-only flag", "path", input.Path)
		}
	}

	data, err := os.ReadFile(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read file: %v", err)), EditFileOutput{}, nil
	}

	encodingName, err := h.resolveEncodingFromData(input.Encoding, data, input.Path)
	if err != nil {
		return errorResult(err.Error()), EditFileOutput{}, nil
	}

	content, err := encoding.Decode(data, encodingName) // name already validated above
	if err != nil {
		return errorResult(fmt.Sprintf("failed to decode file with %s: %v", encodingName, err)), EditFileOutput{}, nil
	}
	slog.Debug("edit_file: decoded content", "path", input.Path, "encoding", encodingName, "originalSize", len(data), "decodedSize", len(content))

	// Detect on decoded text: UTF-16 has a 00 between CR and LF.
	lineEndings := DetectLineEndings([]byte(content))
	if lineEndings.Style == LineEndingMixed {
		slog.Warn("file has mixed line endings", "path", input.Path, "crlf", lineEndings.CRLFCount, "lf", lineEndings.LFCount)
	}

	content = ConvertLineEndings(content, LineEndingLF)
	var modifiedContent string
	if input.Patch != "" {
		modifiedContent, err = applyPatch(content, input.Patch)
	} else {
		modifiedContent, err = applyEdits(content, input.Edits)
	}
	if err != nil {
		return errorResult(err.Error()), EditFileOutput{}, nil
	}

	diff := createUnifiedDiff(content, modifiedContent, input.Path)

	if !input.DryRun {
		if r := cancelled(ctx); r != nil {
			return r, EditFileOutput{}, nil
		}
		if err := atomicWriteFileWithEncoding(v.Path, modifiedContent, encodingName, lineEndings.Style, originalMode); err != nil {
			return errorResult(fmt.Sprintf("failed to write file: %v", err)), EditFileOutput{}, nil
		}
	}

	text := diff
	if readOnlyCleared {
		text += "\nRead-only flag was cleared."
	}

	output := EditFileOutput{Diff: diff, ReadOnlyCleared: readOnlyCleared}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, output, nil
}

type patchHunk struct {
	oldStart int
	oldLines []string
	newLines []string
}

func applyPatch(content, patch string) (string, error) {
	hunks, err := parseUnifiedPatch(patch)
	if err != nil {
		return "", err
	}

	modified := content
	lineOffset := 0
	for i, hunk := range hunks {
		oldText := strings.Join(hunk.oldLines, "\n")
		newText := strings.Join(hunk.newLines, "\n")
		if len(hunk.oldLines) == 0 {
			modified, err = insertPatchLines(modified, newText, hunk.oldStart+lineOffset)
		} else {
			modified, err = applyEdits(modified, []EditOperation{{OldText: oldText, NewText: newText}})
		}
		if err != nil {
			return "", fmt.Errorf("patch hunk %d failed: %w", i+1, err)
		}
		lineOffset += len(hunk.newLines) - len(hunk.oldLines)
	}
	return modified, nil
}

func parseUnifiedPatch(patch string) ([]patchHunk, error) {
	lines := strings.Split(ConvertLineEndings(patch, LineEndingLF), "\n")
	i := 0
	for i < len(lines) && lines[i] == "" {
		i++
	}
	if i >= len(lines) || !strings.HasPrefix(lines[i], "--- ") {
		return nil, fmt.Errorf("patch must start with --- and +++ file headers")
	}
	i++
	if i >= len(lines) || !strings.HasPrefix(lines[i], "+++ ") {
		return nil, fmt.Errorf("patch must start with --- and +++ file headers")
	}
	i++

	var hunks []patchHunk
	for i < len(lines) {
		if lines[i] == "" && i == len(lines)-1 {
			break
		}
		if strings.HasPrefix(lines[i], "--- ") {
			return nil, fmt.Errorf("patch contains multiple files")
		}
		oldStart, oldCount, newCount, err := parseHunkHeader(lines[i])
		if err != nil {
			return nil, fmt.Errorf("invalid patch hunk %d: %w", len(hunks)+1, err)
		}
		i++

		hunk := patchHunk{oldStart: oldStart}
		oldSeen, newSeen := 0, 0
		for oldSeen < oldCount || newSeen < newCount {
			if i >= len(lines) {
				return nil, fmt.Errorf("invalid patch hunk %d: unexpected end", len(hunks)+1)
			}
			line := lines[i]
			i++
			if line == `\ No newline at end of file` {
				continue
			}
			if line == "" {
				return nil, fmt.Errorf("invalid patch hunk %d: unprefixed line", len(hunks)+1)
			}
			switch line[0] {
			case ' ':
				hunk.oldLines = append(hunk.oldLines, line[1:])
				hunk.newLines = append(hunk.newLines, line[1:])
				oldSeen++
				newSeen++
			case '-':
				hunk.oldLines = append(hunk.oldLines, line[1:])
				oldSeen++
			case '+':
				hunk.newLines = append(hunk.newLines, line[1:])
				newSeen++
			default:
				return nil, fmt.Errorf("invalid patch hunk %d: line lacks a diff prefix", len(hunks)+1)
			}
			if oldSeen > oldCount || newSeen > newCount {
				return nil, fmt.Errorf("invalid patch hunk %d: line counts exceed header", len(hunks)+1)
			}
		}
		if i < len(lines) && lines[i] == `\ No newline at end of file` {
			i++
		}
		hunks = append(hunks, hunk)
	}
	if len(hunks) == 0 {
		return nil, fmt.Errorf("patch contains no hunks")
	}
	return hunks, nil
}

func parseHunkHeader(header string) (oldStart, oldCount, newCount int, err error) {
	fields := strings.Fields(header)
	if len(fields) < 4 || fields[0] != "@@" || fields[3] != "@@" {
		return 0, 0, 0, fmt.Errorf("expected @@ -old +new @@ header")
	}
	oldStart, oldCount, err = parseHunkRange(fields[1], '-')
	if err != nil {
		return 0, 0, 0, err
	}
	_, newCount, err = parseHunkRange(fields[2], '+')
	return oldStart, oldCount, newCount, err
}

func parseHunkRange(value string, prefix byte) (start, count int, err error) {
	if len(value) < 2 || value[0] != prefix {
		return 0, 0, fmt.Errorf("invalid range %q", value)
	}
	parts := strings.SplitN(value[1:], ",", 2)
	start, err = strconv.Atoi(parts[0])
	if err != nil || start < 0 {
		return 0, 0, fmt.Errorf("invalid range %q", value)
	}
	count = 1
	if len(parts) == 2 {
		count, err = strconv.Atoi(parts[1])
		if err != nil || count < 0 {
			return 0, 0, fmt.Errorf("invalid range %q", value)
		}
	}
	return start, count, nil
}

func insertPatchLines(content, newText string, line int) (string, error) {
	lines := strings.Split(content, "\n")
	if content == "" {
		lines = nil
	}
	if line < 0 || line > len(lines) {
		return "", fmt.Errorf("insertion line %d is outside the file", line)
	}
	newLines := strings.Split(newText, "\n")
	result := make([]string, 0, len(lines)+len(newLines))
	result = append(result, lines[:line]...)
	result = append(result, newLines...)
	result = append(result, lines[line:]...)
	return strings.Join(result, "\n"), nil
}

// applyEdits applies edits sequentially, trying exact then whitespace-flexible match.
// On failure it returns ErrEditNoMatch with a hint pointing at the closest match.
func applyEdits(content string, edits []EditOperation) (string, error) {
	modifiedContent := content

	for _, edit := range edits {
		if edit.OldText == "" {
			return "", ErrOldTextEmpty
		}
		if edit.Similarity != nil && (*edit.Similarity < 0 || *edit.Similarity > 1) {
			return "", fmt.Errorf("similarity must be between 0.0 and 1.0")
		}

		normalizedOld := ConvertLineEndings(edit.OldText, LineEndingLF)
		normalizedNew := ConvertLineEndings(edit.NewText, LineEndingLF)

		// Try exact match first
		if strings.Contains(modifiedContent, normalizedOld) {
			modifiedContent = strings.Replace(modifiedContent, normalizedOld, normalizedNew, 1)
			continue
		}

		// Try whitespace-flexible line matching
		matched, result := tryFlexibleMatch(modifiedContent, normalizedOld, normalizedNew)
		if matched {
			modifiedContent = result
			continue
		}
		if edit.Similarity != nil {
			candidate := closestCandidate(modifiedContent, normalizedOld)
			if candidate.start >= 0 && candidate.score >= *edit.Similarity {
				modifiedContent = replaceLineBlock(modifiedContent, normalizedOld, normalizedNew, candidate.start, candidate.lines)
				continue
			}
			return "", noMatchError(modifiedContent, normalizedOld, edit.OldText, edit.Similarity)
		}

		return "", noMatchError(modifiedContent, normalizedOld, edit.OldText, nil)
	}

	return modifiedContent, nil
}

// noMatchError wraps ErrEditNoMatch, appending the closest matching block if found.
func noMatchError(content, normalizedOld, rawOld string, threshold *float64) error {
	candidate := closestCandidate(content, normalizedOld)
	if candidate.start < 0 || candidate.score == 0 {
		if threshold != nil {
			return fmt.Errorf("%w:\n%s\n\nBest candidate scored %.2f, threshold %g",
				ErrEditNoMatch, rawOld, candidate.score, *threshold)
		}
		return fmt.Errorf("%w:\n%s", ErrEditNoMatch, rawOld)
	}

	lines := strings.Split(content, "\n")
	start := max(0, candidate.start-1)
	end := min(len(lines), candidate.start+candidate.lines+1)
	snippet := strings.Join(lines[start:end], "\n")
	score := ""
	if threshold != nil {
		score = fmt.Sprintf(" Best candidate scored %.2f, threshold %g.", candidate.score, *threshold)
	}

	return fmt.Errorf("%w:\n%s\n\n"+
		"HINT: the closest match starts at line %d (%d line edits away, ignoring whitespace).%s\n"+
		"Actual file content there:\n%s\n\n"+
		"Copy the snippet above into oldText and retry",
		ErrEditNoMatch, rawOld, candidate.start+1, candidate.distance, score, snippet)
}

type matchCandidate struct {
	start    int
	lines    int
	distance int
	score    float64
}

const (
	closestCandidateBudget = 20_000_000 // edit-distance cells before narrowing
	closestCandidateStarts = 48         // start positions kept when narrowed
	closestCandidateBand   = 8          // window sizes either side of len(oldText)
)

// closestCandidate finds the block nearest to oldText by line edit distance.
// Exhaustive search is cubic in oldText, so past the budget it narrows.
func closestCandidate(content, oldText string) matchCandidate {
	contentLines := strings.Split(content, "\n")
	oldLines := strings.Split(oldText, "\n")
	if len(contentLines) == 0 || len(oldLines) == 0 {
		return matchCandidate{start: -1}
	}

	// Trim once: trimming inside the distance loop re-did every line per window.
	trimmedContent := trimLines(contentLines)
	trimmedOld := trimLines(oldLines)

	maxSize := min(len(contentLines), len(oldLines)*2)
	starts, sizes := candidateSearchSpace(trimmedContent, trimmedOld, maxSize)

	prev := make([]int, len(trimmedContent)+1)
	cur := make([]int, len(trimmedContent)+1)

	best := matchCandidate{start: -1, score: -1}
	for _, size := range sizes {
		for _, i := range starts {
			if i+size > len(trimmedContent) {
				continue
			}
			distance := lineEditDistance(trimmedOld, trimmedContent[i:i+size], prev, cur)
			if distance > len(trimmedOld) {
				continue
			}
			score := 1 - float64(distance)/float64(max(len(trimmedOld), size))
			if score > best.score || score == best.score && distance < best.distance {
				best = matchCandidate{start: i, lines: size, distance: distance, score: score}
			}
		}
	}
	return best
}

func trimLines(lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = strings.TrimSpace(line)
	}
	return out
}

// candidateSearchSpace returns starts and sizes ascending; all of both under budget.
func candidateSearchSpace(content, old []string, maxSize int) (starts, sizes []int) {
	work := float64(len(content)) * float64(len(old)) * float64(maxSize) * float64(maxSize) / 2

	if work <= closestCandidateBudget {
		starts = make([]int, len(content))
		for i := range starts {
			starts[i] = i
		}
		sizes = make([]int, 0, maxSize)
		for s := 1; s <= maxSize; s++ {
			sizes = append(sizes, s)
		}
		return starts, sizes
	}

	for s := max(1, len(old)-closestCandidateBand); s <= min(maxSize, len(old)+closestCandidateBand); s++ {
		sizes = append(sizes, s)
	}
	return promisingStarts(content, old, closestCandidateStarts), sizes
}

// promisingStarts keeps the positions whose window shares the most lines with old.
func promisingStarts(content, old []string, keep int) []int {
	inOld := make(map[string]struct{}, len(old))
	for _, line := range old {
		if line != "" {
			inOld[line] = struct{}{}
		}
	}

	window := min(len(old), len(content))
	hits := 0
	for _, line := range content[:window] {
		if _, ok := inOld[line]; ok {
			hits++
		}
	}

	type ranked struct{ start, hits int }
	scores := make([]ranked, 0, len(content)-window+1)
	scores = append(scores, ranked{0, hits})
	for i := window; i < len(content); i++ {
		if _, ok := inOld[content[i]]; ok {
			hits++
		}
		if _, ok := inOld[content[i-window]]; ok {
			hits--
		}
		scores = append(scores, ranked{i - window + 1, hits})
	}

	// Stable, so an earlier position still wins a tie as it did before.
	sort.SliceStable(scores, func(a, b int) bool { return scores[a].hits > scores[b].hits })
	scores = scores[:min(keep, len(scores))]

	starts := make([]int, len(scores))
	for i, s := range scores {
		starts[i] = s.start
	}
	sort.Ints(starts)
	return starts
}

// lineEditDistance is Levenshtein over pre-trimmed lines; prev and cur are
// caller-owned scratch, so a sweep allocates nothing.
func lineEditDistance(a, b []string, prev, cur []int) int {
	prev, cur = prev[:len(b)+1], cur[:len(b)+1]
	for j := range prev {
		prev[j] = j
	}
	for i, left := range a {
		cur[0] = i + 1
		for j, right := range b {
			cost := 1
			if left == right {
				cost = 0
			}
			cur[j+1] = min(prev[j+1]+1, cur[j]+1, prev[j]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func replaceLineBlock(content, oldText, newText string, start, oldLineCount int) string {
	contentLines := strings.Split(content, "\n")
	oldLines := strings.Split(oldText, "\n")
	newLines := strings.Split(newText, "\n")
	baseIndent := getLeadingWhitespace(contentLines[start])
	for i := range newLines {
		if i == 0 {
			newLines[i] = baseIndent + strings.TrimLeft(newLines[i], " \t")
		} else {
			newLines[i] = adjustRelativeIndent(oldLines, newLines[i], i, baseIndent)
		}
	}

	result := make([]string, 0, len(contentLines)-oldLineCount+len(newLines))
	result = append(result, contentLines[:start]...)
	result = append(result, newLines...)
	result = append(result, contentLines[start+oldLineCount:]...)
	return strings.Join(result, "\n")
}

// tryFlexibleMatch matches oldText ignoring whitespace differences, preserving file indentation.
func tryFlexibleMatch(content, oldText, newText string) (bool, string) {
	oldLines := strings.Split(oldText, "\n")
	contentLines := strings.Split(content, "\n")

	if len(contentLines) < len(oldLines) {
		return false, ""
	}

	for i := 0; i <= len(contentLines)-len(oldLines); i++ {
		potentialMatch := contentLines[i : i+len(oldLines)]

		isMatch := true
		for j, oldLine := range oldLines {
			if strings.TrimSpace(oldLine) != strings.TrimSpace(potentialMatch[j]) {
				isMatch = false
				break
			}
		}

		if isMatch {
			return true, replaceLineBlock(content, oldText, newText, i, len(oldLines))
		}
	}

	return false, ""
}

// adjustRelativeIndent applies baseIndent plus the indentation delta between old and new lines.
func adjustRelativeIndent(oldLines []string, newLine string, lineIndex int, baseIndent string) string {
	if lineIndex >= len(oldLines) {
		return newLine
	}

	oldIndent := getLeadingWhitespace(oldLines[lineIndex])
	newIndent := getLeadingWhitespace(newLine)

	relativeIndent := len(newIndent) - len(oldIndent)
	trimmedContent := strings.TrimLeft(newLine, " \t")
	switch {
	case relativeIndent > 0:
		return baseIndent + strings.Repeat(" ", relativeIndent) + trimmedContent
	case relativeIndent < 0:
		// Negative indent: trim characters from the end of baseIndent
		trim := -relativeIndent
		if trim >= len(baseIndent) {
			return trimmedContent
		}
		return baseIndent[:len(baseIndent)-trim] + trimmedContent
	default:
		return baseIndent + trimmedContent
	}
}

func getLeadingWhitespace(s string) string {
	for i, c := range s {
		if c != ' ' && c != '\t' {
			return s[:i]
		}
	}
	return s // entire string is whitespace
}

func createUnifiedDiff(original, modified, filepath string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(original),
		B:        difflib.SplitLines(modified),
		FromFile: filepath,
		ToFile:   filepath,
		Context:  3,
	}
	text, _ := difflib.GetUnifiedDiffString(diff)
	return text
}

// atomicWriteFileWithEncoding encodes UTF-8 content to the target encoding and writes atomically.
func atomicWriteFileWithEncoding(path, content, encodingName, lineEndingStyle string, mode os.FileMode) error {
	content = ConvertLineEndings(content, lineEndingStyle)

	var dataToWrite []byte
	if encoding.IsUTF8(encodingName) {
		dataToWrite = []byte(content)
	} else {
		enc, ok := encoding.Get(encodingName)
		if !ok {
			return fmt.Errorf("unsupported encoding: %s", encodingName)
		}
		encoded, err := encoding.Encode(content, enc, encodingName)
		if err != nil {
			return fmt.Errorf("failed to encode content to %s: %w", encodingName, err)
		}
		dataToWrite = encoded
		slog.Debug("edit_file: encoded content for write", "encoding", encodingName, "utf8Size", len(content), "encodedSize", len(encoded))
	}

	return atomicWriteFile(path, dataToWrite, mode)
}

func isReadOnly(mode os.FileMode) bool {
	return mode&0200 == 0
}

// clearReadOnly adds owner write permission to the file.
func clearReadOnly(path string, currentMode os.FileMode) error {
	newMode := currentMode | 0200
	return os.Chmod(path, newMode)
}
