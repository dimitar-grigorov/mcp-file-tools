// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/config"
)

func TestNewHandler(t *testing.T) {
	dirs := []string{"/tmp", "/home"}
	h := NewHandler(dirs)

	if h == nil {
		t.Fatal("expected handler, got nil")
	}

	got := h.GetAllowedDirectories()
	if len(got) != len(dirs) {
		t.Errorf("expected %d dirs, got %d", len(dirs), len(got))
	}
}

func TestWithConfig(t *testing.T) {
	cfg := &config.Config{
		DefaultEncoding: "utf-8",
	}

	h := NewHandler([]string{"/tmp"}, WithConfig(cfg))

	if h.config != cfg {
		t.Error("expected config to be set via WithConfig option")
	}
}

func TestWithConfig_Nil(t *testing.T) {
	h := NewHandler([]string{"/tmp"}, WithConfig(nil))

	if h.config == nil {
		t.Error("config should not be nil when WithConfig(nil) is passed")
	}
}

func TestGetAllowedDirectories_ReturnsCopy(t *testing.T) {
	dirs := []string{"/tmp", "/home"}
	h := NewHandler(dirs)

	got := h.GetAllowedDirectories()
	got[0] = "/modified"

	// Original should be unchanged
	original := h.GetAllowedDirectories()
	if original[0] == "/modified" {
		t.Error("GetAllowedDirectories should return a copy, not the original slice")
	}
}

func TestUpdateAllowedDirectories(t *testing.T) {
	h := NewHandler([]string{t.TempDir()})

	newDirs := []string{t.TempDir(), t.TempDir(), t.TempDir()}
	h.UpdateAllowedDirectories(newDirs)
	want := normalizeAllowedDirs(newDirs)

	got := h.GetAllowedDirectories()
	if len(got) != len(want) {
		t.Fatalf("expected %d dirs, got %d", len(want), len(got))
	}

	for i, d := range want {
		if got[i] != d {
			t.Errorf("dir[%d] = %q, want canonical %q", i, got[i], d)
		}
	}
}

// A short (8.3) root must accept requests spelled with the long path, and vice versa.
func TestValidatePath_ShortAndLongAllowedDirSpelling(t *testing.T) {
	root := t.TempDir()
	canonical := normalizeAllowedDirs([]string{root})[0]
	file := filepath.Join(canonical, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	for _, configured := range []string{root, canonical} {
		h := NewHandler([]string{configured})
		if _, err := h.validatePath(file); err != nil {
			t.Errorf("root %q: validatePath(%q) = %v, want success", configured, file, err)
		}
	}
}

func TestUpdateAllowedDirectories_Empty(t *testing.T) {
	h := NewHandler([]string{"/tmp", "/home"})

	h.UpdateAllowedDirectories([]string{})

	got := h.GetAllowedDirectories()
	if len(got) != 0 {
		t.Errorf("expected 0 dirs, got %d", len(got))
	}
}
