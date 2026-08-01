// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/config"
)

func writeAndRead(t *testing.T, existing string, in WriteFileInput, cfg *config.Config) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if existing != "" {
		if err := os.WriteFile(p, []byte(existing), 0644); err != nil {
			t.Fatal(err)
		}
	}
	opts := []Option{}
	if cfg != nil {
		opts = append(opts, WithConfig(cfg))
	}
	h := NewHandler([]string{dir}, opts...)
	in.Path = p
	res, _, err := h.HandleWriteFile(context.Background(), nil, in)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("write_file returned an error result: %+v", res.Content)
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(got)
}

// An agent rewriting a CRLF file typically emits LF; the file must stay CRLF.
func TestWriteFile_PreservesLineEndings(t *testing.T) {
	tests := []struct {
		name     string
		existing string
		input    WriteFileInput
		want     string
	}{
		{
			name:     "CRLF file, agent sends mixed content",
			existing: "a\r\nb\r\nc\r\n",
			input:    WriteFileInput{Content: "a\r\nb CHANGED\nc\r\n"},
			want:     "a\r\nb CHANGED\r\nc\r\n",
		},
		{
			name:     "CRLF file, agent sends all-LF content",
			existing: "a\r\nb\r\n",
			input:    WriteFileInput{Content: "a\nb\n"},
			want:     "a\r\nb\r\n",
		},
		{
			name:     "LF file stays LF even if content has CRLF",
			existing: "a\nb\n",
			input:    WriteFileInput{Content: "a\r\nb\r\n"},
			want:     "a\nb\n",
		},
		{
			name:     "mixed file is repaired to the dominant style",
			existing: "a\r\nb\r\nc\n",
			input:    WriteFileInput{Content: "a\nb\nc\n"},
			want:     "a\r\nb\r\nc\r\n",
		},
		{
			name:     "explicit crlf overrides an LF file",
			existing: "a\nb\n",
			input:    WriteFileInput{Content: "a\nb\n", LineEndings: "crlf"},
			want:     "a\r\nb\r\n",
		},
		{
			name:     "explicit lf overrides a CRLF file",
			existing: "a\r\nb\r\n",
			input:    WriteFileInput{Content: "a\r\nb\r\n", LineEndings: "lf"},
			want:     "a\nb\n",
		},
		{
			name:     "asis writes content verbatim",
			existing: "a\r\nb\r\n",
			input:    WriteFileInput{Content: "a\r\nb CHANGED\nc\r\n", LineEndings: "asis"},
			want:     "a\r\nb CHANGED\nc\r\n",
		},
		{
			name:     "new file without a configured default is left alone",
			existing: "",
			input:    WriteFileInput{Content: "a\nb\n"},
			want:     "a\nb\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := writeAndRead(t, tt.existing, tt.input, nil); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteFile_NewFileUsesConfiguredDefault(t *testing.T) {
	cfg := &config.Config{DefaultEncoding: "utf-8", MemoryThreshold: 1 << 26, DefaultLineEndings: "crlf"}
	got := writeAndRead(t, "", WriteFileInput{Content: "a\nb\n"}, cfg)
	if want := "a\r\nb\r\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// An existing file still wins over the configured default.
func TestWriteFile_ExistingFileBeatsConfiguredDefault(t *testing.T) {
	cfg := &config.Config{DefaultEncoding: "utf-8", MemoryThreshold: 1 << 26, DefaultLineEndings: "crlf"}
	got := writeAndRead(t, "a\nb\n", WriteFileInput{Content: "a\nb\n"}, cfg)
	if want := "a\nb\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWriteFile_InvalidLineEndingsPolicy(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	res, _, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{
		Path: filepath.Join(dir, "f.txt"), Content: "x", LineEndings: "windows",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("expected an error result for an unknown lineEndings policy")
	}
}

func TestParseLineEndingPolicy(t *testing.T) {
	for _, tt := range []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", lineEndingsPreserve, false},
		{"preserve", lineEndingsPreserve, false},
		{"CRLF", LineEndingCRLF, false},
		{" lf ", LineEndingLF, false},
		{"asis", lineEndingsAsIs, false},
		{"mixed", "", true},
	} {
		got, err := parseLineEndingPolicy(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseLineEndingPolicy(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("parseLineEndingPolicy(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
