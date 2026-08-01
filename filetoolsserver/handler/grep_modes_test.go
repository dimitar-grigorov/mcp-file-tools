// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// grepModeFixture writes three files: two match, one does not.
func grepModeFixture(t *testing.T) (string, *Handler) {
	t.Helper()
	tempDir := t.TempDir()
	files := map[string]string{
		"a.pas": "unit A;\nvar x: TFormMain;\nbegin\n  TFormMain.Show;\nend.\n",
		"b.pas": "unit B;\nvar y: TFormMain;\nend.\n",
		"c.pas": "unit C;\nvar z: Integer;\nend.\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return tempDir, NewHandler([]string{tempDir})
}

func TestHandleGrep_OutputModeContentIsDefault(t *testing.T) {
	tempDir, h := grepModeFixture(t)

	_, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern: "TFormMain",
		Paths:   []string{tempDir},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Matches) != 3 || output.TotalMatches != 3 {
		t.Fatalf("expected 3 matches, got %d (%d)", len(output.Matches), output.TotalMatches)
	}
	if output.FilesMatched != 2 || output.FilesSearched != 3 {
		t.Errorf("filesMatched=%d filesSearched=%d, want 2 and 3", output.FilesMatched, output.FilesSearched)
	}
	if output.Matches[0].Text != "var x: TFormMain;" {
		t.Errorf("content mode must return the whole line, got %q", output.Matches[0].Text)
	}
	if len(output.Files) != 0 || len(output.Counts) != 0 {
		t.Error("content mode must not populate files or counts")
	}
}

func TestHandleGrep_OutputModeFilesWithMatches(t *testing.T) {
	tempDir, h := grepModeFixture(t)

	_, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern:       "TFormMain",
		Paths:         []string{tempDir},
		OutputMode:    outputModeFiles,
		ContextBefore: 2, // meaningless here: silently ignored
		ContextAfter:  2,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.pas", "b.pas"}
	if len(output.Files) != len(want) {
		t.Fatalf("expected %d files, got %v", len(want), output.Files)
	}
	for i, name := range want {
		if got := filepath.Base(output.Files[i]); got != name {
			t.Errorf("file %d: got %q, want %q", i, got, name)
		}
	}
	if len(output.Matches) != 0 {
		t.Errorf("files_with_matches must not return match text, got %d matches", len(output.Matches))
	}
	// a.pas has two matching lines but the search stops at the first.
	if output.TotalMatches != 2 {
		t.Errorf("totalMatches=%d, want one per returned path", output.TotalMatches)
	}
	if output.FilesMatched != 2 {
		t.Errorf("filesMatched=%d, want 2", output.FilesMatched)
	}
}

func TestHandleGrep_OutputModeCount(t *testing.T) {
	tempDir, h := grepModeFixture(t)

	_, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern:    "TFormMain",
		Paths:      []string{tempDir},
		OutputMode: outputModeCount,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Counts) != 2 {
		t.Fatalf("expected 2 count entries, got %v", output.Counts)
	}
	if filepath.Base(output.Counts[0].Path) != "a.pas" || output.Counts[0].Count != 2 {
		t.Errorf("first entry: got %+v, want a.pas with count 2", output.Counts[0])
	}
	if filepath.Base(output.Counts[1].Path) != "b.pas" || output.Counts[1].Count != 1 {
		t.Errorf("second entry: got %+v, want b.pas with count 1", output.Counts[1])
	}
	if output.TotalMatches != 3 {
		t.Errorf("totalMatches=%d, want the sum of per-file counts (3)", output.TotalMatches)
	}
	if len(output.Matches) != 0 {
		t.Errorf("count mode must not return match text, got %d matches", len(output.Matches))
	}
}

// files_with_matches must not return context even when a caller asks for it.
func TestHandleGrep_NonContentModesDropContext(t *testing.T) {
	tempDir, h := grepModeFixture(t)

	for _, mode := range []string{outputModeFiles, outputModeCount} {
		_, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
			Pattern:       "TFormMain",
			Paths:         []string{tempDir},
			OutputMode:    mode,
			ContextBefore: 1,
			ContextAfter:  1,
		})
		if err != nil {
			t.Fatal(err)
		}
		blob, err := json.Marshal(output)
		if err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{`"before"`, `"after"`, `"text"`} {
			if strings.Contains(string(blob), field) {
				t.Errorf("%s: response carries %s: %s", mode, field, blob)
			}
		}
	}
}

func TestHandleGrep_InvalidOutputMode(t *testing.T) {
	tempDir, h := grepModeFixture(t)

	result, _, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern:    "TFormMain",
		Paths:      []string{tempDir},
		OutputMode: "paths",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected an error for an unknown outputMode")
	}
}

