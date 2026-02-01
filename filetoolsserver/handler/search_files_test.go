package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleSearchFiles_SimplePattern(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file3.go"), []byte("test"), 0644)

	input := SearchFilesInput{Path: tempDir, Pattern: "*.txt"}

	result, output, err := h.HandleSearchFiles(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if len(output.Files) != 2 {
		t.Errorf("expected 2 files, got %d: %v", len(output.Files), output.Files)
	}
}

func TestHandleSearchFiles_RecursivePattern(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	subDir := filepath.Join(tempDir, "subdir")
	os.Mkdir(subDir, 0755)
	deepDir := filepath.Join(subDir, "deep")
	os.Mkdir(deepDir, 0755)

	os.WriteFile(filepath.Join(tempDir, "root.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(subDir, "sub.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(deepDir, "deep.txt"), []byte("test"), 0644)

	input := SearchFilesInput{Path: tempDir, Pattern: "**/*.txt"}

	result, output, err := h.HandleSearchFiles(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if len(output.Files) != 3 {
		t.Errorf("expected 3 files, got %d: %v", len(output.Files), output.Files)
	}
}

func TestHandleSearchFiles_WithExcludePatterns(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	subDir := filepath.Join(tempDir, "subdir")
	os.Mkdir(subDir, 0755)
	nodeModules := filepath.Join(tempDir, "node_modules")
	os.Mkdir(nodeModules, 0755)

	os.WriteFile(filepath.Join(tempDir, "root.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(subDir, "sub.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(nodeModules, "excluded.txt"), []byte("test"), 0644)

	input := SearchFilesInput{
		Path:            tempDir,
		Pattern:         "**/*.txt",
		ExcludePatterns: []string{"node_modules"},
	}

	result, output, err := h.HandleSearchFiles(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if len(output.Files) != 2 {
		t.Errorf("expected 2 files (excluding node_modules), got %d: %v", len(output.Files), output.Files)
	}
}

func TestHandleSearchFiles_NoMatches(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	os.WriteFile(filepath.Join(tempDir, "file.go"), []byte("test"), 0644)

	input := SearchFilesInput{Path: tempDir, Pattern: "*.txt"}

	result, output, err := h.HandleSearchFiles(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if len(output.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(output.Files))
	}
}

func TestHandleSearchFiles_ValidationErrors(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	tests := []struct {
		name  string
		input SearchFilesInput
	}{
		{"empty path", SearchFilesInput{Path: "", Pattern: "*.txt"}},
		{"empty pattern", SearchFilesInput{Path: tempDir, Pattern: ""}},
		{"outside allowed", SearchFilesInput{Path: "/random/path", Pattern: "*.txt"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := h.HandleSearchFiles(context.Background(), nil, tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if !result.IsError {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}
