// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/security"
)

func TestGitignoreMatching(t *testing.T) {
	gi := `# build output
*.dcu
__history/
/dcu
obj/**
!keep.dcu
docs/*.tmp
**/generated
`
	stack := ignoreStack{{relDir: "", patterns: parseGitignore([]byte(gi))}}

	tests := []struct {
		rel   string
		isDir bool
		want  bool
	}{
		{"main.dcu", false, true},          // *.dcu anywhere
		{"src/deep/main.dcu", false, true}, // unanchored matches at depth
		{"keep.dcu", false, false},         // negation wins (last match)
		{"__history", true, true},          // dir-only pattern on a dir
		{"__history", false, false},        // dir-only pattern skips files
		{"src/__history", true, true},      // dir-only, any depth
		{"dcu", true, true},                // anchored to root
		{"src/dcu", true, false},           // anchored: not at depth
		{"obj/a/b.o", false, true},         // obj/** children
		{"obj", true, false},               // obj/** does not match obj itself
		{"docs/x.tmp", false, true},        // anchored glob
		{"other/docs/x.tmp", false, false}, // anchored: only under root docs
		{"a/b/generated", true, true},      // leading ** crosses dirs
		{"generated", true, true},          // leading ** matches zero dirs
		{"main.pas", false, false},         // untouched
	}
	for _, tc := range tests {
		if got := stack.Ignored(tc.rel, tc.isDir); got != tc.want {
			t.Errorf("Ignored(%q, dir=%v) = %v, want %v", tc.rel, tc.isDir, got, tc.want)
		}
	}
}

// Deeper .gitignore files take precedence over outer ones.
func TestGitignoreNestedPrecedence(t *testing.T) {
	stack := ignoreStack{
		{relDir: "", patterns: parseGitignore([]byte("*.log\n"))},
		{relDir: "src", patterns: parseGitignore([]byte("!important.log\n"))},
	}
	if !stack.Ignored("top.log", false) {
		t.Error("outer *.log should ignore top.log")
	}
	if !stack.Ignored("src/debug.log", false) {
		t.Error("outer *.log should reach into src")
	}
	if stack.Ignored("src/important.log", false) {
		t.Error("inner negation should re-include src/important.log")
	}
}

func TestWalkRespectsGitignore(t *testing.T) {
	dir := t.TempDir()
	mk := func(rel, content string) {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mk(".gitignore", "__history/\n*.dcu\n")
	mk("main.pas", "x")
	mk("main.dcu", "x")
	mk("__history/main.pas.old", "x")
	mk("src/.gitignore", "gen.pas\n")
	mk("src/gen.pas", "x")
	mk("src/real.pas", "x")
	mk(".git/config", "x")

	allowed := security.ResolveAllowedDirs([]string{dir})

	collect := func(respect bool) []string {
		var got []string
		err := Walk(context.Background(), dir, Options{AllowedDirs: allowed, RespectGitignore: respect},
			func(e Entry) (Action, error) {
				if !e.IsDir() {
					got = append(got, e.RelPath)
				}
				return Continue, nil
			})
		if err != nil {
			t.Fatal(err)
		}
		sort.Strings(got)
		return got
	}

	got := collect(true)
	want := []string{".gitignore", "main.pas", "src/.gitignore", "src/real.pas"}
	if len(got) != len(want) {
		t.Fatalf("respect=true files = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("respect=true files = %v, want %v", got, want)
		}
	}

	// Off: everything is visible, including .git and ignored files
	all := collect(false)
	if len(all) != 8 {
		t.Fatalf("respect=false files = %v, want 8 entries", all)
	}
}