func TestHandleGrep_MatchesOnly(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	body := "Version=1.2.3 Build=4.5.6\nnothing here\nVersion=7.8.9\n"
	if err := os.WriteFile(filepath.Join(tempDir, "v.ini"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}

	_, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern:     `\d+\.\d+\.\d+`,
		Paths:       []string{tempDir},
		MatchesOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1.2.3", "4.5.6", "7.8.9"}
	if len(output.Matches) != len(want) {
		t.Fatalf("expected %d matches, got %d: %+v", len(want), len(output.Matches), output.Matches)
	}
	for i, text := range want {
		if output.Matches[i].Text != text {
			t.Errorf("match %d: got %q, want %q", i, output.Matches[i].Text, text)
		}
	}
	// Columns stay 1-indexed against the original line.
	if output.Matches[1].Column != 21 {
		t.Errorf("second match column=%d, want 21", output.Matches[1].Column)
	}
	if output.Matches[1].Line != 1 {
		t.Errorf("second match line=%d, want 1", output.Matches[1].Line)
	}
}

// offset pages past maxMatches, which was previously a dead end.
func TestHandleGrep_OffsetPagesPastMaxMatches(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	for i := 0; i < 4; i++ {
		var body string
		for j := 0; j < 10; j++ {
			body += fmt.Sprintf("needle %02d-%02d\n", i, j)
		}
		if err := os.WriteFile(filepath.Join(tempDir, fmt.Sprintf("f%d.txt", i)), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}

	var paged []string
	for offset := 0; ; {
		_, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
			Pattern:    "needle",
			Paths:      []string{tempDir},
			MaxMatches: 7,
			Offset:     offset,
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range output.Matches {
			paged = append(paged, m.Text)
		}
		if !output.Truncated {
			if output.NextOffset != 0 {
				t.Errorf("nextOffset should be unset on the last page, got %d", output.NextOffset)
			}
			break
		}
		if output.NextOffset != offset+7 {
			t.Fatalf("nextOffset=%d, want %d", output.NextOffset, offset+7)
		}
		offset = output.NextOffset
		if offset > 200 {
			t.Fatal("paging did not terminate")
		}
	}

	if len(paged) != 40 {
		t.Fatalf("paging returned %d matches, want 40", len(paged))
	}
	for i := 0; i < 40; i++ {
		want := fmt.Sprintf("needle %02d-%02d", i/10, i%10)
		if paged[i] != want {
			t.Fatalf("paged[%d]=%q, want %q", i, paged[i], want)
		}
	}
}

// A single file overflowing the page must still be pageable.
func TestHandleGrep_OffsetWithinOneFile(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	var body string
	for j := 0; j < 30; j++ {
		body += fmt.Sprintf("hit %02d\n", j)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "one.txt"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}

	_, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern:    "hit",
		Paths:      []string{tempDir},
		MaxMatches: 5,
		Offset:     25,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Matches) != 5 {
		t.Fatalf("expected 5 matches, got %d", len(output.Matches))
	}
	if output.Matches[0].Text != "hit 25" {
		t.Errorf("first match on the page is %q, want %q", output.Matches[0].Text, "hit 25")
	}
	if output.Truncated {
		t.Error("the last page must not report truncation")
	}
}

// maxMatches caps paths in files_with_matches, not match objects.
func TestHandleGrep_FilesWithMatchesRespectsMaxMatches(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	for i := 0; i < 6; i++ {
		path := filepath.Join(tempDir, fmt.Sprintf("f%d.txt", i))
		if err := os.WriteFile(path, []byte("needle\nneedle\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	_, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern:    "needle",
		Paths:      []string{tempDir},
		OutputMode: outputModeFiles,
		MaxMatches: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Files) != 4 {
		t.Fatalf("expected 4 paths, got %d", len(output.Files))
	}
	if !output.Truncated || output.NextOffset != 4 {
		t.Errorf("truncated=%v nextOffset=%d, want true and 4", output.Truncated, output.NextOffset)
	}

	_, page2, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern:    "needle",
		Paths:      []string{tempDir},
		OutputMode: outputModeFiles,
		MaxMatches: 4,
		Offset:     4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Files) != 2 || page2.Truncated {
		t.Errorf("page 2: %d files, truncated=%v; want 2 and false", len(page2.Files), page2.Truncated)
	}
}

// The whole point of files_with_matches is a smaller response.
func TestHandleGrep_FilesWithMatchesIsSmaller(t *testing.T) {
	tempDir, h := grepModeFixture(t)

	size := func(mode string) int {
		_, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
			Pattern:    "TFormMain",
			Paths:      []string{tempDir},
			OutputMode: mode,
		})
		if err != nil {
			t.Fatal(err)
		}
		blob, err := json.Marshal(output)
		if err != nil {
			t.Fatal(err)
		}
		return len(blob)
	}

	content, paths := size(outputModeContent), size(outputModeFiles)
	if paths >= content {
		t.Errorf("files_with_matches (%d bytes) is not smaller than content (%d bytes)", paths, content)
	}
	t.Logf("content=%d bytes files_with_matches=%d bytes", content, paths)
}
