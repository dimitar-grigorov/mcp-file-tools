package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleEditFile_SimpleReplacement(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello World"), 0644)

	input := EditFileInput{
		Path:  testFile,
		Edits: []EditOperation{{OldText: "World", NewText: "Go"}},
	}

	result, output, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if !strings.Contains(output.Diff, "-Hello World") || !strings.Contains(output.Diff, "+Hello Go") {
		t.Errorf("expected diff to show change, got %q", output.Diff)
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "Hello Go" {
		t.Errorf("file should be modified, got %q", content)
	}
}

func TestHandleEditFile_DryRun(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	originalContent := "Hello World"
	os.WriteFile(testFile, []byte(originalContent), 0644)

	input := EditFileInput{
		Path:   testFile,
		Edits:  []EditOperation{{OldText: "World", NewText: "Go"}},
		DryRun: true,
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != originalContent {
		t.Errorf("file should NOT be modified in dry run, got %q", content)
	}
}

func TestHandleEditFile_MultipleEdits(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("foo bar baz"), 0644)

	input := EditFileInput{
		Path: testFile,
		Edits: []EditOperation{
			{OldText: "foo", NewText: "FOO"},
			{OldText: "bar", NewText: "BAR"},
		},
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "FOO BAR baz" {
		t.Errorf("edits should be applied, got %q", content)
	}
}

func TestHandleEditFile_WhitespaceFlexible(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("    indented line"), 0644)

	input := EditFileInput{
		Path:  testFile,
		Edits: []EditOperation{{OldText: "indented line", NewText: "modified line"}},
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success with flexible whitespace matching")
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "    modified line" {
		t.Errorf("indentation should be preserved, got %q", content)
	}
}

func TestHandleEditFile_NoMatch(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello World"), 0644)

	input := EditFileInput{
		Path:  testFile,
		Edits: []EditOperation{{OldText: "Nonexistent", NewText: "New"}},
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Errorf("expected error when oldText not found")
	}
}

func TestHandleEditFile_MultiLine(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3"), 0644)

	input := EditFileInput{
		Path:  testFile,
		Edits: []EditOperation{{OldText: "line1\nline2", NewText: "new1\nnew2"}},
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "new1\nnew2\nline3" {
		t.Errorf("multi-line edit should be applied, got %q", content)
	}
}

func TestHandleEditFile_CRLFPreservation(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	// Write file with CRLF line endings
	os.WriteFile(testFile, []byte("line1\r\nline2\r\nline3"), 0644)

	input := EditFileInput{
		Path:  testFile,
		Edits: []EditOperation{{OldText: "line1\nline2", NewText: "new1\nnew2"}},
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success with CRLF normalization")
	}

	// Verify CRLF line endings are preserved
	content, _ := os.ReadFile(testFile)
	if string(content) != "new1\r\nnew2\r\nline3" {
		t.Errorf("CRLF line endings should be preserved, got %q", content)
	}
}

func TestHandleEditFile_ValidationErrors(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	tests := []struct {
		name  string
		input EditFileInput
	}{
		{"empty path", EditFileInput{Path: "", Edits: []EditOperation{{OldText: "a", NewText: "b"}}}},
		{"empty edits", EditFileInput{Path: filepath.Join(tempDir, "f.txt"), Edits: []EditOperation{}}},
		{"outside allowed", EditFileInput{Path: "/random/path", Edits: []EditOperation{{OldText: "a", NewText: "b"}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := h.HandleEditFile(context.Background(), nil, tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if !result.IsError {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestAdjustRelativeIndent(t *testing.T) {
	tests := []struct {
		name      string
		oldLines  []string
		newLine   string
		lineIndex int
		baseIndent string
		want      string
	}{
		{
			name:       "zero relative indent",
			oldLines:   []string{"    old content"},
			newLine:    "    new content",
			lineIndex:  0,
			baseIndent: "        ",
			want:       "        new content",
		},
		{
			name:       "positive relative indent",
			oldLines:   []string{"    old content"},
			newLine:    "        new content",
			lineIndex:  0,
			baseIndent: "    ",
			want:       "        new content",
		},
		{
			name:       "negative relative indent",
			oldLines:   []string{"        old content"},
			newLine:    "    new content",
			lineIndex:  0,
			baseIndent: "        ",
			want:       "    new content",
		},
		{
			name:       "negative indent exceeds base",
			oldLines:   []string{"        old content"},
			newLine:    "new content",
			lineIndex:  0,
			baseIndent: "    ",
			want:       "new content",
		},
		{
			name:       "line index out of range",
			oldLines:   []string{},
			newLine:    "    new content",
			lineIndex:  0,
			baseIndent: "        ",
			want:       "    new content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adjustRelativeIndent(tt.oldLines, tt.newLine, tt.lineIndex, tt.baseIndent)
			if got != tt.want {
				t.Errorf("adjustRelativeIndent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleEditFile_NegativeRelativeIndent(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// File has a block indented at 8 spaces
	original := "func main() {\n        if true {\n            fmt.Println(\"hello\")\n        }\n}\n"
	testFile := filepath.Join(tempDir, "test.go")
	os.WriteFile(testFile, []byte(original), 0644)

	// oldText has 8-space if block, newText dedents the body to match the if level
	input := EditFileInput{
		Path: testFile,
		Edits: []EditOperation{{
			OldText: "        if true {\n            fmt.Println(\"hello\")\n        }",
			NewText: "        if true {\n        fmt.Println(\"hello\")\n        }",
		}},
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Error("expected success")
	}

	content, _ := os.ReadFile(testFile)
	expected := "func main() {\n        if true {\n        fmt.Println(\"hello\")\n        }\n}\n"
	if string(content) != expected {
		t.Errorf("negative indent not applied.\ngot:  %q\nwant: %q", string(content), expected)
	}
}

func TestEditFile_CP1251Encoding(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// CP1251 encoded Cyrillic text: "Невалиден тип." (Invalid type.)
	// In CP1251: Н=0xCD, е=0xE5, в=0xE2, а=0xE0, л=0xEB, и=0xE8, д=0xE4, н=0xED
	cp1251Content := []byte{
		0xCD, 0xE5, 0xE2, 0xE0, 0xEB, 0xE8, 0xE4, 0xE5, 0xED, // Невалиден
		0x20,       // space
		0xF2, 0xE8, 0xEF, // тип
		0x2E, // .
	}

	testFile := filepath.Join(tempDir, "cyrillic.txt")
	if err := os.WriteFile(testFile, cp1251Content, 0644); err != nil {
		t.Fatal(err)
	}

	// Edit using UTF-8 search text (what Claude sends)
	input := EditFileInput{
		Path:     testFile,
		Edits:    []EditOperation{{OldText: "Невалиден тип.", NewText: "Типът е невалиден."}},
		Encoding: "cp1251",
	}

	result, output, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("edit failed: %v", output)
	}

	// Verify the file was modified correctly
	modifiedData, _ := os.ReadFile(testFile)

	// Expected CP1251: "Типът е невалиден." (The type is invalid.)
	expectedCP1251 := []byte{
		0xD2, 0xE8, 0xEF, 0xFA, 0xF2, // Типът
		0x20,             // space
		0xE5,             // е
		0x20,             // space
		0xED, 0xE5, 0xE2, 0xE0, 0xEB, 0xE8, 0xE4, 0xE5, 0xED, // невалиден
		0x2E, // .
	}

	if string(modifiedData) != string(expectedCP1251) {
		t.Errorf("file content mismatch.\ngot bytes: %v\nwant bytes: %v", modifiedData, expectedCP1251)
	}
}
