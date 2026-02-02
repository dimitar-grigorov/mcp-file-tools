package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleTree_BasicOutput(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create structure: src/handler/read.go, src/server.go, README.md
	os.MkdirAll(filepath.Join(tempDir, "src", "handler"), 0755)
	os.WriteFile(filepath.Join(tempDir, "src", "handler", "read.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tempDir, "src", "server.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tempDir, "README.md"), []byte(""), 0644)

	input := TreeInput{Path: tempDir}
	_, output, err := h.HandleTree(context.Background(), nil, input)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output.FileCount != 3 {
		t.Errorf("expected 3 files, got %d", output.FileCount)
	}
	if output.DirCount != 2 {
		t.Errorf("expected 2 dirs, got %d", output.DirCount)
	}
	// Check indented format
	if !strings.Contains(output.Tree, "src/") {
		t.Error("expected 'src/' in output")
	}
	if !strings.Contains(output.Tree, "  handler/") {
		t.Error("expected indented 'handler/' in output")
	}
}

func TestHandleTree_MaxDepth(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	os.MkdirAll(filepath.Join(tempDir, "a", "b", "c"), 0755)
	os.WriteFile(filepath.Join(tempDir, "a", "b", "c", "deep.txt"), []byte(""), 0644)

	input := TreeInput{Path: tempDir, MaxDepth: 2}
	_, output, _ := h.HandleTree(context.Background(), nil, input)

	// Should see a/ and a/b/ but not a/b/c/
	if !strings.Contains(output.Tree, "a/") {
		t.Error("expected 'a/' at depth 1")
	}
	if !strings.Contains(output.Tree, "  b/") {
		t.Error("expected 'b/' at depth 2")
	}
	if strings.Contains(output.Tree, "c/") {
		t.Error("should NOT see 'c/' at depth 3 when maxDepth=2")
	}
}

func TestHandleTree_MaxFiles(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	for i := 0; i < 20; i++ {
		os.WriteFile(filepath.Join(tempDir, string(rune('a'+i))+".txt"), []byte(""), 0644)
	}

	input := TreeInput{Path: tempDir, MaxFiles: 5}
	_, output, _ := h.HandleTree(context.Background(), nil, input)

	if output.FileCount+output.DirCount > 5 {
		t.Errorf("expected max 5 entries, got %d", output.FileCount+output.DirCount)
	}
	if !output.Truncated {
		t.Error("expected truncated=true")
	}
}

func TestHandleTree_DirsOnly(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	os.MkdirAll(filepath.Join(tempDir, "src"), 0755)
	os.WriteFile(filepath.Join(tempDir, "file.txt"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tempDir, "src", "code.go"), []byte(""), 0644)

	input := TreeInput{Path: tempDir, DirsOnly: true}
	_, output, _ := h.HandleTree(context.Background(), nil, input)

	if output.FileCount != 0 {
		t.Errorf("expected 0 files with dirsOnly, got %d", output.FileCount)
	}
	if output.DirCount != 1 {
		t.Errorf("expected 1 dir, got %d", output.DirCount)
	}
}

func TestHandleTree_Exclude(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	os.MkdirAll(filepath.Join(tempDir, "node_modules"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "src"), 0755)
	os.WriteFile(filepath.Join(tempDir, "node_modules", "pkg.js"), []byte(""), 0644)

	input := TreeInput{Path: tempDir, Exclude: []string{"node_modules"}}
	_, output, _ := h.HandleTree(context.Background(), nil, input)

	if strings.Contains(output.Tree, "node_modules") {
		t.Error("expected node_modules to be excluded")
	}
	if !strings.Contains(output.Tree, "src/") {
		t.Error("expected src/ to be present")
	}
}
