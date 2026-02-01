package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleMoveFile_MoveToNewLocation(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create source file
	srcFile := filepath.Join(tempDir, "source.txt")
	os.WriteFile(srcFile, []byte("test content"), 0644)

	// Create destination directory
	destDir := filepath.Join(tempDir, "subdir")
	os.Mkdir(destDir, 0755)

	destFile := filepath.Join(destDir, "moved.txt")
	input := MoveFileInput{Source: srcFile, Destination: destFile}

	result, output, err := h.HandleMoveFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	if !strings.Contains(output.Message, "Successfully moved") {
		t.Errorf("expected success message, got %q", output.Message)
	}

	// Verify source no longer exists
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Errorf("source file should not exist after move")
	}

	// Verify destination exists with correct content
	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Errorf("destination file should exist: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("destination file has wrong content: %q", content)
	}
}

func TestHandleMoveFile_RenameInSameDirectory(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create source file
	srcFile := filepath.Join(tempDir, "original.txt")
	os.WriteFile(srcFile, []byte("rename test"), 0644)

	destFile := filepath.Join(tempDir, "renamed.txt")
	input := MoveFileInput{Source: srcFile, Destination: destFile}

	result, output, err := h.HandleMoveFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	if !strings.Contains(output.Message, "Successfully moved") {
		t.Errorf("expected success message, got %q", output.Message)
	}

	// Verify source no longer exists
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Errorf("original file should not exist after rename")
	}

	// Verify renamed file exists
	content, err := os.ReadFile(destFile)
	if err != nil {
		t.Errorf("renamed file should exist: %v", err)
	}
	if string(content) != "rename test" {
		t.Errorf("renamed file has wrong content: %q", content)
	}
}

func TestHandleMoveFile_MoveDirectory(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create source directory with a file inside
	srcDir := filepath.Join(tempDir, "srcdir")
	os.Mkdir(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("dir content"), 0644)

	destDir := filepath.Join(tempDir, "destdir")
	input := MoveFileInput{Source: srcDir, Destination: destDir}

	result, output, err := h.HandleMoveFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	if !strings.Contains(output.Message, "Successfully moved") {
		t.Errorf("expected success message, got %q", output.Message)
	}

	// Verify source directory no longer exists
	if _, err := os.Stat(srcDir); !os.IsNotExist(err) {
		t.Errorf("source directory should not exist after move")
	}

	// Verify destination directory exists with file inside
	info, err := os.Stat(destDir)
	if err != nil {
		t.Errorf("destination directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("destination should be a directory")
	}

	// Verify file inside moved directory
	content, err := os.ReadFile(filepath.Join(destDir, "file.txt"))
	if err != nil {
		t.Errorf("file inside moved directory should exist: %v", err)
	}
	if string(content) != "dir content" {
		t.Errorf("file inside moved directory has wrong content: %q", content)
	}
}

func TestHandleMoveFile_DestinationExists(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create source file
	srcFile := filepath.Join(tempDir, "source.txt")
	os.WriteFile(srcFile, []byte("source content"), 0644)

	// Create destination file (already exists)
	destFile := filepath.Join(tempDir, "dest.txt")
	os.WriteFile(destFile, []byte("existing content"), 0644)

	input := MoveFileInput{Source: srcFile, Destination: destFile}

	result, _, err := h.HandleMoveFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error when destination exists")
	}

	// Verify both files still exist with original content
	srcContent, _ := os.ReadFile(srcFile)
	if string(srcContent) != "source content" {
		t.Errorf("source file should be unchanged")
	}

	destContent, _ := os.ReadFile(destFile)
	if string(destContent) != "existing content" {
		t.Errorf("destination file should be unchanged")
	}
}

func TestHandleMoveFile_SourceOutsideAllowedDirs(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Try to move file from outside allowed dirs
	input := MoveFileInput{
		Source:      "/some/random/path/file.txt",
		Destination: filepath.Join(tempDir, "dest.txt"),
	}

	result, _, err := h.HandleMoveFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for source outside allowed directories")
	}
}

func TestHandleMoveFile_DestinationOutsideAllowedDirs(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create source file
	srcFile := filepath.Join(tempDir, "source.txt")
	os.WriteFile(srcFile, []byte("test"), 0644)

	// Try to move file to outside allowed dirs
	input := MoveFileInput{
		Source:      srcFile,
		Destination: "/some/random/path/file.txt",
	}

	result, _, err := h.HandleMoveFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for destination outside allowed directories")
	}
}

func TestHandleMoveFile_EmptySource(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	input := MoveFileInput{
		Source:      "",
		Destination: filepath.Join(tempDir, "dest.txt"),
	}

	result, _, err := h.HandleMoveFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for empty source")
	}
}

func TestHandleMoveFile_EmptyDestination(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	srcFile := filepath.Join(tempDir, "source.txt")
	os.WriteFile(srcFile, []byte("test"), 0644)

	input := MoveFileInput{
		Source:      srcFile,
		Destination: "",
	}

	result, _, err := h.HandleMoveFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for empty destination")
	}
}

func TestHandleMoveFile_SourceNotExists(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	input := MoveFileInput{
		Source:      filepath.Join(tempDir, "nonexistent.txt"),
		Destination: filepath.Join(tempDir, "dest.txt"),
	}

	result, _, err := h.HandleMoveFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for non-existent source")
	}
}
