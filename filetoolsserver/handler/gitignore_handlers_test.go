// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func boolp(b bool) *bool { return &b }

// gitignoreTree: a .gitignore hiding *.dcu and __history/.
func gitignoreTree(t *testing.T) (string, *Handler) {
	t.Helper()
	dir := t.TempDir()
	mk := func(rel, content string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte(content), 0644)
	}
	mk(".gitignore", "*.dcu\n__history/\n")
	mk("main.pas", "procedure Alpha;")
	mk("main.dcu", "binary")
	mk("__history/main.pas.old", "procedure Alpha;")
	return dir, NewHandler([]string{dir})
}

func TestSearchFilesRespectsGitignore(t *testing.T) {
	dir, h := gitignoreTree(t)

	_, out, err := h.HandleSearchFiles(context.Background(), nil, SearchFilesInput{Path: dir, Pattern: "*"})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(out.Files, ";")
	if strings.Contains(joined, "main.dcu") || strings.Contains(joined, "__history") {
		t.Errorf("default should skip ignored entries: %v", out.Files)
	}

	_, out, err = h.HandleSearchFiles(context.Background(), nil, SearchFilesInput{
		Path: dir, Pattern: "*", RespectGitignore: boolp(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	if joined := strings.Join(out.Files, ";"); !strings.Contains(joined, "main.dcu") {
		t.Errorf("respectGitignore=false should include main.dcu: %v", out.Files)
	}
}

func TestGrepRespectsGitignore(t *testing.T) {
	dir, h := gitignoreTree(t)

	_, out, err := h.HandleGrep(context.Background(), nil, GrepInput{Pattern: "Alpha", Paths: []string{dir}})
	if err != nil {
		t.Fatal(err)
	}
	if out.FilesMatched != 1 {
		t.Errorf("default FilesMatched = %d, want 1 (main.pas only)", out.FilesMatched)
	}

	_, out, err = h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern: "Alpha", Paths: []string{dir}, RespectGitignore: boolp(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.FilesMatched != 2 {
		t.Errorf("respectGitignore=false FilesMatched = %d, want 2", out.FilesMatched)
	}
}

func TestTreeRespectsGitignore(t *testing.T) {
	dir, h := gitignoreTree(t)

	_, out, err := h.HandleTree(context.Background(), nil, TreeInput{Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.Tree, "main.dcu") || strings.Contains(out.Tree, "__history") {
		t.Errorf("default tree should skip ignored entries:\n%s", out.Tree)
	}

	_, out, err = h.HandleTree(context.Background(), nil, TreeInput{Path: dir, RespectGitignore: boolp(false)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.Tree, "main.dcu") {
		t.Errorf("respectGitignore=false tree should include main.dcu:\n%s", out.Tree)
	}
}
