// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An unresolvable encoding used to decode silently wrong; the search still runs,
// but the response now says the name was ignored.
func TestHandleGrep_UnknownEncodingHints(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello world\n"), 0644); err != nil {
		t.Fatal(err)
	}
	h := NewHandler([]string{dir})

	res, out, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern: "hello", Paths: []string{dir}, Encoding: "cp1251x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("grep should still search: %+v", res.Content)
	}
	if len(out.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(out.Matches))
	}
	if !strings.Contains(out.Hint, "cp1251x") {
		t.Errorf("hint does not name the bad encoding: %q", out.Hint)
	}
}

// A known name, alias included, must not produce a hint.
func TestHandleGrep_KnownEncodingHasNoHint(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	h := NewHandler([]string{dir})

	for _, name := range []string{"", "cp1251", "utf8", "utf16le"} {
		_, out, err := h.HandleGrep(context.Background(), nil, GrepInput{
			Pattern: "hello", Paths: []string{dir}, Encoding: name,
		})
		if err != nil {
			t.Fatal(err)
		}
		if out.Hint != "" {
			t.Errorf("encoding %q produced a hint: %q", name, out.Hint)
		}
	}
}

// filesSearched counted every collected file even when a full page stopped the walk.
func TestHandleGrep_FilesSearchedCountsOnlyWhatItRead(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt", "d.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("match\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	h := NewHandler([]string{dir})

	_, full, err := h.HandleGrep(context.Background(), nil, GrepInput{Pattern: "match", Paths: []string{dir}})
	if err != nil {
		t.Fatal(err)
	}
	if full.FilesSearched != 4 {
		t.Errorf("un-truncated FilesSearched = %d, want 4", full.FilesSearched)
	}

	_, page, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern: "match", Paths: []string{dir}, MaxMatches: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !page.Truncated {
		t.Fatal("expected truncation with maxMatches=2 over 4 matching files")
	}
	if page.FilesSearched > 3 {
		t.Errorf("FilesSearched = %d, want at most 3 once the page filled", page.FilesSearched)
	}
}
