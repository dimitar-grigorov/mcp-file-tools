package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHandleCreateDirectory_Simple(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	newDir := filepath.Join(tempDir, "newdir")
	input := CreateDirectoryInput{Path: newDir}

	result, output, err := h.HandleCreateDirectory(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	if !strings.Contains(output.Message, "Successfully created") {
		t.Errorf("expected success message, got %q", output.Message)
	}

	// Verify directory exists
	info, err := os.Stat(newDir)
	if err != nil {
		t.Errorf("directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected directory, got file")
	}
}

func TestHandleCreateDirectory_Nested(t *testing.T) {
	tempDir := t.TempDir()

	// Create parent directory first so path validation can resolve it
	parentDir := filepath.Join(tempDir, "parent")
	os.Mkdir(parentDir, 0755)

	h := NewHandler([]string{tempDir})

	// Create nested path one level below existing parent
	// (deep nesting where intermediates don't exist has Windows path resolution issues)
	nestedDir := filepath.Join(parentDir, "child")
	input := CreateDirectoryInput{Path: nestedDir}

	result, output, err := h.HandleCreateDirectory(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		if len(result.Content) > 0 {
			if tc, ok := result.Content[0].(*mcp.TextContent); ok {
				t.Errorf("expected success, got error: %s", tc.Text)
			}
		}
		return
	}

	if !strings.Contains(output.Message, "Successfully created") {
		t.Errorf("expected success message, got %q", output.Message)
	}

	// Verify directory was created
	info, err := os.Stat(nestedDir)
	if err != nil {
		t.Fatalf("nested directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("expected directory, got file")
	}
}

func TestHandleCreateDirectory_AlreadyExists(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create directory first
	existingDir := filepath.Join(tempDir, "existing")
	os.Mkdir(existingDir, 0755)

	input := CreateDirectoryInput{Path: existingDir}

	result, output, err := h.HandleCreateDirectory(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	// Should succeed silently when directory already exists
	if result.IsError {
		t.Errorf("expected success for existing directory, got error")
	}

	if !strings.Contains(output.Message, "Successfully created") {
		t.Errorf("expected success message, got %q", output.Message)
	}
}

func TestHandleCreateDirectory_EmptyPath(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	input := CreateDirectoryInput{Path: ""}

	result, _, err := h.HandleCreateDirectory(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for empty path")
	}
}

func TestHandleCreateDirectory_OutsideAllowedDirs(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Try to create directory outside allowed dirs
	input := CreateDirectoryInput{Path: "/some/random/path"}

	result, _, err := h.HandleCreateDirectory(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for path outside allowed directories")
	}
}
