// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleGrep_UTF16(t *testing.T) {
	for _, v := range utf16Variants() {
		t.Run(v.name, func(t *testing.T) {
			dir := t.TempDir()
			h := NewHandler([]string{dir})
			path := writeUTF16File(t, dir, "sample.mq5", utf16SampleCRLF, v)

			result, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
				Pattern:  "Привет",
				Paths:    []string{path},
				Encoding: v.encoding,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatal("expected success, got error")
			}
			if len(output.Matches) != 1 {
				t.Fatalf("expected 1 match in UTF-16 file, got %d", len(output.Matches))
			}
			match := output.Matches[0]
			if match.Line != 2 {
				t.Errorf("Line = %d, want 2", match.Line)
			}
			if match.Text != "// Привет, мир" {
				t.Errorf("Text = %q, want %q", match.Text, "// Привет, мир")
			}
			if match.Column != 4 {
				t.Errorf("Column = %d, want 4", match.Column)
			}
		})
	}
}

// A match on line 1 must not be shifted by the BOM.
func TestHandleGrep_UTF16FirstLineColumn(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := writeUTF16File(t, dir, "sample.mq5", utf16SampleCRLF, utf16Variant{littleEndian: true, withBOM: true})

	_, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern: "#property",
		Paths:   []string{path},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(output.Matches))
	}
	if output.Matches[0].Column != 1 {
		t.Errorf("Column = %d, want 1", output.Matches[0].Column)
	}
	if output.Matches[0].Text != "#property strict" {
		t.Errorf("Text = %q, want %q", output.Matches[0].Text, "#property strict")
	}
}

// Aliases are reported under their canonical name.
func TestHandleGrep_UTF16EncodingAlias(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := writeUTF16File(t, dir, "sample.mq5", utf16SampleLF, utf16Variant{littleEndian: true})

	_, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern:  "città",
		Paths:    []string{path},
		Encoding: "UTF-16LE",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(output.Matches))
	}
	if output.Matches[0].Encoding != "utf-16-le" {
		t.Errorf("Encoding = %q, want utf-16-le", output.Matches[0].Encoding)
	}
}

func TestDecodeFileContent(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		forced   string
		want     string
		wantName string
	}{
		{"utf-8 mixed scripts", []byte(utf16SampleLF), "", utf16SampleLF, "utf-8"},
		{"utf-8 bom stripped", append([]byte{0xEF, 0xBB, 0xBF}, "abc"...), "", "abc", "utf-8"},
		{"cp1251", []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2}, "", "Привет", "windows-1251"},
		{"utf-16-le bom", utf16Bytes(utf16SampleLF, true, true), "", utf16SampleLF, "utf-16-le"},
		{"utf-16-be forced", utf16Bytes(utf16SampleLF, false, false), "utf16be", utf16SampleLF, "utf-16-be"},
		{"unknown forced falls back", []byte("abc"), "klingon", "abc", "utf-8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, name := decodeFileContent(tt.data, tt.forced, "utf-8")
			if got != tt.want {
				t.Errorf("content = %q, want %q", got, tt.want)
			}
			if name != tt.wantName {
				t.Errorf("encoding = %q, want %q", name, tt.wantName)
			}
		})
	}
}

func TestHandleGrep_RealUTF16Fixtures(t *testing.T) {
	patterns := map[string]string{
		"utf-16-le": "易建联",
		"utf-16-be": "Unicode",
	}

	fixtureDir, err := filepath.Abs(realFixtureDir)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler([]string{fixtureDir})

	for _, f := range loadRealFixtures(t) {
		t.Run(f.Encoding, func(t *testing.T) {
			readRealFixture(t, f)
			pattern, ok := patterns[f.Encoding]
			if !ok {
				t.Fatalf("no grep pattern for fixture encoding %q", f.Encoding)
			}

			// BOM-less needs an explicit encoding.
			requested := f.Encoding
			if f.BOM != "none" {
				requested = ""
			}

			result, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
				Pattern:  pattern,
				Paths:    []string{filepath.Join(fixtureDir, f.File)},
				Encoding: requested,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatalf("grep failed for the real %s fixture", f.Encoding)
			}
			if len(output.Matches) == 0 {
				t.Fatalf("expected matches for %q in the real %s fixture", pattern, f.Encoding)
			}
			if output.FilesMatched != 1 {
				t.Errorf("FilesMatched = %d, want 1", output.FilesMatched)
			}
			if output.Matches[0].Encoding != f.Encoding {
				t.Errorf("Encoding = %q, want %q", output.Matches[0].Encoding, f.Encoding)
			}
		})
	}
}

func TestHandleGrep_MQLFixtures(t *testing.T) {
	for _, f := range mqlFixtures() {
		t.Run(f.file, func(t *testing.T) {
			original, err := os.ReadFile(filepath.Join("testdata/mql", f.file))
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			h := NewHandler([]string{dir})
			path := filepath.Join(dir, f.file)
			if err := os.WriteFile(path, original, 0644); err != nil {
				t.Fatal(err)
			}

			// Auto-detection only.
			result, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
				Pattern: "città",
				Paths:   []string{path},
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatalf("grep failed for %s", f.file)
			}
			if len(output.Matches) != 1 {
				t.Fatalf("expected 1 match in %s, got %d", f.file, len(output.Matches))
			}
			if got := output.Matches[0].Text; got != "// Italiano: città" {
				t.Errorf("Text = %q, want %q", got, "// Italiano: città")
			}
		})
	}
}
