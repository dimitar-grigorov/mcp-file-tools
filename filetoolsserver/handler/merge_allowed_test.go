// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"testing"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/security"
)

// normDir normalizes a directory the way the roots handler does before merging.
func normDir(t *testing.T, dir string) string {
	t.Helper()
	got, err := security.NormalizeAllowedDirs([]string{dir})
	if err != nil || len(got) == 0 {
		t.Fatalf("normalize %s: %v", dir, err)
	}
	return got[0]
}

func TestMergeAllowedDirectories_PreservesBaselineFirst(t *testing.T) {
	base := t.TempDir()
	root := t.TempDir()

	h := NewHandler([]string{base})
	merged := h.MergeAllowedDirectories([]string{normDir(t, root)})

	if len(merged) != 2 {
		t.Fatalf("expected 2 dirs, got %d: %v", len(merged), merged)
	}
	if merged[0] != normDir(t, base) {
		t.Errorf("expected baseline %q first, got %q", normDir(t, base), merged[0])
	}
}

// A client switching its roots must not leave stale roots behind, and the CLI
// baseline must survive every merge.
func TestMergeAllowedDirectories_DropsStaleRoots(t *testing.T) {
	base := t.TempDir()
	rootA := t.TempDir()
	rootB := t.TempDir()

	h := NewHandler([]string{base})
	h.MergeAllowedDirectories([]string{normDir(t, rootA)})
	merged := h.MergeAllowedDirectories([]string{normDir(t, rootB)})

	want := map[string]bool{normDir(t, base): true, normDir(t, rootB): true}
	if len(merged) != len(want) {
		t.Fatalf("expected %d dirs, got %d: %v", len(want), len(merged), merged)
	}
	for _, d := range merged {
		if !want[d] {
			t.Errorf("unexpected dir %q (stale root not dropped?)", d)
		}
	}
}

func TestMergeAllowedDirectories_EmptyKeepsBaseline(t *testing.T) {
	base := t.TempDir()
	h := NewHandler([]string{base})

	merged := h.MergeAllowedDirectories(nil)
	if len(merged) != 1 || merged[0] != normDir(t, base) {
		t.Errorf("expected only baseline %q, got %v", normDir(t, base), merged)
	}
}

func TestMergeAllowedDirectories_ReturnsDefensiveCopy(t *testing.T) {
	base := t.TempDir()
	h := NewHandler([]string{base})

	merged := h.MergeAllowedDirectories(nil)
	merged[0] = "/mutated"

	if got := h.GetAllowedDirectories(); got[0] == "/mutated" {
		t.Error("mutating the returned slice affected handler state; expected a defensive copy")
	}
}

// A derived baseline must survive revoked roots too, or an empty list leaves no access.
func TestMergeAllowedDirectories_EmptyRootsKeepDerivedBaseline(t *testing.T) {
	base := t.TempDir()
	h := NewHandler([]string{base})

	h.MergeAllowedDirectories([]string{normDir(t, t.TempDir())})
	merged := h.MergeAllowedDirectories(nil)

	if len(merged) != 1 || merged[0] != normDir(t, base) {
		t.Errorf("expected only baseline %q, got %v", normDir(t, base), merged)
	}
}

func TestHasExplicitDirs(t *testing.T) {
	base := t.TempDir()

	if h := NewHandler([]string{base}); h.HasExplicitDirs() {
		t.Error("a baseline without WithExplicitDirs should not count as user-configured")
	}
	if h := NewHandler([]string{base}, WithExplicitDirs(true)); !h.HasExplicitDirs() {
		t.Error("WithExplicitDirs(true) should mark the baseline user-configured")
	}
	if h := NewHandler(nil, WithExplicitDirs(true)); h.HasExplicitDirs() {
		t.Error("an empty baseline is never user-configured")
	}
}
