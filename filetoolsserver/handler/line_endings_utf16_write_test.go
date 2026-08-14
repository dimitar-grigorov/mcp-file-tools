// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
)

// writeUTF16 writes text as UTF-16 with a BOM.
func writeUTF16(t *testing.T, path, text, charset string) {
	t.Helper()
	enc, ok := encoding.Get(charset)
	if !ok {
		t.Fatalf("unknown charset %q", charset)
	}
	body, err := encoding.Encode(text, enc, charset)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(encoding.BOMBytesFor(charset), body...), 0644); err != nil {
		t.Fatal(err)
	}
}

// readUTF16 reads a UTF-16 file back as UTF-8, BOM stripped.
func readUTF16(t *testing.T, path, charset string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text, err := encoding.Decode(data[len(encoding.BOMBytesFor(charset)):], charset)
	if err != nil {
		t.Fatal(err)
	}
	return text
}

// The 00 between CR and LF made every CRLF look like a lone LF, so "preserve"
// concluded the file was LF and stripped the CRs it was meant to keep.
func TestWriteFile_PreservesCRLFOnUTF16(t *testing.T) {
	for _, charset := range []string{"utf-16-le", "utf-16-be"} {
		t.Run(charset, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, "f.txt")
			writeUTF16(t, p, "a\r\nb\r\n", charset)

			h := NewHandler([]string{dir})
			res, out, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{
				Path:    p,
				Content: "a\r\nb CHANGED\r\n",
			})
			if err != nil {
				t.Fatal(err)
			}
			if res.IsError {
				t.Fatalf("write_file failed: %+v", res.Content)
			}
			if got, want := readUTF16(t, p, charset), "a\r\nb CHANGED\r\n"; got != want {
				t.Errorf("line endings not preserved: got %q, want %q", got, want)
			}
			if out.LineEndings != "" {
				t.Errorf("reported a normalisation that did not happen: %q", out.LineEndings)
			}
		})
	}
}

// An agent rewriting a UTF-16 CRLF file typically emits LF; it must come back CRLF.
func TestWriteFile_RestoresCRLFOnUTF16FromLFContent(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	writeUTF16(t, p, "a\r\nb\r\n", "utf-16-le")

	h := NewHandler([]string{dir})
	_, out, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{
		Path:    p,
		Content: "a\nb CHANGED\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := readUTF16(t, p, "utf-16-le"), "a\r\nb CHANGED\r\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if out.LineEndings != LineEndingCRLF {
		t.Errorf("LineEndings = %q, want %q", out.LineEndings, LineEndingCRLF)
	}
}

func TestEditFile_PreservesCRLFOnUTF16(t *testing.T) {
	for _, charset := range []string{"utf-16-le", "utf-16-be"} {
		t.Run(charset, func(t *testing.T) {
			dir := t.TempDir()
			p := filepath.Join(dir, "f.txt")
			writeUTF16(t, p, "alpha\r\nbeta\r\ngamma\r\n", charset)

			h := NewHandler([]string{dir})
			res, _, err := h.HandleEditFile(context.Background(), nil, EditFileInput{
				Path:  p,
				Edits: []EditOperation{{OldText: "beta", NewText: "BETA"}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if res.IsError {
				t.Fatalf("edit_file failed: %+v", res.Content)
			}
			if got, want := readUTF16(t, p, charset), "alpha\r\nBETA\r\ngamma\r\n"; got != want {
				t.Errorf("line endings not preserved: got %q, want %q", got, want)
			}
		})
	}
}

// A UTF-8 file must not regress while the UTF-16 path is being fixed.
func TestWriteFile_PreservesCRLFOnUTF8(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte("a\r\nb\r\n"), 0644); err != nil {
		t.Fatal(err)
	}

	h := NewHandler([]string{dir})
	if _, _, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{
		Path: p, Content: "a\nb CHANGED\n",
	}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "a\r\nb CHANGED\r\n" {
		t.Errorf("got %q", string(got))
	}
}

// read_text_file's mixed hint also has to see decoded text to fire on UTF-16.
func TestReadTextFile_MixedHintOnUTF16(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	writeUTF16(t, p, "a\r\nb\nc\r\n", "utf-16-le")

	h := NewHandler([]string{dir})
	_, out, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{Path: p})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Hint, "MIXED line endings") {
		t.Errorf("no mixed-line-ending hint for a mixed UTF-16 file: %q", out.Hint)
	}
}
