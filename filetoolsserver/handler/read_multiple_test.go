package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
)

func TestHandleReadMultipleFiles_Success(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)

	input := ReadMultipleFilesInput{Paths: []string{file1, file2}}
	result, output, err := h.HandleReadMultipleFiles(context.Background(), nil, input)

	if err != nil || result.IsError {
		t.Fatal("expected success")
	}
	if output.SuccessCount != 2 || output.ErrorCount != 0 {
		t.Errorf("expected 2 successes, got %d successes, %d errors", output.SuccessCount, output.ErrorCount)
	}
	if output.Results[0].Content != "content1" || output.Results[1].Content != "content2" {
		t.Errorf("unexpected content")
	}
}

func TestHandleReadMultipleFiles_PartialFailure(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	file1 := filepath.Join(tempDir, "exists.txt")
	file2 := filepath.Join(tempDir, "nonexistent.txt")
	os.WriteFile(file1, []byte("content1"), 0644)

	input := ReadMultipleFilesInput{Paths: []string{file1, file2}}
	result, output, _ := h.HandleReadMultipleFiles(context.Background(), nil, input)

	if result.IsError {
		t.Error("expected partial success, not tool error")
	}
	if output.SuccessCount != 1 || output.ErrorCount != 1 {
		t.Errorf("expected 1 success, 1 error, got %d/%d", output.SuccessCount, output.ErrorCount)
	}
}

func TestHandleReadMultipleFiles_EmptyPaths(t *testing.T) {
	h := NewHandler([]string{t.TempDir()})
	result, _, _ := h.HandleReadMultipleFiles(context.Background(), nil, ReadMultipleFilesInput{Paths: []string{}})
	if !result.IsError {
		t.Error("expected error for empty paths")
	}
}

func TestHandleReadMultipleFiles_WithEncoding(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	enc, _ := encoding.Get("cp1251")
	cp1251Bytes, _ := enc.NewEncoder().Bytes([]byte("Здравей свят!"))
	file1 := filepath.Join(tempDir, "cyrillic.txt")
	os.WriteFile(file1, cp1251Bytes, 0644)

	input := ReadMultipleFilesInput{Paths: []string{file1}, Encoding: "cp1251"}
	_, output, _ := h.HandleReadMultipleFiles(context.Background(), nil, input)

	if !strings.Contains(output.Results[0].Content, "Здравей свят!") {
		t.Errorf("expected Cyrillic content, got %q", output.Results[0].Content)
	}
}

func TestHandleReadMultipleFiles_PathOutsideAllowed(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	input := ReadMultipleFilesInput{Paths: []string{filepath.Join(tempDir, "..", "..", "etc", "passwd")}}
	_, output, _ := h.HandleReadMultipleFiles(context.Background(), nil, input)

	if !strings.Contains(output.Results[0].Error, "access denied") {
		t.Errorf("expected 'access denied' error, got %q", output.Results[0].Error)
	}
}
