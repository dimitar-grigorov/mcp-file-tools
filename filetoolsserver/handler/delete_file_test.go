package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleDeleteFile_Success(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	file := filepath.Join(tempDir, "test.txt")
	os.WriteFile(file, []byte("content"), 0644)

	result, _, err := h.HandleDeleteFile(context.Background(), nil, DeleteFileInput{Path: file})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Error("expected success")
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestHandleDeleteFile_NotExists(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	result, _, err := h.HandleDeleteFile(context.Background(), nil, DeleteFileInput{Path: filepath.Join(tempDir, "nonexistent.txt")})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for non-existent file")
	}
}

func TestHandleDeleteFile_IsDirectory(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	dir := filepath.Join(tempDir, "subdir")
	os.Mkdir(dir, 0755)

	result, _, err := h.HandleDeleteFile(context.Background(), nil, DeleteFileInput{Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for directory")
	}
}

func TestHandleDeleteFile_OutsideAllowed(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	result, _, err := h.HandleDeleteFile(context.Background(), nil, DeleteFileInput{Path: "/some/random/file.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for path outside allowed directories")
	}
}
