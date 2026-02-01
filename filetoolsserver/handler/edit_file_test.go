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

func TestHandleEditFile_CRLFNormalization(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("line1\r\nline2"), 0644)

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
