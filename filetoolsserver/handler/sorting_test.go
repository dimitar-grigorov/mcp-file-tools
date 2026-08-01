// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// sortFixture writes files whose name, mtime and size orders all differ.
//
//	name:  a.txt  b.txt  c.txt
//	mtime: c.txt (newest) b.txt a.txt
//	size:  b.txt (largest) a.txt c.txt
func sortFixture(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	base := time.Now().Add(-72 * time.Hour)
	files := []struct {
		name string
		size int
		age  time.Duration
	}{
		{"a.txt", 20, 0},
		{"b.txt", 30, 24 * time.Hour},
		{"c.txt", 10, 48 * time.Hour},
	}
	for _, f := range files {
		path := filepath.Join(tempDir, f.name)
		if err := os.WriteFile(path, make([]byte, f.size), 0644); err != nil {
			t.Fatal(err)
		}
		stamp := base.Add(f.age)
		if err := os.Chtimes(path, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	return tempDir
}

func baseNames(paths []string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = filepath.Base(p)
	}
	return out
}

func equalNames(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestHandleSearchFiles_SortBy(t *testing.T) {
	tempDir := sortFixture(t)
	h := NewHandler([]string{tempDir})

	tests := []struct {
		sortBy  string
		reverse bool
		want    []string
	}{
		{"", false, []string{"a.txt", "b.txt", "c.txt"}}, // default is name
		{sortByName, false, []string{"a.txt", "b.txt", "c.txt"}},
		{sortByName, true, []string{"c.txt", "b.txt", "a.txt"}},
		{sortByMtime, false, []string{"c.txt", "b.txt", "a.txt"}}, // newest first
		{sortByMtime, true, []string{"a.txt", "b.txt", "c.txt"}},
		{sortBySize, false, []string{"b.txt", "a.txt", "c.txt"}}, // largest first
		{sortBySize, true, []string{"c.txt", "a.txt", "b.txt"}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_reverse=%v", tt.sortBy, tt.reverse), func(t *testing.T) {
			_, output, err := h.HandleSearchFiles(context.Background(), nil, SearchFilesInput{
				Path:    tempDir,
				Pattern: "*.txt",
				SortBy:  tt.sortBy,
				Reverse: tt.reverse,
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := baseNames(output.Files); !equalNames(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleListDirectory_SortBy(t *testing.T) {
	tempDir := sortFixture(t)
	h := NewHandler([]string{tempDir})

	tests := []struct {
		sortBy  string
		reverse bool
		want    []string
	}{
		{"", false, []string{"a.txt", "b.txt", "c.txt"}},
		{sortByMtime, false, []string{"c.txt", "b.txt", "a.txt"}},
		{sortByMtime, true, []string{"a.txt", "b.txt", "c.txt"}},
		{sortBySize, false, []string{"b.txt", "a.txt", "c.txt"}},
		{sortByName, true, []string{"c.txt", "b.txt", "a.txt"}},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_reverse=%v", tt.sortBy, tt.reverse), func(t *testing.T) {
			_, output, err := h.HandleListDirectory(context.Background(), nil, ListDirectoryInput{
				Path:    tempDir,
				SortBy:  tt.sortBy,
				Reverse: tt.reverse,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !equalNames(output.Files, tt.want) {
				t.Errorf("got %v, want %v", output.Files, tt.want)
			}
		})
	}
}

// Directories keep their [DIR] prefix but sort on the bare name.
func TestHandleListDirectory_SortKeepsDirPrefix(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	if err := os.Mkdir(filepath.Join(tempDir, "b_dir"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(tempDir, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	_, output, err := h.HandleListDirectory(context.Background(), nil, ListDirectoryInput{Path: tempDir})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.txt", "[DIR] b_dir", "c.txt"}
	if !equalNames(output.Files, want) {
		t.Errorf("got %v, want %v", output.Files, want)
	}
}

func TestSortBy_Invalid(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	result, _, err := h.HandleSearchFiles(context.Background(), nil, SearchFilesInput{
		Path: tempDir, Pattern: "*", SortBy: "created",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("search_files: expected an error for an unknown sortBy")
	}

	result, _, err = h.HandleListDirectory(context.Background(), nil, ListDirectoryInput{
		Path: tempDir, SortBy: "created",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("list_directory: expected an error for an unknown sortBy")
	}
}

// countingStat records how often the sort keys are read off an entry.
func countingStat(n *int64) statFunc {
	return func(e fs.DirEntry) (int64, int64) {
		atomic.AddInt64(n, 1)
		return statEntry(e)
	}
}

// Sorting by name must not stat anything: on Linux/macOS Info() is an lstat per
// entry, so the default path has to stay as cheap as an unsorted walk.
func TestSortByName_MakesNoInfoCalls(t *testing.T) {
	tempDir := sortFixture(t)
	h := NewHandler([]string{tempDir})

	run := func(sortBy string) int64 {
		var calls int64
		files, _, err := searchFiles(context.Background(), tempDir, searchOptions{
			pattern:     "*.txt",
			allowedDirs: h.ResolvedAllowedDirs(),
			maxResults:  defaultMaxResults,
			sortBy:      sortBy,
			stat:        countingStat(&calls),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 3 {
			t.Fatalf("%s: expected 3 files, got %d", sortBy, len(files))
		}
		return calls
	}

	if n := run(sortByName); n != 0 {
		t.Errorf("search_files sortBy=name made %d Info() calls, want 0", n)
	}
	if n := run(sortByMtime); n != 3 {
		t.Errorf("search_files sortBy=mtime made %d Info() calls, want one per match", n)
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	var calls int64
	if _, err := listDirectoryEntries(entries, "*", sortByName, false, countingStat(&calls)); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Errorf("list_directory sortBy=name made %d Info() calls, want 0", calls)
	}
	if _, err := listDirectoryEntries(entries, "*", sortBySize, false, countingStat(&calls)); err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Errorf("list_directory sortBy=size made %d Info() calls, want one per entry", calls)
	}
}

// A file whose Info() fails must not sink the call; it keeps its place, sorted last.
func TestSort_InfoErrorKeepsEntry(t *testing.T) {
	tempDir := sortFixture(t)
	h := NewHandler([]string{tempDir})

	failing := func(e fs.DirEntry) (int64, int64) {
		if e.Name() == "b.txt" {
			return 0, 0 // what statEntry returns when Info() errors
		}
		return statEntry(e)
	}
	files, truncated, err := searchFiles(context.Background(), tempDir, searchOptions{
		pattern:     "*.txt",
		allowedDirs: h.ResolvedAllowedDirs(),
		maxResults:  defaultMaxResults,
		sortBy:      sortByMtime,
		stat:        failing,
	})
	if err != nil {
		t.Fatal(err)
	}
	if truncated {
		t.Error("unexpected truncation")
	}
	want := []string{"c.txt", "a.txt", "b.txt"} // b.txt has no mtime, so it lands last
	if got := baseNames(files); !equalNames(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// Ranking has to happen before the cap: mtime must return the newest maxResults
// files, not the first maxResults in walk order sorted among themselves.
func TestHandleSearchFiles_SortBeforeTruncation(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Walk order is lexical, so the newest files are visited last on purpose.
	base := time.Now().Add(-100 * time.Hour)
	for i := 0; i < 20; i++ {
		path := filepath.Join(tempDir, fmt.Sprintf("f%02d.txt", i))
		if err := os.WriteFile(path, make([]byte, i+1), 0644); err != nil {
			t.Fatal(err)
		}
		stamp := base.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(path, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}

	for _, tt := range []struct {
		sortBy string
		want   []string
	}{
		{sortByMtime, []string{"f19.txt", "f18.txt", "f17.txt"}},
		{sortBySize, []string{"f19.txt", "f18.txt", "f17.txt"}},
	} {
		t.Run(tt.sortBy, func(t *testing.T) {
			_, output, err := h.HandleSearchFiles(context.Background(), nil, SearchFilesInput{
				Path:       tempDir,
				Pattern:    "*.txt",
				MaxResults: 3,
				SortBy:     tt.sortBy,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !output.Truncated {
				t.Error("expected truncated to be true")
			}
			if got := baseNames(output.Files); !equalNames(got, tt.want) {
				t.Errorf("got %v, want %v — the cap was applied before the ranking", got, tt.want)
			}
		})
	}

	// Reverse picks the other end of the same ranking.
	_, output, err := h.HandleSearchFiles(context.Background(), nil, SearchFilesInput{
		Path: tempDir, Pattern: "*.txt", MaxResults: 3, SortBy: sortByMtime, Reverse: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := baseNames(output.Files), []string{"f00.txt", "f01.txt", "f02.txt"}; !equalNames(got, want) {
		t.Errorf("reverse: got %v, want %v", got, want)
	}
}

// sortBy=name keeps the walk's early stop, so the cap comes first there.
func TestHandleSearchFiles_NameTruncatesInWalkOrder(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	for i := 0; i < 20; i++ {
		if err := os.WriteFile(filepath.Join(tempDir, fmt.Sprintf("f%02d.txt", i)), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	_, output, err := h.HandleSearchFiles(context.Background(), nil, SearchFilesInput{
		Path: tempDir, Pattern: "*.txt", MaxResults: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !output.Truncated {
		t.Error("expected truncated to be true")
	}
	if got, want := baseNames(output.Files), []string{"f00.txt", "f01.txt", "f02.txt"}; !equalNames(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// The bounded heap must not lose or duplicate entries as it evicts.
func TestTopN_KeepsTheBestEntries(t *testing.T) {
	top := newTopN(3, sortBySize, false)
	for i := 0; i < 50; i++ {
		top.add(sortEntry{key: fmt.Sprintf("f%02d", i), value: fmt.Sprintf("f%02d", i), size: int64(i)})
	}
	if !top.truncated() {
		t.Error("expected truncated")
	}
	if got, want := top.values(), []string{"f49", "f48", "f47"}; !equalNames(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// Fewer entries than the limit is not truncation.
	small := newTopN(10, sortByName, false)
	small.add(sortEntry{key: "b", value: "b"})
	small.add(sortEntry{key: "a", value: "a"})
	if small.truncated() {
		t.Error("did not expect truncation below the limit")
	}
	if got, want := small.values(), []string{"a", "b"}; !equalNames(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
