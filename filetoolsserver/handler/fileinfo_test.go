package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleGetFileInfo_File(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")
	content := "Hello, World!"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	input := GetFileInfoInput{
		Path: testFile,
	}

	result, output, err := h.HandleGetFileInfo(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	if output.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), output.Size)
	}

	if !output.IsFile {
		t.Errorf("expected IsFile to be true")
	}

	if output.IsDirectory {
		t.Errorf("expected IsDirectory to be false")
	}

	if output.Created == "" {
		t.Errorf("expected Created to be set")
	}

	if output.Modified == "" {
		t.Errorf("expected Modified to be set")
	}

	if output.Accessed == "" {
		t.Errorf("expected Accessed to be set")
	}

	if output.Permissions == "" {
		t.Errorf("expected Permissions to be set")
	}
}

func TestHandleGetFileInfo_Directory(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testDir := filepath.Join(tempDir, "subdir")

	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	input := GetFileInfoInput{
		Path: testDir,
	}

	result, output, err := h.HandleGetFileInfo(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	if output.IsFile {
		t.Errorf("expected IsFile to be false for directory")
	}

	if !output.IsDirectory {
		t.Errorf("expected IsDirectory to be true for directory")
	}
}

func TestHandleGetFileInfo_EmptyPath(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	input := GetFileInfoInput{
		Path: "",
	}

	result, _, err := h.HandleGetFileInfo(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for empty path")
	}

	text := extractTextFromResult(result.Content)
	if !strings.Contains(text, "path is required") {
		t.Errorf("expected 'path is required' message, got %q", text)
	}
}

func TestHandleGetFileInfo_OutsideAllowedDir(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	input := GetFileInfoInput{
		Path: filepath.Join(tempDir, "..", "..", "nonexistent", "file.txt"),
	}

	result, _, err := h.HandleGetFileInfo(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for path outside allowed directories")
	}

	text := extractTextFromResult(result.Content)
	if !strings.Contains(text, "access denied") {
		t.Errorf("expected 'access denied' message, got %q", text)
	}
}

func TestHandleGetFileInfo_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	input := GetFileInfoInput{
		Path: filepath.Join(tempDir, "nonexistent.txt"),
	}

	result, _, err := h.HandleGetFileInfo(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for non-existent file")
	}

	text := extractTextFromResult(result.Content)
	if !strings.Contains(text, "failed to get file info") {
		t.Errorf("expected 'failed to get file info' message, got %q", text)
	}
}
