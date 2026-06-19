package filetoolsserver

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/dimitar-grigorov/mcp-file-tools/filetoolsserver/handler"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestUpdateAllowedDirectoriesFromRoots_ValidRoots(t *testing.T) {
	// Create temp directories for testing
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	h := handler.NewHandler([]string{})

	// Create mock roots with file:// URIs (standard format for both platforms)
	roots := []*mcp.Root{
		{URI: "file:///" + filepath.ToSlash(tempDir1)},
		{URI: "file:///" + filepath.ToSlash(tempDir2)},
	}

	updateAllowedDirectoriesFromRoots(h, roots)

	dirs := h.GetAllowedDirectories()
	if len(dirs) != 2 {
		t.Errorf("expected 2 directories, got %d", len(dirs))
	}
}

func TestUpdateAllowedDirectoriesFromRoots_EmptyRoots(t *testing.T) {
	h := handler.NewHandler([]string{})

	updateAllowedDirectoriesFromRoots(h, []*mcp.Root{})

	dirs := h.GetAllowedDirectories()
	if len(dirs) != 0 {
		t.Errorf("expected 0 directories, got %d", len(dirs))
	}
}

func TestUpdateAllowedDirectoriesFromRoots_WindowsPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows platform")
	}

	tempDir := t.TempDir()

	h := handler.NewHandler([]string{})

	// Windows format with drive letter
	roots := []*mcp.Root{
		{URI: "file:///" + filepath.ToSlash(tempDir)},
	}

	updateAllowedDirectoriesFromRoots(h, roots)

	dirs := h.GetAllowedDirectories()
	if len(dirs) != 1 {
		t.Errorf("expected 1 directory, got %d", len(dirs))
	}

	// Check that path is properly formatted for Windows
	if len(dirs) > 0 && len(dirs[0]) > 1 && dirs[0][1] != ':' {
		t.Errorf("expected Windows path with drive letter, got %s", dirs[0])
	}
}

func TestUpdateAllowedDirectoriesFromRoots_UnixPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test on Windows")
	}

	tempDir := t.TempDir()

	h := handler.NewHandler([]string{})

	// Standard file URI: file:///home/user (three slashes)
	roots := []*mcp.Root{
		{URI: "file://" + tempDir}, // file:// + /tmp/... = file:///tmp/...
	}

	updateAllowedDirectoriesFromRoots(h, roots)

	dirs := h.GetAllowedDirectories()
	if len(dirs) != 1 {
		t.Errorf("expected 1 directory, got %d", len(dirs))
	}

	// Check that path starts with /
	if len(dirs) > 0 && dirs[0][0] != '/' {
		t.Errorf("expected Unix path starting with /, got %s", dirs[0])
	}
}

func TestFileURIToPath(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string
		skip string // GOOS to skip on
	}{
		{name: "Windows drive letter", uri: "file:///C:/Users/test", want: "C:/Users/test", skip: "linux"},
		{name: "Unix absolute path", uri: "file:///home/user/project", want: "/home/user/project", skip: "windows"},
		{name: "not a file URI", uri: "/some/path", want: "/some/path"},
		{name: "empty string", uri: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip == runtime.GOOS {
				t.Skipf("skipping on %s", runtime.GOOS)
			}
			got := fileURIToPath(tt.uri)
			if got != tt.want {
				t.Errorf("fileURIToPath(%q) = %q, want %q", tt.uri, got, tt.want)
			}
		})
	}
}

func TestUpdateAllowedDirectoriesFromRoots_MergesWithCLIBaseline(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// Start with one CLI directory (baseline).
	h := handler.NewHandler([]string{tempDir1})

	initialDirs := h.GetAllowedDirectories()
	if len(initialDirs) != 1 {
		t.Fatalf("expected 1 initial directory, got %d", len(initialDirs))
	}

	// Update with a different root - should augment the CLI baseline, not replace it.
	roots := []*mcp.Root{
		{URI: "file:///" + filepath.ToSlash(tempDir2)},
	}

	updateAllowedDirectoriesFromRoots(h, roots)

	// After update, both the CLI baseline and the root should be present.
	updatedDirs := h.GetAllowedDirectories()
	if len(updatedDirs) != 2 {
		t.Fatalf("expected 2 directories after merge, got %d: %v", len(updatedDirs), updatedDirs)
	}

	want := map[string]bool{}
	for _, d := range []string{tempDir1, tempDir2} {
		resolved, _ := filepath.EvalSymlinks(d)
		want[resolved] = true
	}
	for _, d := range updatedDirs {
		resolved, _ := filepath.EvalSymlinks(d)
		delete(want, resolved)
	}
	if len(want) != 0 {
		t.Errorf("merged dirs missing expected entries: %v (got %v)", want, updatedDirs)
	}
}

func TestUpdateAllowedDirectoriesFromRoots_DedupsBaseline(t *testing.T) {
	tempDir1 := t.TempDir()

	h := handler.NewHandler([]string{tempDir1})

	// Root identical to the CLI baseline must not produce a duplicate.
	roots := []*mcp.Root{
		{URI: "file:///" + filepath.ToSlash(tempDir1)},
	}

	updateAllowedDirectoriesFromRoots(h, roots)

	updatedDirs := h.GetAllowedDirectories()
	if len(updatedDirs) != 1 {
		t.Errorf("expected 1 directory after dedup, got %d: %v", len(updatedDirs), updatedDirs)
	}
}
