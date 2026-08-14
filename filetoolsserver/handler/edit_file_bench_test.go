// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"fmt"
	"strings"
	"testing"
)

func syntheticSource(lines int) string {
	var b strings.Builder
	for i := range lines {
		fmt.Fprintf(&b, "  procedure DoThing%d(const Value: Integer);\n", i)
	}
	return b.String()
}

// closestCandidate runs on every failed edit, so this is the cost of a mistake.
func BenchmarkClosestCandidate(b *testing.B) {
	oldText := "  procedure Missing(const Value: Integer);\n  begin\n  end;"
	for _, lines := range []int{500, 2000, 10000} {
		content := syntheticSource(lines)
		b.Run(fmt.Sprintf("%dlines", lines), func(b *testing.B) {
			for b.Loop() {
				closestCandidate(content, oldText)
			}
		})
	}
}

// Cost grows with the pasted block, not just with the file.
func BenchmarkClosestCandidate_LargeOldText(b *testing.B) {
	content := syntheticSource(2000)
	for _, oldLines := range []int{10, 50, 200} {
		var sb strings.Builder
		for i := range oldLines {
			fmt.Fprintf(&sb, "  procedure Absent%d(const V: Integer);\n", i)
		}
		oldText := sb.String()
		b.Run(fmt.Sprintf("%dlines", oldLines), func(b *testing.B) {
			for b.Loop() {
				closestCandidate(content, oldText)
			}
		})
	}
}

func BenchmarkApplyEdits_NoMatch(b *testing.B) {
	content := syntheticSource(2000)
	edits := []EditOperation{{OldText: "procedure NotPresent;", NewText: "x"}}
	for b.Loop() {
		_, _, _ = applyEdits(content, edits)
	}
}
