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

// change_line_endings must not silently corrupt UTF-32: byte-level \r insertion breaks the
// 4-byte alignment. Until UTF-32 is handled per code unit, it must refuse and leave the file intact.
func TestChangeLineEndings_UTF32NotCorrupted(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})

	// UTF-32LE BOM + "a\n" (BOM=FF FE 00 00, 'a'=61.., '\n'=0A..)
	data := []byte{0xFF, 0xFE, 0x00, 0x00, 0x61, 0x00, 0x00, 0x00, 0x0A, 0x00, 0x00, 0x00}
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)

	result, _, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{Path: path, Style: "crlf"})
	if err != nil {
		t.Fatal(err)
	}

	after, _ := os.ReadFile(path)
	if len(after)%4 != 0 {
		t.Errorf("UTF-32 corrupted: output is %d bytes, not a multiple of 4", len(after))
	}
	if !result.IsError && !bytes.Equal(before, after) {
		t.Errorf("silently rewrote UTF-32 (%d -> %d bytes) instead of refusing", len(before), len(after))
	}
}
