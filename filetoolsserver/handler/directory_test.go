package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleListDirectory_WithPattern(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	os.WriteFile(filepath.Join(tempDir, "file1.pas"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file2.pas"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file3.dfm"), []byte("test"), 0644)

	input := ListDirectoryInput{
		Path:    tempDir,
		Pattern: "*.pas",
	}

	result, output, err := h.HandleListDirectory(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	fileList := strings.Join(output.Files, " ")
	if !strings.Contains(fileList, "file1.pas") || !strings.Contains(fileList, "file2.pas") {
		t.Errorf("expected .pas files in result, got %q", fileList)
	}
	if strings.Contains(fileList, "file3.dfm") {
		t.Errorf("did not expect .dfm file in result, got %q", fileList)
	}
}

func TestHandleListDirectory_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	emptyDir := filepath.Join(tempDir, "empty")
	os.Mkdir(emptyDir, 0755)

	input := ListDirectoryInput{Path: emptyDir}

	result, output, err := h.HandleListDirectory(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Error("expected success")
	}
	if output.Files == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(output.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(output.Files))
	}
}

func TestHandleListDirectory_WithSubdirectory(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tempDir, "file.txt"), []byte("test"), 0644)

	input := ListDirectoryInput{
		Path: tempDir,
	}

	result, output, err := h.HandleListDirectory(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	fileList := strings.Join(output.Files, " ")
	if !strings.Contains(fileList, "[DIR] subdir") {
		t.Errorf("expected directory marker for subdir, got %q", fileList)
	}
}
