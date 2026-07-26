// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

// Guards against the double-encoding class reported in openai/codex#4013 and #13743,
// where UTF-8 bytes get reinterpreted as CP1252 and re-encoded on save.

func TestHandleWriteFile_UTF8NotDoubleEncoded(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	path := filepath.Join(tempDir, "nordic.txt")

	result, _, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{
		Path:     path,
		Content:  "æ ø å",
		Encoding: "utf-8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error: %v", result.Content)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := []byte{0xC3, 0xA6, 0x20, 0xC3, 0xB8, 0x20, 0xC3, 0xA5}
	if !bytes.Equal(got, want) {
		t.Errorf("bytes on disk = % x, want % x", got, want)
	}

	// c3 83 c2 a5 is the double-encoded form of å
	if bytes.Contains(got, []byte{0xC3, 0x83, 0xC2, 0xA5}) {
		t.Error("content was double-encoded on write")
	}
}

func TestHandleWriteFile_AstralAndSymbolsSurviveRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	cases := []struct {
		name    string
		content string
	}{
		{"nordic", "æ ø å Æ Ø Å"},
		{"math symbols", "≈ ≠ ±"},
		{"astral emoji", "🔴 🈶"},
		{"cyrillic", "Ще проверим"},
		{"zero width space", "a\u200bb"},
		{"mixed", "å ≈ 🔴 Щ"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(tempDir, tc.name+".txt")

			if _, _, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{
				Path:     path,
				Content:  tc.content,
				Encoding: "utf-8",
			}); err != nil {
				t.Fatal(err)
			}

			_, output, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{Path: path})
			if err != nil {
				t.Fatal(err)
			}

			if output.Content != tc.content {
				t.Errorf("round trip = %q, want %q", output.Content, tc.content)
			}
		})
	}
}

func TestHandleWriteFile_CP1252UsesSingleBytes(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	path := filepath.Join(tempDir, "cp1252.txt")

	if _, _, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{
		Path:     path,
		Content:  "æ ø å",
		Encoding: "cp1252",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := []byte{0xE6, 0x20, 0xF8, 0x20, 0xE5}
	if !bytes.Equal(got, want) {
		t.Errorf("bytes on disk = % x, want % x", got, want)
	}
}

func TestHandleReadTextFile_CP1252DecodesWithoutMojibake(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	path := filepath.Join(tempDir, "legacy.txt")

	// real CP1252 bytes, as a Windows editor would leave them
	if err := os.WriteFile(path, []byte{0xE6, 0x20, 0xF8, 0x20, 0xE5}, 0o644); err != nil {
		t.Fatal(err)
	}

	_, output, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{
		Path:     path,
		Encoding: "cp1252",
	})
	if err != nil {
		t.Fatal(err)
	}

	if output.Content != "æ ø å" {
		t.Errorf("content = %q, want %q", output.Content, "æ ø å")
	}
}

func TestHandleReadTextFile_CP1252ThenWriteBackPreservesBytes(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	path := filepath.Join(tempDir, "preserve.txt")

	original := []byte{0xE6, 0x20, 0xF8, 0x20, 0xE5}
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}

	_, output, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{
		Path:     path,
		Encoding: "cp1252",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{
		Path:     path,
		Content:  output.Content,
		Encoding: "cp1252",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(got, original) {
		t.Errorf("after read/write cycle = % x, want % x", got, original)
	}
}
