// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

//go:build windows

package security

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// shortNameDirForTest makes a directory with a long name inside root and returns its
// 8.3 form plus the canonical long form, skipping when 8dot3name is off on this volume.
func shortNameDirForTest(t *testing.T, root string) (short, long string) {
	t.Helper()

	long = filepath.Join(root, "Long Directory Name")
	if err := os.Mkdir(long, 0755); err != nil {
		t.Fatal(err)
	}
	long, err := resolveExistingPath(long)
	if err != nil {
		t.Fatal(err)
	}

	pathPtr, err := windows.UTF16PtrFromString(long)
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]uint16, maxLongPathUTF16Units)
	length, err := windows.GetShortPathName(pathPtr, &buffer[0], uint32(len(buffer)))
	if err != nil {
		t.Skipf("cannot derive an 8.3 short name (8dot3name likely disabled): %v", err)
	}
	short = filepath.Clean(windows.UTF16ToString(buffer[:length]))
	if pathsEqual(short, long) {
		t.Skipf("8dot3name is disabled on this volume: %q has no distinct short name", long)
	}
	return short, long
}

func TestExpandShortPath_MatchesResolvedLongForm(t *testing.T) {
	short, long := shortNameDirForTest(t, t.TempDir())

	if got := expandShortPath(short); !pathsEqual(got, long) {
		t.Fatalf("expandShortPath(%q) = %q, want %q", short, got, long)
	}
}

func TestExpandShortPath_KeepsMissingSuffix(t *testing.T) {
	short, long := shortNameDirForTest(t, t.TempDir())

	got := expandShortPath(filepath.Join(short, "does", "not", "exist"))
	want := filepath.Join(long, "does", "not", "exist")
	if !pathsEqual(got, want) {
		t.Fatalf("expandShortPath = %q, want %q", got, want)
	}
}

func TestValidatePath_AllowsShortNameRequest(t *testing.T) {
	short, long := shortNameDirForTest(t, t.TempDir())
	file := filepath.Join(long, "file.txt")
	if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	allowedDirs, err := NormalizeAllowedDirs([]string{long})
	if err != nil {
		t.Fatal(err)
	}

	requested := filepath.Join(short, "file.txt")
	validated, err := ValidatePath(requested, allowedDirs)
	if err != nil {
		t.Fatalf("ValidatePath(%q) with allowed dir %q: %v", requested, long, err)
	}
	if !pathsEqual(validated, file) {
		t.Fatalf("validated path = %q, want %q", validated, file)
	}
}

func TestValidatePath_AllowsShortNameAllowedDir(t *testing.T) {
	short, long := shortNameDirForTest(t, t.TempDir())

	// A configured-but-missing allowed dir keeps its 8.3 form until expanded.
	allowedDirs, err := NormalizeAllowedDirs([]string{filepath.Join(short, "later")})
	if err != nil {
		t.Fatal(err)
	}

	requested := filepath.Join(long, "later", "file.txt")
	if _, err := ValidatePath(requested, allowedDirs); err != nil {
		t.Fatalf("ValidatePath(%q) with allowed dirs %q: %v", requested, allowedDirs, err)
	}
}

func TestValidatePath_DeniesOutsideShortNamePath(t *testing.T) {
	allowedDir := t.TempDir()
	shortOutside, longOutside := shortNameDirForTest(t, t.TempDir())
	file := filepath.Join(longOutside, "file.txt")
	if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	allowedDirs, err := NormalizeAllowedDirs([]string{allowedDir})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{
		filepath.Join(shortOutside, "file.txt"),
		filepath.Join(shortOutside, "missing.txt"),
	} {
		if _, err := ValidatePath(path, allowedDirs); !errors.Is(err, ErrPathDenied) {
			t.Errorf("ValidatePath(%q) error = %v, want ErrPathDenied", path, err)
		}
	}
}
