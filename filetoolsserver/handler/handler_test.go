package handler

import (
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
	h := NewHandler([]string{"/tmp"})

	newDirs := []string{"/home", "/var", "/opt"}
	h.UpdateAllowedDirectories(newDirs)

	got := h.GetAllowedDirectories()
	if len(got) != len(newDirs) {
		t.Fatalf("expected %d dirs, got %d", len(newDirs), len(got))
	}

	for i, d := range newDirs {
		if got[i] != d {
			t.Errorf("dir[%d] = %q, want %q", i, got[i], d)
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
