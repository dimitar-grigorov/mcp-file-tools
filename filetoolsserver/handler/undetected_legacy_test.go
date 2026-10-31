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

// Too short to detect: "日本語のメモ\r\nfruit = APPLE\r\n" in Shift_JIS.
var shortSJIS = []byte("\x93\xfa\x96\x7b\x8c\xea\x82\xcc\x83\x81\x83\x82\r\nfruit = APPLE\r\n")

func TestUndetectedLegacyFile(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := filepath.Join(dir, "f.txt")
	ctx := context.Background()
	reset := func() { os.WriteFile(path, shortSJIS, 0644) }

	reset()
	res, _, _ := h.HandleEditFile(ctx, nil, EditFileInput{Path: path, Edits: []EditOperation{{OldText: "APPLE", NewText: "林檎"}}})
	if !res.IsError {
		t.Error("edit adding utf-8 text should be refused")
	}
	res, _, _ = h.HandleWriteFile(ctx, nil, WriteFileInput{Path: path, Content: "林檎\n"})
	if !res.IsError {
		t.Error("write of non-ASCII text should be refused")
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, shortSJIS) {
		t.Fatalf("file changed: %x", got)
	}

	// ASCII edits keep the legacy bytes as they are.
	res, _, _ = h.HandleEditFile(ctx, nil, EditFileInput{Path: path, Edits: []EditOperation{{OldText: "APPLE", NewText: "ORANGE"}}})
	if res.IsError {
		t.Fatal("ascii edit should pass")
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, bytes.Replace(shortSJIS, []byte("APPLE"), []byte("ORANGE"), 1)) {
		t.Errorf("ascii edit not byte-exact: %x", got)
	}

	reset()
	_, out, _ := h.HandleReadTextFile(ctx, nil, ReadTextFileInput{Path: path})
	if out.Hint == "" {
		t.Error("read should hint at an explicit encoding")
	}
	_, out, _ = h.HandleReadTextFile(ctx, nil, ReadTextFileInput{Path: path, Encoding: "shift_jis"})
	if !bytes.Contains([]byte(out.Content), []byte("日本語のメモ")) {
		t.Error("explicit shift_jis should decode cleanly")
	}
}
