// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

// The fuzzy fallback for edit_file: find the block nearest to oldText, for a similarity threshold or a no-match hint.

import (
	"sort"
	"strings"
)

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

// lineEditDistance is Levenshtein over pre-trimmed lines; prev and cur are caller-owned scratch, so a sweep allocates nothing.
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
