// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// plantedSource hides a block at plantAt and returns it with one line altered.
func plantedSource(totalLines, plantAt, blockLines int) (content, oldText string) {
	lines := make([]string, totalLines)
	for i := range lines {
		lines[i] = fmt.Sprintf("  Filler%d := %d;", i, i)
	}
	block := make([]string, blockLines)
	for i := range block {
		block[i] = fmt.Sprintf("  Planted%d := CalculateSomething(%d);", i, i)
		lines[plantAt+i] = block[i]
	}
	block[blockLines/2] = "  Planted999 := CalculateSomethingElse(999);"
	return strings.Join(lines, "\n"), strings.Join(block, "\n")
}

// Past the budget only a shortlist of starts is searched; still has to land right.
func TestClosestCandidate_NarrowedSearchFindsPlantedBlock(t *testing.T) {
	for _, tc := range []struct {
		name       string
		total      int
		plantAt    int
		blockLines int
	}{
		{"exact search, small block", 400, 150, 6},
		{"narrowed, 60-line block", 3000, 1800, 60},
		{"narrowed, 200-line block", 4000, 2500, 200},
		{"narrowed, block at file start", 3000, 0, 80},
		{"narrowed, block at file end", 3000, 2900, 90},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content, oldText := plantedSource(tc.total, tc.plantAt, tc.blockLines)

			got := closestCandidate(content, oldText)

			if got.start != tc.plantAt {
				t.Errorf("start = %d, want %d (score %.3f, distance %d)",
					got.start, tc.plantAt, got.score, got.distance)
			}
			if got.distance != 1 {
				t.Errorf("distance = %d, want 1", got.distance)
			}
			// One line of blockLines differs.
			if want := 1 - 1/float64(tc.blockLines); got.score < want {
				t.Errorf("score = %.3f, want >= %.3f", got.score, want)
			}
		})
	}
}

// naiveClosestCandidate is the pre-optimisation version, kept as the oracle.
func naiveClosestCandidate(content, oldText string) matchCandidate {
	contentLines := strings.Split(content, "\n")
	oldLines := strings.Split(oldText, "\n")
	if len(contentLines) == 0 || len(oldLines) == 0 {
		return matchCandidate{start: -1}
	}
	distance := func(a, b []string) int {
		previous := make([]int, len(b)+1)
		for j := range previous {
			previous[j] = j
		}
		for i, left := range a {
			current := make([]int, len(b)+1)
			current[0] = i + 1
			for j, right := range b {
				cost := 1
				if strings.TrimSpace(left) == strings.TrimSpace(right) {
					cost = 0
				}
				current[j+1] = min(previous[j+1]+1, current[j]+1, previous[j]+cost)
			}
			previous = current
		}
		return previous[len(b)]
	}

	best := matchCandidate{start: -1, score: -1}
	maxLines := min(len(contentLines), len(oldLines)*2)
	for size := 1; size <= maxLines; size++ {
		for i := 0; i <= len(contentLines)-size; i++ {
			d := distance(oldLines, contentLines[i:i+size])
			if d > len(oldLines) {
				continue
			}
			score := 1 - float64(d)/float64(max(len(oldLines), size))
			if score > best.score || score == best.score && d < best.distance {
				best = matchCandidate{start: i, lines: size, distance: d, score: score}
			}
		}
	}
	return best
}

// Diverging would mean a fuzzy edit rewrites a different part of the file.
func TestClosestCandidate_MatchesNaiveImplementation(t *testing.T) {
	// Deterministic pseudo-random corpus; no seed dependency across Go versions.
	next := uint32(12345)
	rnd := func(n int) int {
		next = next*1664525 + 1013904223
		return int(next>>16) % n
	}
	vocab := []string{
		"begin", "end;", "  x := 1;", "  y := 2;", "if a then", "  Inc(i);",
		"", "   ", "  // comment", "procedure Foo;", "  Result := nil;",
	}
	build := func(n int) string {
		lines := make([]string, n)
		for i := range lines {
			lines[i] = vocab[rnd(len(vocab))]
		}
		return strings.Join(lines, "\n")
	}

	for i := range 400 {
		content := build(1 + rnd(40))
		oldText := build(1 + rnd(6))

		got := closestCandidate(content, oldText)
		want := naiveClosestCandidate(content, oldText)

		if got != want {
			t.Fatalf("case %d diverged\ncontent=%q\noldText=%q\ngot  =%+v\nwant =%+v",
				i, content, oldText, got, want)
		}
	}
}

// Below the budget the search must stay exhaustive.
func TestCandidateSearchSpace_ExactBelowBudget(t *testing.T) {
	content := trimLines(strings.Split(syntheticSource(200), "\n"))
	old := trimLines(strings.Split(syntheticSource(4), "\n"))

	starts, sizes := candidateSearchSpace(content, old, min(len(content), len(old)*2))

	if len(starts) != len(content) {
		t.Errorf("starts = %d, want every position (%d)", len(starts), len(content))
	}
	if len(sizes) != min(len(content), len(old)*2) {
		t.Errorf("sizes = %d, want every size up to %d", len(sizes), len(old)*2)
	}
}

func TestPromisingStarts_RanksOverlapHighest(t *testing.T) {
	content := []string{"a", "b", "zzz", "yyy", "xxx", "a", "b", "c"}
	old := []string{"a", "b", "c"}

	starts := promisingStarts(content, old, 3)

	if len(starts) != 3 {
		t.Fatalf("got %d starts, want 3", len(starts))
	}
	// The window at 5 holds all three lines; it must survive the cut.
	if !slices.Contains(starts, 5) {
		t.Errorf("starts = %v, want the full match at 5 kept", starts)
	}
	for i := 1; i < len(starts); i++ {
		if starts[i-1] >= starts[i] {
			t.Errorf("starts = %v, want ascending", starts)
		}
	}
}
