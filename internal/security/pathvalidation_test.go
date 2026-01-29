package security

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestIsPathWithinAllowedDirectories_BasicCases(t *testing.T) {
	// Skip on Windows - these tests use Unix paths
	// The Windows-specific tests cover the same logic with Windows paths
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix path tests on Windows - see TestIsPathWithinAllowedDirectories_WindowsPaths")
	}

	tests := []struct {
		name         string
		path         string
		allowedDirs  []string
		expected     bool
		description  string
	}{
		{
			name:        "exact match",
			path:        "/home/user/project",
			allowedDirs: []string{"/home/user/project"},
			expected:    true,
			description: "exact directory match should be allowed",
		},
		{
			name:        "subdirectory",
			path:        "/home/user/project/src/main.go",
			allowedDirs: []string{"/home/user/project"},
			expected:    true,
			description: "subdirectory should be allowed",
		},
		{
			name:        "prefix attack - project2",
			path:        "/home/user/project2/file.txt",
			allowedDirs: []string{"/home/user/project"},
			expected:    false,
			description: "prefix attack should be blocked",
		},
		{
			name:        "prefix attack - project_backup",
			path:        "/home/user/project_backup/file.txt",
			allowedDirs: []string{"/home/user/project"},
			expected:    false,
			description: "prefix attack with underscore should be blocked",
		},
		{
			name:        "sibling directory",
			path:        "/home/user/other/file.txt",
			allowedDirs: []string{"/home/user/project"},
			expected:    false,
			description: "sibling directory should be blocked",
		},
		{
			name:        "parent directory",
			path:        "/home/user/file.txt",
			allowedDirs: []string{"/home/user/project"},
			expected:    false,
			description: "parent directory should be blocked",
		},
		{
			name:        "multiple allowed dirs - first match",
			path:        "/home/user/project1/file.txt",
			allowedDirs: []string{"/home/user/project1", "/home/user/project2"},
			expected:    true,
			description: "should match first allowed directory",
		},
		{
			name:        "multiple allowed dirs - second match",
			path:        "/home/user/project2/file.txt",
			allowedDirs: []string{"/home/user/project1", "/home/user/project2"},
			expected:    true,
			description: "should match second allowed directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPathWithinAllowedDirectories(tt.path, tt.allowedDirs)
			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expected, result)
			}
		})
	}
}

func TestIsPathWithinAllowedDirectories_SecurityVulnerabilities(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		allowedDirs []string
		expected    bool
		description string
	}{
		{
			name:        "null byte injection",
			path:        "/home/user/project\x00/etc/passwd",
			allowedDirs: []string{"/home/user/project"},
			expected:    false,
			description: "null byte injection should be blocked",
		},
		{
			name:        "empty path",
			path:        "",
			allowedDirs: []string{"/home/user/project"},
			expected:    false,
			description: "empty path should be rejected",
		},
		{
			name:        "empty allowed dirs",
			path:        "/home/user/project/file.txt",
			allowedDirs: []string{},
			expected:    false,
			description: "empty allowed dirs should reject all paths",
		},
		{
			name:        "relative path",
			path:        "./file.txt",
			allowedDirs: []string{"/home/user/project"},
			expected:    false,
			description: "relative paths should be rejected",
		},
		{
			name:        "relative path with parent",
			path:        "../file.txt",
			allowedDirs: []string{"/home/user/project"},
			expected:    false,
			description: "relative paths with parent should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPathWithinAllowedDirectories(tt.path, tt.allowedDirs)
			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expected, result)
			}
		})
	}
}

func TestIsPathWithinAllowedDirectories_WindowsPaths(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping Windows-specific tests")
	}

	tests := []struct {
		name        string
		path        string
		allowedDirs []string
		expected    bool
		description string
	}{
		{
			name:        "Windows drive letter",
			path:        "C:\\Users\\user\\project\\file.txt",
			allowedDirs: []string{"C:\\Users\\user\\project"},
			expected:    true,
			description: "Windows path should be allowed",
		},
		{
			name:        "Windows prefix attack",
			path:        "C:\\Users\\user\\project2\\file.txt",
			allowedDirs: []string{"C:\\Users\\user\\project"},
			expected:    false,
			description: "Windows prefix attack should be blocked",
		},
		{
			name:        "Windows case insensitive drive",
			path:        "c:\\Users\\user\\project\\file.txt",
			allowedDirs: []string{"C:\\Users\\user\\project"},
			expected:    true,
			description: "drive letter case should be normalized",
		},
		{
			name:        "UNC path",
			path:        "\\\\server\\share\\project\\file.txt",
			allowedDirs: []string{"\\\\server\\share\\project"},
			expected:    true,
			description: "UNC paths should be supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPathWithinAllowedDirectories(tt.path, tt.allowedDirs)
			if result != tt.expected {
				t.Errorf("%s: expected %v, got %v", tt.description, tt.expected, result)
			}
		})
	}
}

func TestValidatePath_FileOperations(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	allowedDir := filepath.Join(tempDir, "allowed")
	forbiddenDir := filepath.Join(tempDir, "forbidden")

	if err := os.MkdirAll(allowedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(forbiddenDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test file
	testFile := filepath.Join(allowedDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		path        string
		allowedDirs []string
		shouldError bool
		description string
	}{
		{
			name:        "existing file in allowed dir",
			path:        testFile,
			allowedDirs: []string{allowedDir},
			shouldError: false,
			description: "existing file should be validated",
		},
		{
			name:        "non-existent file in allowed dir",
			path:        filepath.Join(allowedDir, "new.txt"),
			allowedDirs: []string{allowedDir},
			shouldError: false,
			description: "non-existent file in allowed dir should be allowed",
		},
		{
			name:        "file in forbidden dir",
			path:        filepath.Join(forbiddenDir, "test.txt"),
			allowedDirs: []string{allowedDir},
			shouldError: true,
			description: "file in forbidden dir should be rejected",
		},
		{
			name:        "relative path to allowed file",
			path:        "test.txt",
			allowedDirs: []string{allowedDir},
			shouldError: true, // Will fail unless cwd is allowedDir
			description: "relative path should be resolved and validated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidatePath(tt.path, tt.allowedDirs)
			if tt.shouldError && err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("%s: unexpected error: %v", tt.description, err)
			}
		})
	}
}

