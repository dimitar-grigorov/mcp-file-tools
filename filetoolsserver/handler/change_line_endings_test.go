package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleChangeLineEndings_CRLFtoLF(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("line1\r\nline2\r\nline3\r\n"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ChangeLineEndingsInput{Path: testFile, Style: "lf"}
	result, output, err := h.HandleChangeLineEndings(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error")
	}

	if output.OriginalStyle != "crlf" {
		t.Errorf("expected originalStyle=crlf, got %s", output.OriginalStyle)
	}
	if output.NewStyle != "lf" {
		t.Errorf("expected newStyle=lf, got %s", output.NewStyle)
	}
	if output.LinesChanged != 3 {
		t.Errorf("expected linesChanged=3, got %d", output.LinesChanged)
	}

	// Verify file content
	data, _ := os.ReadFile(testFile)
	if string(data) != "line1\nline2\nline3\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestHandleChangeLineEndings_LFtoCRLF(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ChangeLineEndingsInput{Path: testFile, Style: "crlf"}
	result, output, err := h.HandleChangeLineEndings(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error")
	}

	if output.OriginalStyle != "lf" {
		t.Errorf("expected originalStyle=lf, got %s", output.OriginalStyle)
	}
	if output.LinesChanged != 3 {
		t.Errorf("expected linesChanged=3, got %d", output.LinesChanged)
	}

	data, _ := os.ReadFile(testFile)
	if string(data) != "line1\r\nline2\r\nline3\r\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestHandleChangeLineEndings_MixedToLF(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	// Mix of CRLF and LF
	if err := os.WriteFile(testFile, []byte("line1\r\nline2\nline3\r\n"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ChangeLineEndingsInput{Path: testFile, Style: "lf"}
	result, output, err := h.HandleChangeLineEndings(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error")
	}

	if output.OriginalStyle != "mixed" {
		t.Errorf("expected originalStyle=mixed, got %s", output.OriginalStyle)
	}
	if output.LinesChanged != 2 {
		t.Errorf("expected linesChanged=2 (CRLF lines), got %d", output.LinesChanged)
	}

	data, _ := os.ReadFile(testFile)
	if string(data) != "line1\nline2\nline3\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestHandleChangeLineEndings_AlreadyCorrect(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("line1\nline2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ChangeLineEndingsInput{Path: testFile, Style: "lf"}
	result, output, err := h.HandleChangeLineEndings(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error")
	}

	if output.LinesChanged != 0 {
		t.Errorf("expected linesChanged=0 for no-op, got %d", output.LinesChanged)
	}
	if output.OriginalStyle != "lf" {
		t.Errorf("expected originalStyle=lf, got %s", output.OriginalStyle)
	}
}

func TestHandleChangeLineEndings_InvalidStyle(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ChangeLineEndingsInput{Path: testFile, Style: "mac"}
	result, _, err := h.HandleChangeLineEndings(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Errorf("expected error for invalid style")
	}
}

func TestHandleChangeLineEndings_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("no newlines here"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ChangeLineEndingsInput{Path: testFile, Style: "lf"}
	result, output, err := h.HandleChangeLineEndings(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error")
	}

	if output.LinesChanged != 0 {
		t.Errorf("expected linesChanged=0 for file with no line endings, got %d", output.LinesChanged)
	}
}
