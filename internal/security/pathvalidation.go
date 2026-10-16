// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package security

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// ValidatePath resolves a path and ensures it's within allowed directories.
func ValidatePath(requestedPath string, allowedDirs []string) (string, error) {
	if len(allowedDirs) == 0 {
		return "", ErrNoAllowedDirs
	}

	absolute, err := absolutePath(requestedPath)
	if err != nil {
		return "", err
	}

	normalized := normalizePath(absolute)

	if !isPathWithinAllowedDirectories(normalized, allowedDirs) {
		// Retry with 8.3 names expanded, once here rather than per allowed dir, since it hits the disk.
		if expanded := expandShortPath(absolute); expanded != absolute {
			absolute = expanded
			normalized = normalizePath(absolute)
		}
		if !isPathWithinAllowedDirectories(normalized, allowedDirs) && !resolvesInside(absolute, allowedDirs) {
			return "", fmt.Errorf("%w: %s", ErrPathDenied, absolute)
		}
	}

	resolvedAllowedDirs := ResolveAllowedDirs(allowedDirs)

	resolvedPath, exists, err := resolvePathAllowMissing(absolute)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrParentNotExists, filepath.Dir(absolute))
		}
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	if !isPathWithinAllowedDirectories(normalizePath(resolvedPath), resolvedAllowedDirs) {
		if exists {
			return "", fmt.Errorf("%w: %s", ErrSymlinkDenied, resolvedPath)
		}
		return "", fmt.Errorf("%w: %s", ErrParentDirDenied, filepath.Dir(resolvedPath))
	}

	if !exists {
		return absolute, nil
	}
	return resolvedPath, nil
}

// ValidateEntryPath is ValidatePath for deleting or renaming the entry itself, so a symlink is not followed to its target.
func ValidateEntryPath(requestedPath string, allowedDirs []string) (string, error) {
	resolved, err := ValidatePath(requestedPath, allowedDirs)
	if err != nil {
		return "", err
	}
	absolute, err := absolutePath(requestedPath)
	if err != nil {
		return "", err
	}
	if filepath.Dir(absolute) == absolute {
		return resolved, nil
	}
	parent, err := ValidatePath(filepath.Dir(absolute), allowedDirs)
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(absolute)), nil
}

func absolutePath(requestedPath string) (string, error) {
	expanded := expandHome(unquote(requestedPath))
	if filepath.IsAbs(expanded) {
		return filepath.Clean(expanded), nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	return filepath.Clean(filepath.Join(cwd, expanded)), nil
}

// IsPathSafeResolved checks if a path (after resolving links) is within pre-resolved allowed dirs.
func IsPathSafeResolved(path string, resolvedAllowedDirs []string) bool {
	if path == "" || len(resolvedAllowedDirs) == 0 {
		return false
	}

	resolved, err := resolveExistingPath(path)
	if err != nil {
		return false
	}

	return isPathWithinAllowedDirectories(filepath.Clean(resolved), resolvedAllowedDirs)
}

// ResolveAllowedDirs resolves links in allowed directories once; unresolvable ones are dropped.
func ResolveAllowedDirs(allowedDirs []string) []string {
	resolved := make([]string, 0, len(allowedDirs))
	for _, dir := range allowedDirs {
		resolvedDir, _, err := resolvePathAllowMissing(dir)
		if err != nil {
			continue
		}
		resolved = append(resolved, normalizePath(resolvedDir))
	}
	return resolved
}

func NormalizeAllowedDirs(dirs []string) ([]string, error) {
	var normalized []string
	for _, dir := range dirs {
		one, err := NormalizeAllowedDir(dir)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, one)
	}
	return normalized, nil
}

// NormalizeAllowedDir canonicalizes one dir — home, absolute, symlinks, 8.3 — so checks compare like with like.
func NormalizeAllowedDir(dir string) (string, error) {
	// The user's own config, so any stray quote goes: Windows argv turns "C:\dir\" into C:\dir".
	expanded := expandHome(strings.Trim(dir, "\"' \t\n"))

	absolute, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("invalid directory %s: %w", dir, err)
	}

	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("cannot resolve directory %s: %w", dir, err)
	}
	if os.IsNotExist(err) {
		resolved = absolute
	} else {
		info, err := os.Stat(resolved)
		if err != nil {
			return "", fmt.Errorf("cannot stat directory %s: %w", resolved, err)
		}
		if !info.IsDir() {
			return "", fmt.Errorf("%w: %s", ErrNotDirectory, resolved)
		}
	}

	return normalizePath(expandShortPath(filepath.Clean(resolved))), nil
}

// isPathWithinAllowedDirectories is lexical only; pair it with a resolved check as ValidatePath does.
func isPathWithinAllowedDirectories(absolutePath string, allowedDirs []string) bool {
	if absolutePath == "" || len(allowedDirs) == 0 {
		return false
	}

	if strings.Contains(absolutePath, "\x00") {
		return false
	}

	normalized := filepath.Clean(absolutePath)
	if !filepath.IsAbs(normalized) {
		return false
	}

	normalized = normalizePath(normalized)

	for _, allowedDir := range allowedDirs {
		cleanAllowed := normalizePath(filepath.Clean(allowedDir))

		if pathsEqual(normalized, cleanAllowed) {
			return true
		}

		// A root ("C:\", "/") already ends in a separator.
		sep := string(filepath.Separator)
		allowedPrefix := cleanAllowed
		if !strings.HasSuffix(allowedPrefix, sep) {
			allowedPrefix += sep
		}
		if pathHasPrefix(normalized, allowedPrefix) {
			return true
		}
	}

	return false
}

// resolvesInside matches an alias spelling (macOS /var for /private/var) once resolved; errors mean not inside.
// Safe: it only reaches what is really inside an allowed dir, and the resolved check stays authoritative.
func resolvesInside(absolute string, allowedDirs []string) bool {
	resolved, _, err := resolvePathAllowMissing(absolute)
	if err != nil {
		return false
	}
	return isPathWithinAllowedDirectories(normalizePath(resolved), allowedDirs)
}

// resolvePathAllowMissing resolves the nearest existing ancestor and re-attaches the rest; an unresolvable existing path fails closed.
func resolvePathAllowMissing(path string) (resolved string, exists bool, err error) {
	current := filepath.Clean(path)
	var missing []string

	for {
		resolvedCurrent, resolveErr := resolveExistingPath(current)
		if resolveErr == nil {
			resolvedCurrent = filepath.Clean(resolvedCurrent)
			for _, m := range slices.Backward(missing) {
				resolvedCurrent = filepath.Join(resolvedCurrent, m)
			}
			return filepath.Clean(resolvedCurrent), len(missing) == 0, nil
		}
		if !os.IsNotExist(resolveErr) {
			return "", false, resolveErr
		}

		if _, lstatErr := os.Lstat(current); lstatErr == nil {
			return "", false, fmt.Errorf("existing path cannot be resolved: %s: %w", current, resolveErr)
		} else if !os.IsNotExist(lstatErr) {
			return "", false, lstatErr
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", false, resolveErr
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		if path == "~" {
			return home
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// unquote strips a model's wrapping quotes on input only; trimming resolved paths let "work'" pass as "work".
func unquote(p string) string {
	p = strings.TrimSpace(p)
	if len(p) >= 2 && (p[0] == '"' || p[0] == '\'') && p[len(p)-1] == p[0] {
		p = strings.TrimSpace(p[1 : len(p)-1])
	}
	return p
}

func normalizePath(p string) string {
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" && len(p) >= 2 && p[1] == ':' {
		p = strings.ToUpper(p[:1]) + p[1:]
	}

	return p
}
