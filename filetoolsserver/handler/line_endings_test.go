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
	// Test reader wrapper works - detection logic is tested in TestDetectLineEndings
	r := bytes.NewReader([]byte("line1\r\nline2\r\n"))
	got, err := DetectLineEndingsFromReader(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Style != LineEndingCRLF {
		t.Errorf("Style = %q, want %q", got.Style, LineEndingCRLF)
	}
	if got.CRLFCount != 2 {
		t.Errorf("CRLFCount = %d, want 2", got.CRLFCount)
	}
}

func TestDetectLineEndingsFromFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(path, []byte("line1\r\nline2\r\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := DetectLineEndingsFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Style != LineEndingCRLF {
		t.Errorf("Style = %q, want %q", got.Style, LineEndingCRLF)
	}
}

func TestDetectLineEndingsFromFile_NotFound(t *testing.T) {
	_, err := DetectLineEndingsFromFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestHandleDetectLineEndings(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	tests := []struct {
		name               string
		content            []byte
		wantStyle          string
		wantTotalLines     int
		wantInconsistent   []int
		wantInconsistentNil bool
	}{
		{
			name:             "pure CRLF file",
			content:          []byte("line1\r\nline2\r\nline3"),
			wantStyle:        LineEndingCRLF,
			wantTotalLines:   3,
			wantInconsistent: []int{},
		},
		{
			name:             "pure LF file",
			content:          []byte("line1\nline2\nline3"),
			wantStyle:        LineEndingLF,
			wantTotalLines:   3,
			wantInconsistent: []int{},
		},
		{
			name:             "mixed - mostly CRLF with one LF",
			content:          []byte("line1\r\nline2\nline3\r\nline4\r\n"),
			wantStyle:        LineEndingMixed,
			wantTotalLines:   5,
			wantInconsistent: []int{2}, // line 2 has LF when CRLF is dominant
		},
		{
			name:             "mixed - mostly LF with one CRLF",
			content:          []byte("line1\nline2\r\nline3\nline4\n"),
			wantStyle:        LineEndingMixed,
			wantTotalLines:   5,
			wantInconsistent: []int{2}, // line 2 has CRLF when LF is dominant
		},
		{
			name:             "no line endings",
			content:          []byte("single line without newline"),
			wantStyle:        LineEndingNone,
			wantTotalLines:   1,
			wantInconsistent: []int{},
		},
		{
			name:             "empty file",
			content:          []byte{},
			wantStyle:        LineEndingNone,
			wantTotalLines:   1,
			wantInconsistent: []int{},
		},
		{
			name:             "mixed - equal CRLF and LF",
			content:          []byte("line1\r\nline2\n"),
			wantStyle:        LineEndingMixed,
			wantTotalLines:   3,
			wantInconsistent: []int{2}, // LF is minority when counts equal (CRLF >= LF)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, tt.name+".txt")
			if err := os.WriteFile(testFile, tt.content, 0644); err != nil {
				t.Fatal(err)
			}

			input := DetectLineEndingsInput{Path: testFile}
			result, output, err := h.HandleDetectLineEndings(nil, nil, input)
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Errorf("expected success, got error")
			}

			if output.Style != tt.wantStyle {
				t.Errorf("Style = %q, want %q", output.Style, tt.wantStyle)
			}
			if output.TotalLines != tt.wantTotalLines {
				t.Errorf("TotalLines = %d, want %d", output.TotalLines, tt.wantTotalLines)
			}
			if len(output.InconsistentLines) != len(tt.wantInconsistent) {
				t.Errorf("InconsistentLines = %v, want %v", output.InconsistentLines, tt.wantInconsistent)
			} else {
				for i, line := range output.InconsistentLines {
					if line != tt.wantInconsistent[i] {
						t.Errorf("InconsistentLines[%d] = %d, want %d", i, line, tt.wantInconsistent[i])
					}
				}
			}
		})
	}
}

func TestHandleDetectLineEndings_PathValidation(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Test path outside allowed directory
	input := DetectLineEndingsInput{Path: "/not/allowed/path.txt"}
	result, _, err := h.HandleDetectLineEndings(nil, nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for path outside allowed directory")
	}
}
