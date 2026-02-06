package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHandleCopyFile_Success(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	src := filepath.Join(tempDir, "source.txt")
	dst := filepath.Join(tempDir, "dest.txt")
	os.WriteFile(src, []byte("content"), 0644)

	// Set a specific mod time on source to verify preservation
	fixedTime := time.Date(2020, 6, 15, 12, 0, 0, 0, time.UTC)
	os.Chtimes(src, fixedTime, fixedTime)

	result, _, err := h.HandleCopyFile(context.Background(), nil, CopyFileInput{Source: src, Destination: dst})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Error("expected success")
	}

	if _, err := os.Stat(src); err != nil {
		t.Error("source should still exist")
	}

	content, err := os.ReadFile(dst)
	if err != nil {
		t.Error("destination should exist")
	}
	if string(content) != "content" {
		t.Errorf("wrong content: %s", content)
	}

	// Verify mod time is preserved
	dstInfo, _ := os.Stat(dst)
	if !dstInfo.ModTime().Equal(fixedTime) {
		t.Errorf("mod time not preserved: got %v, want %v", dstInfo.ModTime(), fixedTime)
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
