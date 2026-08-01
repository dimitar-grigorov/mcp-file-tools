// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// No tool may cross a directory link that leaves the allowed directories.
func TestTraversal_DirectoryLinkEscapeIsSkipped(t *testing.T) {
	allowedDir := t.TempDir()
	outsideDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("classified"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(allowedDir, "keep.txt"), []byte("public"), 0644); err != nil {
		t.Fatal(err)
	}
	createDirLink(t, outsideDir, filepath.Join(allowedDir, "escape"))
	h := NewHandler([]string{allowedDir})

	_, treeOut, err := h.HandleTree(context.Background(), nil, TreeInput{Path: allowedDir})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(treeOut.Tree, "escape") || strings.Contains(treeOut.Tree, "secret.txt") {
		t.Errorf("tree exposed the escape: %q", treeOut.Tree)
	}

	_, searchOut, err := h.HandleSearchFiles(context.Background(), nil, SearchFilesInput{Path: allowedDir, Pattern: "*.txt"})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range searchOut.Files {
		if strings.Contains(f, "secret") {
			t.Errorf("search_files exposed %q", f)
		}
	}

	_, grepOut, err := h.HandleGrep(context.Background(), nil, GrepInput{Paths: []string{allowedDir}, Pattern: "classified|public"})
	if err != nil {
		t.Fatal(err)
	}
	if grepOut.FilesSearched != 1 {
		t.Errorf("grep searched %d files, want only the one inside the allowed dir", grepOut.FilesSearched)
	}
	for _, m := range grepOut.Matches {
		if strings.Contains(m.Path, "secret") {
			t.Errorf("grep exposed %q", m.Path)
		}
	}
}

func TestTraversal_CancelledContext(t *testing.T) {
	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "sub"), 0755)
	os.WriteFile(filepath.Join(tempDir, "sub", "a.txt"), []byte("x"), 0644)
	h := NewHandler([]string{tempDir})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, out, err := h.HandleTree(ctx, nil, TreeInput{Path: tempDir}); err != nil {
		t.Fatal(err)
	} else if !out.Truncated {
		t.Error("tree: expected truncated=true on a cancelled context")
	}
	if res, _, err := h.HandleSearchFiles(ctx, nil, SearchFilesInput{Path: tempDir, Pattern: "*.txt"}); err != nil {
		t.Fatal(err)
	} else if !res.IsError {
		t.Error("search_files: expected an error result on a cancelled context")
	}
	if _, out, err := h.HandleGrep(ctx, nil, GrepInput{Paths: []string{tempDir}, Pattern: "x"}); err != nil {
		t.Fatal(err)
	} else if len(out.Matches) != 0 {
		t.Error("grep: expected no matches on a cancelled context")
	}
}

func TestHandleTree_ExactIndentedOutput(t *testing.T) {
	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "src", "handler"), 0755)
	os.WriteFile(filepath.Join(tempDir, "README.md"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tempDir, "src", "server.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tempDir, "src", "handler", "read.go"), []byte(""), 0644)
	h := NewHandler([]string{tempDir})

	_, out, err := h.HandleTree(context.Background(), nil, TreeInput{Path: tempDir})
	if err != nil {
		t.Fatal(err)
	}
	want := "README.md\nsrc/\n  handler/\n    read.go\n  server.go\n"
	if out.Tree != want {
		t.Errorf("tree =\n%q\nwant\n%q", out.Tree, want)
	}
}

// search_files must still see exactly what a plain recursive walk sees.
func TestHandleSearchFiles_MatchesReferenceWalk(t *testing.T) {
	tempDir := t.TempDir()
	for _, dir := range []string{"a", "a/b", "a/b/c", "z"} {
		os.MkdirAll(filepath.Join(tempDir, filepath.FromSlash(dir)), 0755)
	}
	for _, file := range []string{"top.txt", "a/one.txt", "a/b/two.txt", "a/b/c/three.txt", "z/four.md"} {
		os.WriteFile(filepath.Join(tempDir, filepath.FromSlash(file)), []byte("x"), 0644)
	}
	h := NewHandler([]string{tempDir})

	var want []string
	filepath.WalkDir(tempDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".txt" {
			return err
		}
		rel, _ := filepath.Rel(tempDir, path)
		want = append(want, filepath.ToSlash(rel))
		return nil
	})

	_, out, err := h.HandleSearchFiles(context.Background(), nil, SearchFilesInput{Path: tempDir, Pattern: "**/*.txt"})
	if err != nil {
		t.Fatal(err)
	}
	// ValidatePath may normalize the root (8.3 names), so compare suffixes.
	if len(out.Files) != len(want) {
		t.Fatalf("files = %v, want %d entries matching %v", out.Files, len(want), want)
	}
	for i, f := range out.Files {
		if !strings.HasSuffix(filepath.ToSlash(f), "/"+want[i]) {
			t.Errorf("files[%d] = %q, want it to end with %q", i, f, want[i])
		}
	}
}

// createDirLink creates a directory symlink, falling back to a junction on Windows.
func createDirLink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err == nil {
		return
	} else if runtime.GOOS != "windows" {
		t.Skipf("directory symlinks are not supported in this environment: %v", err)
	}
	output, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
	if err != nil {
		t.Skipf("directory junctions are not supported in this environment: %v (%s)", err, output)
	}
}
