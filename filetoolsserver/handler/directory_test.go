package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Helper to extract text from MCP content
func extractTextFromResultDir(content []mcp.Content) string {
	for _, c := range content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func TestHandleListDirectory_AllFiles(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create test files
	files := []string{"file1.txt", "file2.pas", "file3.dfm"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tempDir, f), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	input := ListDirectoryInput{
		Path: tempDir,
	}

	result, output, err := h.HandleListDirectory(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	fileList := strings.Join(output.Files, " ")
	for _, f := range files {
		if !strings.Contains(fileList, f) {
			t.Errorf("expected file %q in result, got %q", f, fileList)
		}
	}
}

func TestHandleListDirectory_WithPattern(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create test files
	if err := os.WriteFile(filepath.Join(tempDir, "file1.pas"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "file2.pas"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "file3.dfm"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

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

func TestHandleListDirectory_WithSubdirectory(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create a subdirectory
	subdir := filepath.Join(tempDir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file
	if err := os.WriteFile(filepath.Join(tempDir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

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
	if !strings.Contains(fileList, "file.txt") {
		t.Errorf("expected file.txt in result, got %q", fileList)
	}
}

func TestHandleListDirectory_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

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

	// Check if empty or contains "No files found"
	if len(output.Files) == 0 {
		// Empty list is acceptable
	} else {
		fileList := strings.Join(output.Files, " ")
		if !strings.Contains(fileList, "No files found") {
			t.Errorf("expected empty list or 'No files found' message, got %q", fileList)
		}
	}
}

func TestHandleListDirectory_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Try to access a directory outside allowed directories
	input := ListDirectoryInput{
		Path: filepath.Join(tempDir, "..", "..", "nonexistent", "directory"),
	}

	result, _, err := h.HandleListDirectory(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for directory outside allowed directories")
	}

	text := extractTextFromResultDir(result.Content)
	// Path validation happens first, so we get "access denied" not "failed to read directory"
	if !strings.Contains(text, "access denied") {
		t.Errorf("expected 'access denied' message, got %q", text)
	}
}

func TestHandleListDirectory_EmptyPath(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	input := ListDirectoryInput{
		Path: "",
	}

	result, _, err := h.HandleListDirectory(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for empty path")
	}

	text := extractTextFromResultDir(result.Content)
	if !strings.Contains(text, "path is required") {
		t.Errorf("expected 'path is required' message, got %q", text)
	}
}

func TestHandleListDirectory_InvalidPattern(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create a file
	if err := os.WriteFile(filepath.Join(tempDir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ListDirectoryInput{
		Path:    tempDir,
		Pattern: "[invalid",
	}

	result, _, err := h.HandleListDirectory(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for invalid pattern")
	}

	text := extractTextFromResultDir(result.Content)
	if !strings.Contains(text, "invalid pattern") {
		t.Errorf("expected 'invalid pattern' message, got %q", text)
	}
}
