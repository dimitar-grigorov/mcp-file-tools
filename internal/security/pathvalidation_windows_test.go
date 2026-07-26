// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

//go:build windows

package security

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestValidatePath_RejectsJunctionEscape(t *testing.T) {
	allowedDir := t.TempDir()
	outsideDir := t.TempDir()
	junction := filepath.Join(allowedDir, "escape")
	createJunctionForTest(t, outsideDir, junction)

	if _, err := ValidatePath(junction, []string{allowedDir}); !errors.Is(err, ErrSymlinkDenied) {
		t.Fatalf("ValidatePath(%q) error = %v, want ErrSymlinkDenied", junction, err)
	}

	// Single and multi-level missing suffixes must both be denied.
	for _, path := range []string{
		filepath.Join(junction, "new.txt"),
		filepath.Join(junction, "missing", "nested", "new.txt"),
	} {
		if _, err := ValidatePath(path, []string{allowedDir}); !errors.Is(err, ErrParentDirDenied) {
			t.Errorf("ValidatePath(%q) error = %v, want ErrParentDirDenied", path, err)
		}
	}

	if IsPathSafeResolved(junction, ResolveAllowedDirs([]string{allowedDir})) {
		t.Error("junction escape resolved as safe")
	}
}

func TestValidatePath_AllowsMissingPathThroughSafeJunction(t *testing.T) {
	allowedDir := t.TempDir()
	target := filepath.Join(allowedDir, "target")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	junction := filepath.Join(allowedDir, "safe-link")
	createJunctionForTest(t, target, junction)

	requested := filepath.Join(junction, "missing", "nested", "new.txt")
	validated, err := ValidatePath(requested, []string{allowedDir})
	if err != nil {
		t.Fatal(err)
	}
	if validated != requested {
		t.Fatalf("validated path = %q, want %q", validated, requested)
	}
}

func TestValidatePath_DriveRootAllowedDir(t *testing.T) {
	tempDir := t.TempDir()
	driveRoot := filepath.VolumeName(tempDir) + `\`
	file := filepath.Join(tempDir, "file.txt")
	if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := ValidatePath(file, []string{driveRoot}); err != nil {
		t.Fatalf("ValidatePath(%q) with allowed dir %q: %v", file, driveRoot, err)
	}
}

func TestResolveExistingPath_ResolvesJunctionTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	junction := filepath.Join(root, "junction")
	createJunctionForTest(t, target, junction)

	resolved, err := resolveExistingPath(junction)
	if err != nil {
		t.Fatal(err)
	}
	resolvedInfo, err := os.Stat(resolved)
	if err != nil {
		t.Fatal(err)
	}
	targetInfo, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(resolvedInfo, targetInfo) {
		t.Fatalf("resolved path = %q, want same directory as %q", resolved, target)
	}
}

func TestNormalizeWindowsFinalPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: `\\?\C:\work\file.txt`, want: `C:\work\file.txt`},
		{input: `\\?\UNC\server\share\file.txt`, want: `\\server\share\file.txt`},
		{input: `\??\C:\work\file.txt`, want: `C:\work\file.txt`},
		{input: `\??\UNC\server\share\file.txt`, want: `\\server\share\file.txt`},
		{input: `C:\work\file.txt`, want: `C:\work\file.txt`},
	}
	for _, tt := range tests {
		if got := normalizeWindowsFinalPath(tt.input); got != tt.want {
			t.Errorf("normalizeWindowsFinalPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func createJunctionForTest(t *testing.T, target, link string) {
	t.Helper()
	output, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
	if err != nil {
		t.Skipf("directory junctions are not supported in this environment: %v (%s)", err, output)
	}
}
