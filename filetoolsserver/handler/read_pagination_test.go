// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// readChunk reads path with the given offset/limit (nil = omitted).
func readChunk(t *testing.T, h *Handler, path string, offset, limit *int) ReadTextFileOutput {
	t.Helper()
	result, output, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{
		Path:   path,
		Offset: offset,
		Limit:  limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("read_text_file failed for %s", path)
	}
	return output
}

func intPtr(v int) *int { return &v }

func TestHandleReadTextFile_OffsetLimitPreservesLineEndings(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	tests := []struct {
		name           string
		content        string
		offset, limit  *int
		wantContent    string
		wantStart      int
		wantEnd        int
		wantTotalLines int
	}{
		{
			name: "crlf first chunk", content: "a\r\nb\r\nc\r\n",
			offset: intPtr(1), limit: intPtr(2),
			wantContent: "a\r\nb\r\n", wantStart: 1, wantEnd: 2, wantTotalLines: 3,
		},
		{
			name: "crlf middle chunk", content: "a\r\nb\r\nc\r\n",
			offset: intPtr(2), limit: intPtr(1),
			wantContent: "b\r\n", wantStart: 2, wantEnd: 2, wantTotalLines: 3,
		},
		{
			name: "crlf last chunk", content: "a\r\nb\r\nc\r\n",
			offset: intPtr(3), limit: intPtr(5),
			wantContent: "c\r\n", wantStart: 3, wantEnd: 3, wantTotalLines: 3,
		},
		{
			name: "crlf unterminated last line", content: "a\r\nb\r\nc",
			offset: intPtr(2), limit: intPtr(2),
			wantContent: "b\r\nc", wantStart: 2, wantEnd: 3, wantTotalLines: 3,
		},
		{
			name: "crlf offset past end", content: "a\r\nb\r\nc\r\n",
			offset:      intPtr(4),
			wantContent: "", wantStart: 4, wantEnd: 3, wantTotalLines: 3,
		},
		{
			name: "lf middle chunk", content: "a\nb\nc\n",
			offset: intPtr(2), limit: intPtr(2),
			wantContent: "b\nc\n", wantStart: 2, wantEnd: 3, wantTotalLines: 3,
		},
		{
			name: "lf unterminated last line", content: "a\nb\nc",
			offset:      intPtr(3),
			wantContent: "c", wantStart: 3, wantEnd: 3, wantTotalLines: 3,
		},
		{
			name: "lone cr is not a line break", content: "a\rb\nc\n",
			offset: intPtr(1), limit: intPtr(1),
			wantContent: "a\rb\n", wantStart: 1, wantEnd: 1, wantTotalLines: 2,
		},
		{
			name: "limit beyond end keeps terminator", content: "a\r\n",
			limit:       intPtr(9),
			wantContent: "a\r\n", wantStart: 1, wantEnd: 1, wantTotalLines: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tempDir, tt.name+".txt")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			output := readChunk(t, h, path, tt.offset, tt.limit)

			if output.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", output.Content, tt.wantContent)
			}
			if output.StartLine != tt.wantStart {
				t.Errorf("StartLine = %d, want %d", output.StartLine, tt.wantStart)
			}
			if output.EndLine != tt.wantEnd {
				t.Errorf("EndLine = %d, want %d", output.EndLine, tt.wantEnd)
			}
			if output.TotalLines != tt.wantTotalLines {
				t.Errorf("TotalLines = %d, want %d", output.TotalLines, tt.wantTotalLines)
			}
		})
	}
}

func TestHandleReadTextFile_TotalLinesTermination(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{"newline terminated", "a\nb\n", 2},
		{"not terminated", "a\nb", 2},
		{"crlf terminated", "a\r\nb\r\n", 2},
		{"single line no newline", "a", 1},
		{"blank line before eof", "a\n\n", 2},
		{"empty file", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tempDir, tt.name+".txt")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			output := readChunk(t, h, path, nil, nil)

			if output.TotalLines != tt.want {
				t.Errorf("TotalLines = %d, want %d", output.TotalLines, tt.want)
			}
			if output.Content != tt.content {
				t.Errorf("Content = %q, want %q", output.Content, tt.content)
			}
		})
	}
}

// Decoding happens before pagination, so UTF-16 chunks must stay CRLF-faithful too.
func TestHandleReadTextFile_OffsetLimitUTF16CRLF(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	for _, v := range utf16Variants() {
		t.Run(v.name, func(t *testing.T) {
			path := writeUTF16File(t, tempDir, v.name+".txt", utf16SampleCRLF, v)

			offset, limit := 2, 2
			result, output, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{
				Path:     path,
				Encoding: v.encoding,
				Offset:   &offset,
				Limit:    &limit,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatalf("read_text_file failed for %s", v.name)
			}

			want := "// Привет, мир\r\n// città\r\n"
			if output.Content != want {
				t.Errorf("Content = %q, want %q", output.Content, want)
			}
			if output.TotalLines != 4 {
				t.Errorf("TotalLines = %d, want 4", output.TotalLines)
			}
			if output.StartLine != 2 || output.EndLine != 3 {
				t.Errorf("lines %d-%d, want 2-3", output.StartLine, output.EndLine)
			}
		})
	}
}
