// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/encoding"
)

func utf32Bytes(t *testing.T, charset, text string) []byte {
	t.Helper()
	enc, _ := encoding.Get(charset)
	b, err := enc.NewEncoder().Bytes([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// A byte-level \r insertion breaks UTF-32's 4-byte alignment, BOM or not, so the conversion has to go per code unit.
func TestChangeLineEndings_UTF32PerCodeUnit(t *testing.T) {
	tests := []struct {
		name, charset, encoding string
		bom                     bool
		from, style, want       string
	}{
		{"le with bom to crlf", "utf-32-le", "", true, "a\nб\n", "crlf", "a\r\nб\r\n"},
		{"be with bom to lf", "utf-32-be", "", true, "a\r\nб\r\n", "lf", "a\nб\n"},
		{"le without bom, named", "utf-32-le", "utf-32-le", false, "a\nб\n", "crlf", "a\r\nб\r\n"},
		// U+1000D and U+1000A share their low 16 bits with CR and LF.
		{"whole code points compared", "utf-32-be", "", true, "\U0001000D\U0001000A\n", "crlf", "\U0001000D\U0001000A\r\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			h := NewHandler([]string{dir})

			var prefix []byte
			if tc.bom {
				prefix = encoding.BOMBytesFor(tc.charset)
			}
			path := filepath.Join(dir, "f.txt")
			if err := os.WriteFile(path, append(prefix, utf32Bytes(t, tc.charset, tc.from)...), 0644); err != nil {
				t.Fatal(err)
			}

			result, _, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{Path: path, Style: tc.style, Encoding: tc.encoding})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatalf("conversion failed: %v", result.Content)
			}

			got, _ := os.ReadFile(path)
			if want := append(prefix, utf32Bytes(t, tc.charset, tc.want)...); !bytes.Equal(got, want) {
				t.Errorf("got % x\nwant % x", got, want)
			}
		})
	}
}

// Rewriting at the wrong width corrupts the file, so these refuse and leave it as it was.
func TestChangeLineEndings_RefusesTheWrongWidth(t *testing.T) {
	withBOM := append(encoding.BOMBytesFor("utf-32-le"), utf32Bytes(t, "utf-32-le", "a\nb\n")...)
	tests := []struct {
		name, encoding string
		data           []byte
	}{
		{"utf-32 bom read as utf-16", "utf-16-le", withBOM},
		{"bom-less utf-32, not named", "", utf32Bytes(t, "utf-32-le", "line one\nline two\n")},
		{"bom-less utf-32 read as utf-8", "utf-8", utf32Bytes(t, "utf-32-be", "a\nb\n")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			h := NewHandler([]string{dir})
			path := filepath.Join(dir, "f.txt")
			if err := os.WriteFile(path, tc.data, 0644); err != nil {
				t.Fatal(err)
			}

			result, _, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{Path: path, Style: "crlf", Encoding: tc.encoding})
			if err != nil {
				t.Fatal(err)
			}
			if !result.IsError {
				t.Error("converted instead of refusing")
			}
			if got, _ := os.ReadFile(path); !bytes.Equal(got, tc.data) {
				t.Errorf("file changed:\ngot  % x\nwant % x", got, tc.data)
			}
		})
	}
}