func TestValidatePath_Symlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping symlink tests on Windows (requires admin privileges)")
	}

	// Create temporary directory structure
	tempDir := t.TempDir()
	allowedDir := filepath.Join(tempDir, "allowed")
	forbiddenDir := filepath.Join(tempDir, "forbidden")

	if err := os.MkdirAll(allowedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(forbiddenDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create files
	allowedFile := filepath.Join(allowedDir, "allowed.txt")
	forbiddenFile := filepath.Join(forbiddenDir, "forbidden.txt")
	if err := os.WriteFile(allowedFile, []byte("allowed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(forbiddenFile, []byte("forbidden"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create symlinks
	goodSymlink := filepath.Join(allowedDir, "good-link.txt")
	badSymlink := filepath.Join(allowedDir, "bad-link.txt")

	if err := os.Symlink(allowedFile, goodSymlink); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(forbiddenFile, badSymlink); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		path        string
		allowedDirs []string
		shouldError bool
		description string
	}{
		{
			name:        "symlink to allowed file",
			path:        goodSymlink,
			allowedDirs: []string{allowedDir},
			shouldError: false,
			description: "symlink to allowed file should be allowed",
		},
		{
			name:        "symlink to forbidden file",
			path:        badSymlink,
			allowedDirs: []string{allowedDir},
			shouldError: true,
			description: "symlink to forbidden file should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidatePath(tt.path, tt.allowedDirs)
			if tt.shouldError && err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("%s: unexpected error: %v", tt.description, err)
			}
			if tt.shouldError && err != nil {
				if !strings.Contains(err.Error(), "outside allowed directories") {
					t.Errorf("%s: expected 'outside allowed directories' error, got: %v", tt.description, err)
				}
			}
		})
	}
}

func TestValidatePath_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()
	allowedDir := filepath.Join(tempDir, "allowed")

	if err := os.MkdirAll(allowedDir, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		path        string
		description string
	}{
		{
			name:        "parent traversal",
			path:        filepath.Join(allowedDir, "..", "..", "etc", "passwd"),
			description: "path traversal with .. should be blocked",
		},
		{
			name:        "multiple parent traversal",
			path:        filepath.Join(allowedDir, "..", "..", "..", "etc", "passwd"),
			description: "multiple .. should be blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidatePath(tt.path, []string{allowedDir})
			if err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
			}
			if err != nil && !strings.Contains(err.Error(), "outside allowed directories") {
				t.Errorf("%s: expected 'outside allowed directories' error, got: %v", tt.description, err)
			}
		})
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home directory")
	}

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "tilde only",
			path:     "~",
			expected: home,
		},
		{
			name:     "tilde with path",
			path:     "~/Documents/file.txt",
			expected: filepath.Join(home, "Documents", "file.txt"),
		},
		{
			name:     "no tilde",
			path:     "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "relative path",
			path:     "relative/path",
			expected: "relative/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandHome(tt.path)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestNormalizeAllowedDirs(t *testing.T) {
	tempDir := t.TempDir()
	existingDir := filepath.Join(tempDir, "existing")
	if err := os.MkdirAll(existingDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file (not a directory)
	notADir := filepath.Join(tempDir, "file.txt")
	if err := os.WriteFile(notADir, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		dirs        []string
		shouldError bool
		description string
	}{
		{
			name:        "existing directory",
			dirs:        []string{existingDir},
			shouldError: false,
			description: "existing directory should be normalized",
		},
		{
			name:        "non-existent directory",
			dirs:        []string{filepath.Join(tempDir, "nonexistent")},
			shouldError: false,
			description: "non-existent directory should be allowed",
		},
		{
			name:        "file instead of directory",
			dirs:        []string{notADir},
			shouldError: true,
			description: "file instead of directory should be rejected",
		},
		{
			name:        "multiple directories",
			dirs:        []string{existingDir, filepath.Join(tempDir, "another")},
			shouldError: false,
			description: "multiple directories should be normalized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NormalizeAllowedDirs(tt.dirs)
			if tt.shouldError && err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
			}
			if !tt.shouldError && err != nil {
				t.Errorf("%s: unexpected error: %v", tt.description, err)
			}
			if !tt.shouldError && err == nil {
				if len(result) != len(tt.dirs) {
					t.Errorf("%s: expected %d directories, got %d", tt.description, len(tt.dirs), len(result))
				}
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		contains string // What the result should contain
	}{
		{
			name:     "clean path",
			path:     "/home/user/project",
			contains: "project",
		},
		{
			name:     "path with quotes",
			path:     "\"/home/user/project\"",
			contains: "project",
		},
		{
			name:     "path with spaces",
			path:     "  /home/user/project  ",
			contains: "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePath(tt.path)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("expected result to contain %s, got %s", tt.contains, result)
			}
		})
	}
}

func TestValidatePath_NoAllowedDirs(t *testing.T) {
	_, err := ValidatePath("/any/path", []string{})
	if err == nil {
		t.Error("expected error when no allowed directories configured")
	}
	if err != nil && !strings.Contains(err.Error(), "no allowed directories") {
		t.Errorf("expected 'no allowed directories' error, got: %v", err)
	}
}
