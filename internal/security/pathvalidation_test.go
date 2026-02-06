package security

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
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
				// Symlink to forbidden file should return ErrSymlinkDenied
				if !errors.Is(err, ErrSymlinkDenied) {
					t.Errorf("%s: expected ErrSymlinkDenied, got: %v", tt.description, err)
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
			if err != nil && !errors.Is(err, ErrPathDenied) {
				t.Errorf("%s: expected ErrPathDenied, got: %v", tt.description, err)
			}
		})
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	// Test ~ expands to home
	if got := ExpandHome("~"); got != home {
		t.Errorf("ExpandHome(~) = %q, want %q", got, home)
	}

	// Test ~/path expands correctly
	want := filepath.Join(home, "Documents")
	if got := ExpandHome("~/Documents"); got != want {
		t.Errorf("ExpandHome(~/Documents) = %q, want %q", got, want)
	}

	// Test non-tilde paths unchanged
	if got := ExpandHome("/usr/bin"); got != "/usr/bin" {
		t.Errorf("ExpandHome(/usr/bin) = %q, want /usr/bin", got)
	}
}

func FuzzIsPathWithinAllowedDirectories(f *testing.F) {
	seeds := []string{
		"/home/user/project/file.txt",
		"/home/user/project/../../../etc/passwd",
		"/home/user/project2/file.txt",
		"/home/user/project\x00/etc/passwd",
		"",
		"./relative",
		"../escape",
		"/home/user/project/./file.txt",
		"/home/user/project//double//slash",
		"/home/user/project/\t",
		"/home/user/project/ ",
		"\"/home/user/project/quoted\"",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	allowedDir := "/home/user/project"
	allowedDirWithSep := allowedDir + "/"

	f.Fuzz(func(t *testing.T, path string) {
		result := IsPathWithinAllowedDirectories(path, []string{allowedDir})

		if !result {
			return
		}

		// Invariant: accepted paths must be within allowed directory
		cleaned := normalizePath(filepath.Clean(path))
		if cleaned != allowedDir && !hasPathPrefix(cleaned, allowedDirWithSep) {
			t.Errorf("path %q was allowed but cleaned path %q is not within %q", path, cleaned, allowedDir)
		}
	})
}

func FuzzNormalizePath(f *testing.F) {
	seeds := []string{
		"C:\\Users\\test",
		"/home/user",
		"\"quoted path\"",
		"  spaces  ",
		"\x00null",
		"",
		"~/home",
		"c:\\mixed/separators\\path",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, path string) {
		_ = normalizePath(path) // must not panic
	})
}

func hasPathPrefix(path, prefix string) bool {
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
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

