package handler

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLineEndings(t *testing.T) {
	tests := []struct {
		name      string
		input     []byte
		wantStyle string
		wantCRLF  int
		wantLF    int
	}{
		{
			name:      "CRLF only",
			input:     []byte("line1\r\nline2\r\nline3"),
			wantStyle: LineEndingCRLF,
			wantCRLF:  2,
			wantLF:    0,
		},
		{
			name:      "LF only",
			input:     []byte("line1\nline2\nline3"),
			wantStyle: LineEndingLF,
			wantCRLF:  0,
			wantLF:    2,
		},
		{
			name:      "mixed line endings",
			input:     []byte("line1\r\nline2\nline3"),
			wantStyle: LineEndingMixed,
			wantCRLF:  1,
			wantLF:    1,
		},
		{
			name:      "no line endings",
			input:     []byte("single line"),
			wantStyle: LineEndingNone,
			wantCRLF:  0,
			wantLF:    0,
		},
		{
			name:      "empty file",
			input:     []byte{},
			wantStyle: LineEndingNone,
			wantCRLF:  0,
			wantLF:    0,
		},
		{
			name:      "trailing CRLF",
			input:     []byte("line1\r\n"),
			wantStyle: LineEndingCRLF,
			wantCRLF:  1,
			wantLF:    0,
		},
		{
			name:      "trailing LF",
			input:     []byte("line1\n"),
			wantStyle: LineEndingLF,
			wantCRLF:  0,
			wantLF:    1,
		},
		{
			name:      "standalone CR ignored",
			input:     []byte("line1\rline2\nline3"),
			wantStyle: LineEndingLF,
			wantCRLF:  0,
			wantLF:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLineEndings(tt.input)
			if got.Style != tt.wantStyle {
				t.Errorf("Style = %q, want %q", got.Style, tt.wantStyle)
			}
			if got.CRLFCount != tt.wantCRLF {
				t.Errorf("CRLFCount = %d, want %d", got.CRLFCount, tt.wantCRLF)
			}
			if got.LFCount != tt.wantLF {
				t.Errorf("LFCount = %d, want %d", got.LFCount, tt.wantLF)
			}
		})
	}
}

func TestConvertLineEndings(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		target string
		want   string
	}{
		// To LF
		{"CRLF to LF", "line1\r\nline2\r\n", LineEndingLF, "line1\nline2\n"},
		{"LF to LF (unchanged)", "line1\nline2\n", LineEndingLF, "line1\nline2\n"},
		{"mixed to LF", "line1\r\nline2\nline3", LineEndingLF, "line1\nline2\nline3"},

		// To CRLF
		{"LF to CRLF", "line1\nline2\n", LineEndingCRLF, "line1\r\nline2\r\n"},
		{"CRLF to CRLF (unchanged)", "line1\r\nline2\r\n", LineEndingCRLF, "line1\r\nline2\r\n"},
		{"mixed to CRLF", "line1\r\nline2\nline3", LineEndingCRLF, "line1\r\nline2\r\nline3"},

		// Other styles (treated as LF)
		{"to mixed (becomes LF)", "line1\r\nline2\r\n", LineEndingMixed, "line1\nline2\n"},
		{"to none (becomes LF)", "line1\r\nline2\r\n", LineEndingNone, "line1\nline2\n"},

		// Edge cases
		{"no line endings", "single line", LineEndingCRLF, "single line"},
		{"empty string", "", LineEndingCRLF, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertLineEndings(tt.input, tt.target)
			if got != tt.want {
				t.Errorf("ConvertLineEndings(%q, %q) = %q, want %q", tt.input, tt.target, got, tt.want)
			}
		})
	}
}

func TestDetectLineEndingsFromReader(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantStyle string
		wantCRLF  int
		wantLF    int
	}{
		{"CRLF only", "line1\r\nline2\r\n", LineEndingCRLF, 2, 0},
		{"LF only", "line1\nline2\n", LineEndingLF, 0, 2},
		{"mixed", "line1\r\nline2\nline3", LineEndingMixed, 1, 1},
		{"none", "no newlines", LineEndingNone, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader([]byte(tt.input))
			got, err := DetectLineEndingsFromReader(r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Style != tt.wantStyle {
				t.Errorf("Style = %q, want %q", got.Style, tt.wantStyle)
			}
			if got.CRLFCount != tt.wantCRLF {
				t.Errorf("CRLFCount = %d, want %d", got.CRLFCount, tt.wantCRLF)
			}
			if got.LFCount != tt.wantLF {
				t.Errorf("LFCount = %d, want %d", got.LFCount, tt.wantLF)
			}
		})
	}
}

func TestDetectLineEndingsFromFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		content   []byte
		wantStyle string
	}{
		{"CRLF file", []byte("line1\r\nline2\r\n"), LineEndingCRLF},
		{"LF file", []byte("line1\nline2\n"), LineEndingLF},
		{"mixed file", []byte("line1\r\nline2\n"), LineEndingMixed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tempDir, tt.name+".txt")
			if err := os.WriteFile(path, tt.content, 0644); err != nil {
				t.Fatal(err)
			}

			got, err := DetectLineEndingsFromFile(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Style != tt.wantStyle {
				t.Errorf("Style = %q, want %q", got.Style, tt.wantStyle)
			}
		})
	}
}

func TestDetectLineEndingsFromFile_NotFound(t *testing.T) {
	_, err := DetectLineEndingsFromFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
