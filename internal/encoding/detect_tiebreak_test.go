// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"os"
	"path/filepath"
	"testing"
)

// Chunked detection weighted each chunk's verdict, then picked the winner by
// ranging a map, so a tie resolved differently from run to run.
func TestDetectChunked_IsDeterministic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mixed.txt")

	// Two chunks of equal size, each pulling toward a different single-byte codec.
	cyrillic := []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2, 0x20} // "Привет " in cp1251
	greek := []byte{0xEA, 0xE1, 0xEB, 0xE7, 0xEC, 0xDD, 0x20}    // Greek in cp1253

	var data []byte
	for len(data) < ChunkSize {
		data = append(data, cyrillic...)
	}
	for len(data) < ChunkSize*2 {
		data = append(data, greek...)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	first, err := DetectFromFile(path, "chunked")
	if err != nil {
		t.Fatal(err)
	}
	for i := range 30 {
		got, err := DetectFromFile(path, "chunked")
		if err != nil {
			t.Fatal(err)
		}
		if got.Charset != first.Charset {
			t.Fatalf("run %d returned %q, first run returned %q", i, got.Charset, first.Charset)
		}
	}
}
