package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHandleListDirectory_AllFiles(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()

	// Create test files
	files := []string{"file1.txt", "file2.pas", "file3.dfm"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	params := &mcp.CallToolParamsFor[ListDirectoryInput]{
		Arguments: ListDirectoryInput{
			Path: dir,
		},
	}

	result, err := h.HandleListDirectory(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	text := extractText(result.Content)
	for _, f := range files {
		if !strings.Contains(text, f) {
			t.Errorf("expected file %q in result, got %q", f, text)
		}
	}
}

func TestHandleListDirectory_WithPattern(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()

	// Create test files
	if err := os.WriteFile(filepath.Join(dir, "file1.pas"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file2.pas"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file3.dfm"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	params := &mcp.CallToolParamsFor[ListDirectoryInput]{
		Arguments: ListDirectoryInput{
			Path:    dir,
			Pattern: "*.pas",
		},
	}

	result, err := h.HandleListDirectory(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "file1.pas") || !strings.Contains(text, "file2.pas") {
		t.Errorf("expected .pas files in result, got %q", text)
	}
	if strings.Contains(text, "file3.dfm") {
		t.Errorf("did not expect .dfm file in result, got %q", text)
	}
}

func TestHandleListDirectory_WithSubdirectory(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()

	// Create a subdirectory
	subdir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	params := &mcp.CallToolParamsFor[ListDirectoryInput]{
		Arguments: ListDirectoryInput{
			Path: dir,
		},
	}

	result, err := h.HandleListDirectory(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "[DIR] subdir") {
		t.Errorf("expected directory marker for subdir, got %q", text)
	}
	if !strings.Contains(text, "file.txt") {
		t.Errorf("expected file.txt in result, got %q", text)
	}
}

func TestHandleListDirectory_EmptyDirectory(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()

	params := &mcp.CallToolParamsFor[ListDirectoryInput]{
		Arguments: ListDirectoryInput{
			Path: dir,
		},
	}

	result, err := h.HandleListDirectory(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "No files found") {
		t.Errorf("expected 'No files found' message, got %q", text)
	}
}

func TestHandleListDirectory_NotFound(t *testing.T) {
	h := NewHandler()

	params := &mcp.CallToolParamsFor[ListDirectoryInput]{
		Arguments: ListDirectoryInput{
			Path: "/nonexistent/directory",
		},
	}

	result, err := h.HandleListDirectory(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for nonexistent directory")
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "failed to read directory") {
		t.Errorf("expected 'failed to read directory' message, got %q", text)
	}
}

func TestHandleListDirectory_EmptyPath(t *testing.T) {
	h := NewHandler()

	params := &mcp.CallToolParamsFor[ListDirectoryInput]{
		Arguments: ListDirectoryInput{
			Path: "",
		},
	}

	result, err := h.HandleListDirectory(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for empty path")
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "path is required") {
		t.Errorf("expected 'path is required' message, got %q", text)
	}
}

func TestHandleListDirectory_InvalidPattern(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()

	// Create a file
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	params := &mcp.CallToolParamsFor[ListDirectoryInput]{
		Arguments: ListDirectoryInput{
			Path:    dir,
			Pattern: "[invalid",
		},
	}

	result, err := h.HandleListDirectory(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for invalid pattern")
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "invalid pattern") {
		t.Errorf("expected 'invalid pattern' message, got %q", text)
	}
}
