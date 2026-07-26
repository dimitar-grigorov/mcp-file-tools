package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Truncated results must be the same set, in the same order, on every run.
func TestHandleGrep_TruncationIsDeterministic(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	for i := 0; i < 40; i++ {
		var b strings.Builder
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&b, "needle %02d-%02d\n", i, j)
		}
		path := filepath.Join(tempDir, fmt.Sprintf("f%02d.txt", i))
		if err := os.WriteFile(path, []byte(b.String()), 0644); err != nil {
			t.Fatal(err)
		}
	}

	var first string
	for run := 0; run < 20; run++ {
		_, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
			Pattern:    "needle",
			Paths:      []string{tempDir},
			MaxMatches: 25,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !output.Truncated {
			t.Fatalf("run %d: expected truncated results", run)
		}
		if output.TotalMatches != 25 {
			t.Fatalf("run %d: expected 25 matches, got %d", run, output.TotalMatches)
		}
		var b strings.Builder
		for _, m := range output.Matches {
			fmt.Fprintf(&b, "%s:%d:%s\n", filepath.Base(m.Path), m.Line, m.Text)
		}
		got := b.String()
		if run == 0 {
			first = got
			continue
		}
		if got != first {
			t.Fatalf("run %d differs from run 0:\n--- run 0 ---\n%s\n--- run %d ---\n%s", run, first, run, got)
		}
	}

	// Input order is file order, so the surviving matches are the first ones.
	var want strings.Builder
	for i := 0; i < 25; i++ {
		fmt.Fprintf(&want, "f%02d.txt:%d:needle %02d-%02d\n", i/10, i%10+1, i/10, i%10)
	}
	if first != want.String() {
		t.Errorf("truncated set is not the first 25 matches in file order:\ngot:\n%s\nwant:\n%s", first, want.String())
	}
}
