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

// The text diff cannot show bytes that change outside the edit, so the result has to say so.
func TestHandleEditFile_ReportsReencodeDrift(t *testing.T) {
	tests := []struct {
		name, encoding string
		data           []byte
		drift          bool
	}{
		// 87 90 is NEC's duplicate of 81 E0, and the encoder writes the latter.
		{"shift_jis duplicate", "shift_jis", append([]byte{0x87, 0x90}, "\r\nabc\r\n"...), true},
		{"shift_jis exact", "shift_jis", append([]byte{0x81, 0xE0}, "\r\nabc\r\n"...), false},
		{"cp1251", "windows-1251", append([]byte{0xCF, 0xF0}, "\r\nabc\r\n"...), false},
		{"utf-16 with BOM", "utf-16-le", []byte{0xFF, 0xFE, 'x', 0, '\r', 0, '\n', 0, 'a', 0, 'b', 0, 'c', 0, '\r', 0, '\n', 0}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			h := NewHandler([]string{dir})
			path := filepath.Join(dir, "f.txt")
			if err := os.WriteFile(path, tc.data, 0644); err != nil {
				t.Fatal(err)
			}
			result, _, err := h.HandleEditFile(context.Background(), nil, EditFileInput{
				Path: path, Encoding: tc.encoding, Edits: []EditOperation{{OldText: "abc", NewText: "abd"}},
			})
			if err != nil {
				t.Fatal(err)
			}
			text := extractTextFromResult(result.Content)
			if result.IsError {
				t.Fatal(text)
			}
			if got := strings.Contains(text, "untouched bytes"); got != tc.drift {
				t.Errorf("drift note = %v, want %v in %q", got, tc.drift, text)
			}
		})
	}
}
