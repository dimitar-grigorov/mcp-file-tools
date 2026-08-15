// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/config"
	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/encoding"
)

// sparseCyrillicSource is what defeats detection: a big mostly-ASCII unit.
func sparseCyrillicSource(t *testing.T) []byte {
	t.Helper()
	var b strings.Builder
	for i := range 400 {
		b.WriteString("  Writeln('padding line to dilute the sample ")
		b.WriteByte(byte('0' + i%10))
		b.WriteString("');\n")
	}
	b.WriteString("  ShowMessage('вече извикан');\n")

	enc, ok := encoding.Get("cp1251")
	if !ok {
		t.Fatal("cp1251 missing from the registry")
	}
	data, err := encoding.Encode(b.String(), enc, "cp1251")
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	return data
}

func TestResolveEncoding_InconclusiveUsesConfiguredDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unit1.pas")
	if err := os.WriteFile(path, sparseCyrillicSource(t), 0644); err != nil {
		t.Fatal(err)
	}

	cyrillic := &config.Config{DefaultEncoding: "cp1251", MemoryThreshold: 1 << 26}
	h := NewHandler([]string{dir}, WithConfig(cyrillic))

	got, err := h.resolveEncoding("", path)
	if err != nil {
		t.Fatal(err)
	}
	// Detection worked or fell through to the default; utf-8 would be the bug.
	if got.name == "utf-8" {
		t.Errorf("resolved utf-8 for a cp1251 file with MCP_DEFAULT_ENCODING=cp1251 (detected %q)", got.detectedEncoding)
	}
}

// The default configuration must keep behaving as it always has.
func TestResolveEncoding_DefaultConfigStillFallsBackToUTF8(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(path, []byte("plain ascii\n"), 0644); err != nil {
		t.Fatal(err)
	}

	h := NewHandler([]string{dir}, WithConfig(&config.Config{
		DefaultEncoding: config.DefaultEncoding, MemoryThreshold: 1 << 26,
	}))

	got, err := h.resolveEncoding("", path)
	if err != nil {
		t.Fatal(err)
	}
	if !encoding.IsUTF8(got.name) {
		t.Errorf("resolved %q for plain ASCII, want utf-8", got.name)
	}
	if got.encoder != nil {
		t.Error("utf-8 must resolve to a nil encoder so decodeContent passes bytes through")
	}
}

// A cp1251 default must not corrupt a file that really is utf-8.
func TestReadTextFile_DetectedUTF8BeatsCyrillicDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")
	const want = "Привет, мир! Това е UTF-8 текст с достатъчно съдържание за разпознаване."
	if err := os.WriteFile(path, []byte(want), 0644); err != nil {
		t.Fatal(err)
	}

	h := NewHandler([]string{dir}, WithConfig(&config.Config{
		DefaultEncoding: "cp1251", MemoryThreshold: 1 << 26,
	}))

	res, out, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("read failed: %v", out)
	}
	if out.Content != want {
		t.Errorf("content = %q, want %q", out.Content, want)
	}
}
