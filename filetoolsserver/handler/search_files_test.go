// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"fmt"
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

func TestMatchGlobPattern(t *testing.T) {
	tests := []struct {
		path, pattern string
		want          bool
	}{
		{"src/a/test/b/x.go", "src/**/test/**/*.go", true},
		{"src/a/x.go", "src/**/test/**/*.go", false},
		{"a/b/x.go", "**/*.go", true},
		{"x.go", "**/*.go", true},
		{"a/b/c.pas", "*.pas", true},
		{"a/b/c.pas", "*.dfm", false},
		{"dir/sub/f.txt", "dir/**", true},
		{"dir", "dir/**", true},
		{"other/f.txt", "dir/**", false},
		{"dir/a/b/f.txt", "dir/**/f.txt", true},
		{"dir/f.txt", "dir/**/f.txt", true},
		{"src/x.go", "src/*.go", true},
		{"src/sub/x.go", "src/*.go", false},
	}
	for _, tt := range tests {
		if got := matchGlobPattern(tt.path, tt.pattern); got != tt.want {
			t.Errorf("matchGlobPattern(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
		}
	}
}

func TestHandleSearchFiles_MultipleDoubleStars(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	deep := filepath.Join(tempDir, "src", "a", "test", "b")
	os.MkdirAll(deep, 0755)
	os.WriteFile(filepath.Join(deep, "x.go"), []byte("t"), 0644)
	os.WriteFile(filepath.Join(tempDir, "src", "y.go"), []byte("t"), 0644)

	input := SearchFilesInput{Path: tempDir, Pattern: "src/**/test/**/*.go"}
	_, output, err := h.HandleSearchFiles(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Files) != 1 {
		t.Errorf("expected 1 file, got %d: %v", len(output.Files), output.Files)
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

func TestHandleSearchFiles_MaxResults(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	for i := 0; i < 20; i++ {
		os.WriteFile(filepath.Join(tempDir, fmt.Sprintf("file%03d.txt", i)), []byte("test"), 0644)
	}

	result, output, err := h.HandleSearchFiles(context.Background(), nil, SearchFilesInput{
		Path:       tempDir,
		Pattern:    "*.txt",
		MaxResults: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Error("expected success")
	}
	if len(output.Files) != 5 {
		t.Errorf("expected 5 files (max), got %d", len(output.Files))
	}
	if !output.Truncated {
		t.Error("expected truncated to be true")
	}
}
