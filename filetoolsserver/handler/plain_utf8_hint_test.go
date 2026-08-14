// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"
)

func TestPlainUTF8Hint_OnlyOncePerFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "plain.txt")
	os.WriteFile(p, []byte("hello world\n"), 0644)
	h := NewHandler([]string{dir})
	ctx := context.Background()

	_, first, _ := h.HandleReadTextFile(ctx, nil, ReadTextFileInput{Path: p})
	if first.Hint == "" {
		t.Fatal("expected a hint on the first read of a plain utf-8 file")
	}

	_, second, _ := h.HandleReadTextFile(ctx, nil, ReadTextFileInput{Path: p})
	if second.Hint != "" {
		t.Errorf("expected no hint on the second read, got %q", second.Hint)
	}
}

// The seen-set is keyed by path and never evicts, so it is capped; past the cap
// the hint stops rather than the map growing for the life of the process.
func TestPlainUTF8Hint_StopsAtCap(t *testing.T) {
	h := NewHandler([]string{t.TempDir()})

	for i := range plainUTF8HintCap {
		if h.plainUTF8HintFor(fmt.Sprintf("/f%d.txt", i), "utf-8", false) == "" {
			t.Fatalf("path %d got no hint, still under the cap", i)
		}
	}
	if got := h.plainUTF8HintFor("/one-too-many.txt", "utf-8", false); got != "" {
		t.Errorf("hint past the cap: %q", got)
	}
	if n := h.plainUTF8Count.Load(); n != plainUTF8HintCap {
		t.Errorf("counted %d, want %d", n, plainUTF8HintCap)
	}
}

func TestPlainUTF8Hint_SilentForNonUTF8(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cyr.txt")
	body, err := charmap.Windows1251.NewEncoder().Bytes([]byte("Привет, мир! Это кириллический текст."))
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(p, body, 0644)
	h := NewHandler([]string{dir})

	_, out, _ := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{Path: p})
	if out.DetectedEncoding == "utf-8" {
		t.Skip("fixture did not detect as non-utf-8")
	}
	if out.Hint != "" {
		t.Errorf("expected no hint for a %s file, got %q", out.DetectedEncoding, out.Hint)
	}
}

func TestPlainUTF8Hint_SilentWithBOM(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bom.txt")
	os.WriteFile(p, append([]byte("\xef\xbb\xbf"), []byte("hello\n")...), 0644)
	h := NewHandler([]string{dir})

	_, out, _ := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{Path: p})
	if out.Hint != "" {
		t.Errorf("expected no hint for a file with a BOM, got %q", out.Hint)
	}
}

// The original ask: surface an already-mixed file to the agent, for free, on read.
func TestReadHint_ReportsMixedLineEndings(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "mixed.txt")
	os.WriteFile(p, []byte("a\r\nb\nc\r\n"), 0644)
	h := NewHandler([]string{dir})

	_, out, _ := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{Path: p})
	if !strings.Contains(out.Hint, "MIXED line endings") {
		t.Errorf("expected a mixed line endings hint, got %q", out.Hint)
	}
}

// write_file reports when it normalised, so the behaviour is not silent.
func TestWriteHint_ReportsNormalisation(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "crlf.txt")
	os.WriteFile(p, []byte("a\r\nb\r\n"), 0644)
	h := NewHandler([]string{dir})

	_, out, _ := h.HandleWriteFile(context.Background(), nil, WriteFileInput{Path: p, Content: "a\nb CHANGED\n"})
	if out.LineEndings != "crlf" {
		t.Errorf("expected lineEndings=crlf in output, got %q", out.LineEndings)
	}
	if !strings.Contains(out.Message, "normalised to CRLF") {
		t.Errorf("expected the message to report normalisation, got %q", out.Message)
	}
}

func TestPlainUTF8Hint_OnWrite(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "new.txt")
	h := NewHandler([]string{dir})

	_, out, _ := h.HandleWriteFile(context.Background(), nil, WriteFileInput{Path: p, Content: "hello\n"})
	if !strings.Contains(out.Message, "built-in file tools") {
		t.Errorf("expected the hint in the write message, got %q", out.Message)
	}
}
