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

// The read path is the hottest tool; this is what one call costs on a mid-size source file.
func BenchmarkHandleReadTextFile(b *testing.B) {
	dir := b.TempDir()
	h := NewHandler([]string{dir})
	for _, tc := range []struct {
		name   string
		lines  int
		cp1251 bool
	}{
		{"utf8_2k", 2000, false},
		{"utf8_50k", 50000, false},
		{"cp1251_50k", 50000, true},
	} {
		var sb strings.Builder
		for i := range tc.lines {
			fmt.Fprintf(&sb, "  // Коментар %d: procedure DoThing%d(const Value: Integer);\r\n", i, i)
		}
		data := []byte(sb.String())
		if tc.cp1251 {
			var err error
			data, err = charmap.Windows1251.NewEncoder().Bytes(data)
			if err != nil {
				b.Fatal(err)
			}
		}
		path := filepath.Join(dir, tc.name+".pas")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			b.Fatal(err)
		}
		b.Run(tc.name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			for b.Loop() {
				res, _, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{Path: path})
				if err != nil || res.IsError {
					b.Fatalf("read failed: %v %v", err, res)
				}
			}
		})
	}
}
