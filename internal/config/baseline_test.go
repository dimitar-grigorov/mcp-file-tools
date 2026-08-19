// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// chdir moves into dir for the test and restores the old working directory after.
func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestResolveBaseline_CLIArgsWin(t *testing.T) {
	cli := t.TempDir()
	t.Setenv(EnvAllowedDirs, t.TempDir())
	chdir(t, t.TempDir())

	b, err := ResolveBaseline([]string{cli})
	if err != nil {
		t.Fatalf("ResolveBaseline: %v", err)
	}
	if b.Source != SourceCLI || !b.Explicit {
		t.Fatalf("source = %q explicit = %v, want %q true", b.Source, b.Explicit, SourceCLI)
	}
	if len(b.Dirs) != 1 || !strings.EqualFold(b.Dirs[0], mustNormalize(t, cli)) {
		t.Errorf("dirs = %v, want [%s]", b.Dirs, cli)
	}
}

func TestResolveBaseline_EnvBeatsWorkingDir(t *testing.T) {
	env := t.TempDir()
	t.Setenv(EnvAllowedDirs, env)
	chdir(t, t.TempDir())

	b, err := ResolveBaseline(nil)
	if err != nil {
		t.Fatalf("ResolveBaseline: %v", err)
	}
	if b.Source != SourceEnv || !b.Explicit {
		t.Fatalf("source = %q explicit = %v, want %q true", b.Source, b.Explicit, SourceEnv)
	}
	if len(b.Dirs) != 1 || !strings.EqualFold(b.Dirs[0], mustNormalize(t, env)) {
		t.Errorf("dirs = %v, want [%s]", b.Dirs, env)
	}
}

// The zero-config path: no arguments, no env, so the workspace we were started in.
func TestResolveBaseline_FallsBackToWorkingDir(t *testing.T) {
	t.Setenv(EnvAllowedDirs, "")
	cwd := t.TempDir()
	chdir(t, cwd)

	b, err := ResolveBaseline(nil)
	if err != nil {
		t.Fatalf("ResolveBaseline: %v", err)
	}
	if b.Source != SourceCWD || b.Explicit {
		t.Fatalf("source = %q explicit = %v, want %q false", b.Source, b.Explicit, SourceCWD)
	}
	if len(b.Dirs) != 1 || !strings.EqualFold(b.Dirs[0], mustNormalize(t, cwd)) {
		t.Errorf("dirs = %v, want [%s]", b.Dirs, cwd)
	}
}

func TestResolveBaseline_EnvKeepsUsableDirsOnly(t *testing.T) {
	good := t.TempDir()
	notADir := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(notADir, []byte("x"), 0o600); err != nil {
		t.Fatalf("write %s: %v", notADir, err)
	}
	t.Setenv(EnvAllowedDirs, strings.Join([]string{notADir, good}, string(os.PathListSeparator)))
	chdir(t, t.TempDir())

	b, err := ResolveBaseline(nil)
	if err != nil {
		t.Fatalf("ResolveBaseline: %v", err)
	}
	if b.Source != SourceEnv || len(b.Dirs) != 1 || !strings.EqualFold(b.Dirs[0], mustNormalize(t, good)) {
		t.Errorf("source = %q dirs = %v, want %q [%s]", b.Source, b.Dirs, SourceEnv, good)
	}
}

func TestResolveBaseline_OptOutDisablesFallback(t *testing.T) {
	t.Setenv(EnvAllowedDirs, "")
	t.Setenv(EnvNoCWDFallback, "1")
	chdir(t, t.TempDir())

	b, err := ResolveBaseline(nil)
	if err != nil {
		t.Fatalf("ResolveBaseline: %v", err)
	}
	if b.Source != SourceNone || len(b.Dirs) != 0 {
		t.Fatalf("source = %q dirs = %v, want %q none", b.Source, b.Dirs, SourceNone)
	}
	if !strings.Contains(b.Reason, EnvNoCWDFallback) {
		t.Errorf("reason = %q, want it to name %s", b.Reason, EnvNoCWDFallback)
	}
}

// "off" and friends mean the fallback stays on; only a real value opts out.
func TestResolveBaseline_OptOutIgnoresOffValues(t *testing.T) {
	t.Setenv(EnvAllowedDirs, "")
	chdir(t, t.TempDir())

	for _, value := range []string{"0", "false", "no", "off", " "} {
		t.Setenv(EnvNoCWDFallback, value)
		b, err := ResolveBaseline(nil)
		if err != nil {
			t.Fatalf("ResolveBaseline(%q): %v", value, err)
		}
		if b.Source != SourceCWD {
			t.Errorf("%s=%q gave source %q, want %q", EnvNoCWDFallback, value, b.Source, SourceCWD)
		}
	}
}

func TestWorkingDirBaseline_RefusesFilesystemRoot(t *testing.T) {
	root := "/"
	if cwd, err := os.Getwd(); err == nil {
		root = filepath.VolumeName(cwd) + string(filepath.Separator)
	}
	chdir(t, root)

	if _, err := workingDirBaseline(); err == nil {
		t.Fatalf("granted filesystem root %s, want refusal", root)
	} else if !strings.Contains(err.Error(), "filesystem root") {
		t.Errorf("error = %v, want it to name the filesystem root", err)
	}
}

func TestWorkingDirBaseline_RefusesHomeDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory on this platform")
	}
	chdir(t, home)

	if _, err := workingDirBaseline(); err == nil {
		t.Fatalf("granted home directory %s, want refusal", home)
	} else if !strings.Contains(err.Error(), "home directory") {
		t.Errorf("error = %v, want it to name the home directory", err)
	}
}

// The parent of every home (C:\Users, /home) is above home, so the same rule covers it.
func TestWorkingDirBaseline_RefusesAncestorOfHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory on this platform")
	}
	parent := filepath.Dir(home)
	if parent == home || filepath.Dir(parent) == parent {
		t.Skip("home has no parent below the filesystem root")
	}
	chdir(t, parent)

	if _, err := workingDirBaseline(); err == nil {
		t.Fatalf("granted %s, an ancestor of home, want refusal", parent)
	}
}

func mustNormalize(t *testing.T, dir string) string {
	t.Helper()
	b, err := ResolveBaseline([]string{dir})
	if err != nil {
		t.Fatalf("normalize %s: %v", dir, err)
	}
	return b.Dirs[0]
}
