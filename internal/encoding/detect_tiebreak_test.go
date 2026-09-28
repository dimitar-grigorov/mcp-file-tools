// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import "testing"

// Chunked detection weighted each chunk's verdict, then picked the winner by
// ranging a map, so a tie resolved differently from run to run.
func TestDetectChunked_IsDeterministic(t *testing.T) {
	tied := []chunkResult{
		{charset: "windows-1253", confidence: 60, weight: ChunkSize},
		{charset: "windows-1251", confidence: 60, weight: ChunkSize},
	}
	for i := range 200 {
		scan := &chunkScan{done: true, results: tied}
		if got := scan.vote(); got.Charset != "windows-1251" {
			t.Fatalf("run %d returned %q, want the name that sorts first", i, got.Charset)
		}
	}
}
