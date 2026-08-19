// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/encoding"
	"golang.org/x/text/encoding/charmap"
)

// A pin says what the tree holds; a file outside it is the case worth reporting.
func TestDetectionCandidates_ReportsAFileThatFitsNone(t *testing.T) {
	dir := t.TempDir()

	greek, err := charmap.Windows1253.NewEncoder().Bytes([]byte(strings.Repeat("Καλημέρα κόσμε, δοκιμαστικό κείμενο.\r\n", 4)))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "greek.txt")
	if err := os.WriteFile(path, greek, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		pinned  []string
		wantOdd bool
	}{
		{"nothing configured stays quiet", nil, false},
		{"pinned set the file does not fit", []string{"utf-8"}, true},
		{"pinned set the file fits", []string{"utf-8", "windows-1253"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler([]string{dir})
			h.config.DetectionCandidates = tt.pinned
			if err := encoding.SetDetectionCandidates(tt.pinned); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = encoding.SetDetectionCandidates(nil) })

			_, out, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{Path: path})
			if err != nil {
				t.Fatalf("read failed: %v", err)
			}
			if got := strings.Contains(out.Hint, "ODD ENCODING"); got != tt.wantOdd {
				t.Errorf("ODD ENCODING in hint = %v, want %v (hint: %q)", got, tt.wantOdd, out.Hint)
			}
		})
	}
}
