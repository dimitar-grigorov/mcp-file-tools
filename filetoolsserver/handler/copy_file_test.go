package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleCopyFile_Success(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	src := filepath.Join(tempDir, "source.txt")
	dst := filepath.Join(tempDir, "dest.txt")
	os.WriteFile(src, []byte("content"), 0644)

	result, _, err := h.HandleCopyFile(context.Background(), nil, CopyFileInput{Source: src, Destination: dst})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Error("expected success")
	}

	// Source should still exist
	if _, err := os.Stat(src); err != nil {
		t.Error("source should still exist")
	}

	// Destination should exist with same content
	content, err := os.ReadFile(dst)
	if err != nil {
		t.Error("destination should exist")
	}
	if string(content) != "content" {
		t.Errorf("wrong content: %s", content)
	}
}

func TestHandleCopyFile_SourceNotExists(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	result, _, err := h.HandleCopyFile(context.Background(), nil, CopyFileInput{
		Source:      filepath.Join(tempDir, "nonexistent.txt"),
		Destination: filepath.Join(tempDir, "dest.txt"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for non-existent source")
	}
}

func TestHandleCopyFile_DestinationExists(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	src := filepath.Join(tempDir, "source.txt")
	dst := filepath.Join(tempDir, "dest.txt")
	os.WriteFile(src, []byte("source"), 0644)
	os.WriteFile(dst, []byte("existing"), 0644)

	result, _, err := h.HandleCopyFile(context.Background(), nil, CopyFileInput{Source: src, Destination: dst})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error when destination exists")
	}
}

func TestHandleCopyFile_SourceIsDirectory(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	src := filepath.Join(tempDir, "srcdir")
	os.Mkdir(src, 0755)

	result, _, err := h.HandleCopyFile(context.Background(), nil, CopyFileInput{
		Source:      src,
		Destination: filepath.Join(tempDir, "dest.txt"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for directory source")
	}
}

func TestHandleCopyFile_OutsideAllowed(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	result, _, err := h.HandleCopyFile(context.Background(), nil, CopyFileInput{
		Source:      "/some/random/file.txt",
		Destination: filepath.Join(tempDir, "dest.txt"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for path outside allowed directories")
	}
}
