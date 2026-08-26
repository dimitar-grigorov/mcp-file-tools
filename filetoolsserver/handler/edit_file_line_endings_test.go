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

// A CRLF file that also contains one bare LF (e.g. embedded in a multi-line
// string literal, far from the edited line) must stay CRLF after edit_file
// touches an unrelated line. Regression test for the bug where edit_file
// passed DetectLineEndings' raw "mixed"/"none" Style straight into
// ConvertLineEndings, which only special-cases "crlf" — every other target,
// including "mixed", silently fell through to the LF branch and stripped
// every CRLF in the file on write-back.
func TestHandleEditFile_MixedLineEndingsRepairToDominantStyle(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "mixed.txt")
	// 3 CRLF-terminated lines, one bare LF embedded inside a "string literal"
	// on an unrelated line: file-wide style is CRLF-dominant "mixed", not pure CRLF.
	original := "line1\r\nliteral := 'a' + Chr(10) + 'b'\nline3\r\nMARKER_OLD\r\n"
	if err := os.WriteFile(testFile, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	input := EditFileInput{
		Path:  testFile,
		Edits: []EditOperation{{OldText: "MARKER_OLD", NewText: "MARKER_NEW"}},
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error result: %+v", result.Content)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	want := "line1\r\nliteral := 'a' + Chr(10) + 'b'\r\nline3\r\nMARKER_NEW\r\n"
	if string(got) != want {
		t.Errorf("edit_file corrupted line endings on a mixed-but-CRLF-dominant file\ngot:  %q\nwant: %q", got, want)
	}

	crlf := strings.Count(string(got), "\r\n")
	if crlf != 4 {
		t.Errorf("expected 4 CRLF line endings preserved, got %d (content: %q)", crlf, got)
	}
}

// A file that's LF-dominant (more bare LF than CRLF) should repair to LF,
// not CRLF — dominantLineEnding picks the more common style either way.
func TestHandleEditFile_MixedLineEndingsRepairToLFWhenDominant(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "mixed_lf_dominant.txt")
	original := "line1\nline2\nline3\r\nMARKER_OLD\n"
	if err := os.WriteFile(testFile, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	input := EditFileInput{
		Path:  testFile,
		Edits: []EditOperation{{OldText: "MARKER_OLD", NewText: "MARKER_NEW"}},
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error result: %+v", result.Content)
	}

	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	want := "line1\nline2\nline3\nMARKER_NEW\n"
	if string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
