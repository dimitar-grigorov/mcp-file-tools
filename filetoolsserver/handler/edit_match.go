// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

// Finding oldText and splicing newText in: exact, then whitespace-flexible, then the edit_fuzzy.go fallback.

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// applyEdits applies edits in order and counts the places changed; several matches need replaceAll.
func applyEdits(content string, edits []EditOperation) (string, int, error) {
	return applyEditsMatching(content, edits, false)
}

// applyEditsMatching with wholeLines matches only complete lines, as a patch hunk means them.
func applyEditsMatching(content string, edits []EditOperation, wholeLines bool) (string, int, error) {
	modifiedContent := content
	replacements := 0

	for _, edit := range edits {
		if edit.OldText == "" {
			return "", 0, ErrOldTextEmpty
		}
		if edit.Similarity != nil && (*edit.Similarity < 0 || *edit.Similarity > 1) {
			return "", 0, fmt.Errorf("similarity must be between 0.0 and 1.0")
		}

		normalizedOld := ConvertLineEndings(edit.OldText, LineEndingLF)
		normalizedNew := ConvertLineEndings(edit.NewText, LineEndingLF)

		// Try exact match first
		if starts := exactMatchStarts(modifiedContent, normalizedOld, wholeLines); len(starts) > 0 {
			if len(starts) > 1 && !edit.ReplaceAll {
				return "", 0, ambiguousError(edit.OldText, lineNumbers(modifiedContent, starts))
			}
			modifiedContent = replaceAt(modifiedContent, starts, len(normalizedOld), normalizedNew)
			replacements += len(starts)
			continue
		}

		// Try whitespace-flexible line matching
		if starts := flexibleMatchStarts(modifiedContent, normalizedOld); len(starts) > 0 {
			if len(starts) > 1 && !edit.ReplaceAll {
				return "", 0, ambiguousError(edit.OldText, offsetLines(starts))
			}
			oldLineCount := len(strings.Split(normalizedOld, "\n"))
			// Last to first: replacing later blocks leaves earlier indexes valid.
			for _, start := range slices.Backward(starts) {
				modifiedContent = replaceLineBlock(modifiedContent, normalizedOld, normalizedNew, start, oldLineCount)
			}
			replacements += len(starts)
			continue
		}

		candidate := closestCandidate(modifiedContent, normalizedOld)
		if edit.Similarity != nil && candidate.start >= 0 && candidate.score >= *edit.Similarity {
			modifiedContent = replaceLineBlock(modifiedContent, normalizedOld, normalizedNew, candidate.start, candidate.lines)
			replacements++
			continue
		}
		return "", 0, noMatchError(modifiedContent, candidate, edit.OldText, edit.Similarity)
	}

	return modifiedContent, replacements, nil
}

// exactMatchStarts returns the offset of every non-overlapping exact match; wholeLines drops those inside a line.
func exactMatchStarts(content, needle string, wholeLines bool) []int {
	var starts []int
	for offset := 0; ; {
		i := strings.Index(content[offset:], needle)
		if i < 0 {
			return starts
		}
		at, end := offset+i, offset+i+len(needle)
		if wholeLines && ((at > 0 && content[at-1] != '\n') || (end < len(content) && content[end] != '\n')) {
			offset = at + 1
			continue
		}
		starts = append(starts, at)
		offset = end
	}
}

// lineNumbers turns ascending byte offsets into 1-based line numbers in one pass.
func lineNumbers(content string, starts []int) []int {
	lines := make([]int, len(starts))
	line, prev := 1, 0
	for i, at := range starts {
		line += strings.Count(content[prev:at], "\n")
		lines[i], prev = line, at
	}
	return lines
}

// replaceAt splices newText over each match of oldLen bytes at the ascending starts.
func replaceAt(content string, starts []int, oldLen int, newText string) string {
	var b strings.Builder
	b.Grow(len(content) + len(starts)*(len(newText)-oldLen))
	prev := 0
	for _, at := range starts {
		b.WriteString(content[prev:at])
		b.WriteString(newText)
		prev = at + oldLen
	}
	b.WriteString(content[prev:])
	return b.String()
}

// flexibleMatchStarts returns each non-overlapping block matching oldText, ignoring per-line whitespace.
func flexibleMatchStarts(content, oldText string) []int {
	oldLines := strings.Split(oldText, "\n")
	contentLines := strings.Split(content, "\n")

	if len(contentLines) < len(oldLines) {
		return nil
	}

	var starts []int
	for i := 0; i <= len(contentLines)-len(oldLines); i++ {
		isMatch := true
		for j, oldLine := range oldLines {
			if strings.TrimSpace(oldLine) != strings.TrimSpace(contentLines[i+j]) {
				isMatch = false
				break
			}
		}
		if isMatch {
			starts = append(starts, i)
			i += len(oldLines) - 1 // non-overlapping
		}
	}
	return starts
}

// offsetLines turns 0-based line indexes into 1-based line numbers.
func offsetLines(starts []int) []int {
	lines := make([]int, len(starts))
	for i, s := range starts {
		lines[i] = s + 1
	}
	return lines
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

// ambiguousError names where oldText matched. Instruction-phrased, like the other hints.
func ambiguousError(rawOld string, lines []int) error {
	shown := lines
	if len(shown) > 10 {
		shown = shown[:10]
	}
	parts := make([]string, len(shown))
	for i, l := range shown {
		parts[i] = strconv.Itoa(l)
	}
	where := strings.Join(parts, ", ")
	if len(shown) < len(lines) {
		where += ", …"
	}
	return fmt.Errorf("%w: %d places (lines %s). NOTHING was changed.\n%s\n\n"+
		"Add surrounding lines to oldText so it picks out one of them, or set replaceAll: true on this edit to change all %d",
		ErrEditAmbiguous, len(lines), where, rawOld, len(lines))
}

// noMatchError wraps ErrEditNoMatch, appending the closest matching block if found.
func noMatchError(content string, candidate matchCandidate, rawOld string, threshold *float64) error {
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
