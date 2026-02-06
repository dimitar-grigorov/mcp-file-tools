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
		{name: "Windows forward slashes", uri: "file:///D:/Projects/app", want: "D:/Projects/app", skip: "linux"},
		{name: "Unix absolute path", uri: "file:///home/user/project", want: "/home/user/project", skip: "windows"},
		{name: "Unix tmp path", uri: "file:///tmp/test", want: "/tmp/test", skip: "windows"},
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

func TestUpdateAllowedDirectoriesFromRoots_ReplacesExisting(t *testing.T) {
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	// Start with one directory
	h := handler.NewHandler([]string{tempDir1})

	// Verify initial state
	initialDirs := h.GetAllowedDirectories()
	if len(initialDirs) != 1 {
		t.Fatalf("expected 1 initial directory, got %d", len(initialDirs))
	}
	initialDir := initialDirs[0]

	// Update with different directory - should replace, not append
	roots := []*mcp.Root{
		{URI: "file:///" + filepath.ToSlash(tempDir2)},
	}

	updateAllowedDirectoriesFromRoots(h, roots)

	// After update, should have exactly 1 directory (replaced, not appended)
	updatedDirs := h.GetAllowedDirectories()
	if len(updatedDirs) != 1 {
		t.Errorf("expected 1 directory after update (replacement), got %d", len(updatedDirs))
	}

	// Verify the directory changed (it's not the same as initial)
	if len(updatedDirs) > 0 {
		updatedDir := updatedDirs[0]

		// The updated directory should NOT be the same as the initial directory
		// (Compare resolved paths to handle symlinks)
		initialResolved, _ := filepath.EvalSymlinks(initialDir)
		updatedResolved, _ := filepath.EvalSymlinks(updatedDir)

		if initialResolved == updatedResolved {
			t.Errorf("directory was not replaced: still have %s", initialResolved)
		}
	}
}
