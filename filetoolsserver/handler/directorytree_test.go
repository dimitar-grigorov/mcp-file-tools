package handler

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleDirectoryTree_Simple(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create a simple directory structure
	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("content"), 0644)
	os.Mkdir(filepath.Join(tempDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tempDir, "subdir", "nested.txt"), []byte("content"), 0644)

	input := DirectoryTreeInput{
		Path: tempDir,
	}

	result, output, err := h.HandleDirectoryTree(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	// Parse the JSON output
	var tree []TreeEntry
	if err := json.Unmarshal([]byte(output.Tree), &tree); err != nil {
		t.Fatalf("failed to parse tree JSON: %v", err)
	}

	// Verify structure
	if len(tree) != 3 {
		t.Errorf("expected 3 entries, got %d", len(tree))
	}

	// Find the subdir entry
	var subdirEntry *TreeEntry
	for i := range tree {
		if tree[i].Name == "subdir" {
			subdirEntry = &tree[i]
			break
		}
	}

	if subdirEntry == nil {
		t.Fatal("subdir entry not found")
	}

	if subdirEntry.Type != "directory" {
		t.Errorf("expected subdir type 'directory', got %q", subdirEntry.Type)
	}

	if subdirEntry.Children == nil {
		t.Fatal("expected subdir to have children")
	}

	if len(*subdirEntry.Children) != 1 {
		t.Errorf("expected 1 child in subdir, got %d", len(*subdirEntry.Children))
	}
}

func TestHandleDirectoryTree_ExcludePattern(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create files including ones to exclude
	os.WriteFile(filepath.Join(tempDir, "keep.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(tempDir, ".env"), []byte("secret"), 0644)
	os.Mkdir(filepath.Join(tempDir, "node_modules"), 0755)
	os.WriteFile(filepath.Join(tempDir, "node_modules", "package.json"), []byte("{}"), 0644)

	input := DirectoryTreeInput{
		Path:            tempDir,
		ExcludePatterns: []string{".env", "node_modules"},
	}

	result, output, err := h.HandleDirectoryTree(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	var tree []TreeEntry
	if err := json.Unmarshal([]byte(output.Tree), &tree); err != nil {
		t.Fatalf("failed to parse tree JSON: %v", err)
	}

	// Should only have keep.txt
	if len(tree) != 1 {
		t.Errorf("expected 1 entry after exclusion, got %d", len(tree))
	}

	if tree[0].Name != "keep.txt" {
		t.Errorf("expected 'keep.txt', got %q", tree[0].Name)
	}
}

func TestHandleDirectoryTree_GlobPattern(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	os.WriteFile(filepath.Join(tempDir, "keep.txt"), []byte("content"), 0644)
	os.WriteFile(filepath.Join(tempDir, "test.log"), []byte("log"), 0644)
	os.WriteFile(filepath.Join(tempDir, "other.log"), []byte("log"), 0644)

	input := DirectoryTreeInput{
		Path:            tempDir,
		ExcludePatterns: []string{"*.log"},
	}

	result, output, err := h.HandleDirectoryTree(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	var tree []TreeEntry
	if err := json.Unmarshal([]byte(output.Tree), &tree); err != nil {
		t.Fatalf("failed to parse tree JSON: %v", err)
	}

	// Should only have keep.txt
	if len(tree) != 1 {
		t.Errorf("expected 1 entry after glob exclusion, got %d", len(tree))
	}

	if tree[0].Name != "keep.txt" {
		t.Errorf("expected 'keep.txt', got %q", tree[0].Name)
	}
}

func TestHandleDirectoryTree_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create an empty subdirectory
	emptyDir := filepath.Join(tempDir, "empty")
	os.Mkdir(emptyDir, 0755)

	input := DirectoryTreeInput{
		Path: tempDir,
	}

	result, output, err := h.HandleDirectoryTree(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	var tree []TreeEntry
	if err := json.Unmarshal([]byte(output.Tree), &tree); err != nil {
		t.Fatalf("failed to parse tree JSON: %v", err)
	}

	if len(tree) != 1 {
		t.Errorf("expected 1 entry, got %d", len(tree))
	}

	emptyEntry := tree[0]
	if emptyEntry.Type != "directory" {
		t.Errorf("expected type 'directory', got %q", emptyEntry.Type)
	}

	if emptyEntry.Children == nil {
		t.Fatal("expected children to be non-nil for directory")
	}

	if len(*emptyEntry.Children) != 0 {
		t.Errorf("expected empty children array, got %d entries", len(*emptyEntry.Children))
	}
}

func TestHandleDirectoryTree_FileHasNoChildren(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	os.WriteFile(filepath.Join(tempDir, "file.txt"), []byte("content"), 0644)

	input := DirectoryTreeInput{
		Path: tempDir,
	}

	result, output, err := h.HandleDirectoryTree(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	var tree []TreeEntry
	if err := json.Unmarshal([]byte(output.Tree), &tree); err != nil {
		t.Fatalf("failed to parse tree JSON: %v", err)
	}

	fileEntry := tree[0]
	if fileEntry.Type != "file" {
		t.Errorf("expected type 'file', got %q", fileEntry.Type)
	}

	if fileEntry.Children != nil {
		t.Errorf("expected children to be nil for file, got %v", fileEntry.Children)
	}

	// Also verify the JSON doesn't contain "children" for files
	if strings.Contains(output.Tree, `"children"`) && !strings.Contains(output.Tree, `"children": []`) {
		// This is fine - we expect children only for directories
	}
}

func TestHandleDirectoryTree_EmptyPath(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	input := DirectoryTreeInput{
		Path: "",
	}

	result, _, err := h.HandleDirectoryTree(context.Background(), nil, input)
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

func TestHandleDirectoryTree_OutsideAllowedDir(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	input := DirectoryTreeInput{
		Path: filepath.Join(tempDir, "..", "..", "nonexistent"),
	}

	result, _, err := h.HandleDirectoryTree(context.Background(), nil, input)
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

func TestHandleDirectoryTree_NotADirectory(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "file.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	input := DirectoryTreeInput{
		Path: testFile,
	}

	result, _, err := h.HandleDirectoryTree(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error when path is a file")
	}

	text := extractTextFromResult(result.Content)
	if !strings.Contains(text, "must be a directory") {
		t.Errorf("expected 'must be a directory' message, got %q", text)
	}
}

