package handler

import (
	"context"
	"testing"
)

func TestHandleListAllowedDirectories(t *testing.T) {
	dirs := []string{"/tmp", "/home/user", "/var/data"}
	h := NewHandler(dirs)

	input := ListAllowedDirectoriesInput{}

	result, output, err := h.HandleListAllowedDirectories(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	if len(output.Directories) != len(dirs) {
		t.Fatalf("expected %d directories, got %d", len(dirs), len(output.Directories))
	}

	for i, d := range dirs {
		if output.Directories[i] != d {
			t.Errorf("dir[%d] = %q, want %q", i, output.Directories[i], d)
		}
	}
}

func TestHandleListAllowedDirectories_Empty(t *testing.T) {
	h := NewHandler([]string{})

	input := ListAllowedDirectoriesInput{}

	result, output, err := h.HandleListAllowedDirectories(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	if len(output.Directories) != 0 {
		t.Errorf("expected 0 directories, got %d", len(output.Directories))
	}
}
