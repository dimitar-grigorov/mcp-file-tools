// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// A result set that exactly fills maxResults is complete, not truncated.
func TestSearchFiles_ExactFitIsNotTruncated(t *testing.T) {
	dir := t.TempDir()
	for i := range 3 {
		p := filepath.Join(dir, fmt.Sprintf("f%d.txt", i))
		if err := os.WriteFile(p, nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	h := NewHandler([]string{dir})

	for _, tc := range []struct {
		maxResults int
		wantFiles  int
		wantTrunc  bool
	}{
		{maxResults: 3, wantFiles: 3, wantTrunc: false},
		{maxResults: 2, wantFiles: 2, wantTrunc: true},
		{maxResults: 5, wantFiles: 3, wantTrunc: false},
	} {
		_, out, err := h.HandleSearchFiles(context.Background(), nil, SearchFilesInput{
			Path: dir, Pattern: "*.txt", MaxResults: tc.maxResults,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Files) != tc.wantFiles || out.Truncated != tc.wantTrunc {
			t.Errorf("maxResults=%d: got %d files truncated=%v, want %d truncated=%v",
				tc.maxResults, len(out.Files), out.Truncated, tc.wantFiles, tc.wantTrunc)
		}
	}
}
