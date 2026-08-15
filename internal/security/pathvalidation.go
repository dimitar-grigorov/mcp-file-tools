// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package security

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ValidatePath resolves a path and ensures it's within allowed directories.
func ValidatePath(requestedPath string, allowedDirs []string) (string, error) {
	if len(allowedDirs) == 0 {
		return "", ErrNoAllowedDirs
	}

	expanded := expandHome(requestedPath)

	var absolute string
	if filepath.IsAbs(expanded) {
		absolute = filepath.Clean(expanded)
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		absolute = filepath.Clean(filepath.Join(cwd, expanded))
	}

	normalized := normalizePath(absolute)

	if !isPathWithinAllowedDirectories(normalized, allowedDirs) {
		// Retry with 8.3 names expanded: the forms may differ only in short/long naming.
		// Expanded once here, never inside the per-allowed-dir loop — it hits the disk.
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
	expanded := expandHome(dir)

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

// isPathWithinAllowedDirectories is the lexical containment test only — it does not
// resolve links, so callers must pair it with a resolved check as ValidatePath does.
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

// resolvesInside is the lexical gate's last resort: allowed dirs are stored resolved,
// so a platform alias spelling (macOS /var for /private/var) only matches once resolved.
// Safe because it can only reach content that is genuinely inside an allowed dir —
// the resolved check below stays authoritative. Errors mean "not inside".
func resolvesInside(absolute string, allowedDirs []string) bool {
	resolved, _, err := resolvePathAllowMissing(absolute)
	if err != nil {
		return false
	}
	return isPathWithinAllowedDirectories(normalizePath(resolved), allowedDirs)
}

// resolvePathAllowMissing resolves the nearest existing ancestor and re-projects the
// missing suffix onto it. Existing but unresolvable paths fail closed.
func resolvePathAllowMissing(path string) (resolved string, exists bool, err error) {
	current := filepath.Clean(path)
	var missing []string

	for {
		resolvedCurrent, resolveErr := resolveExistingPath(current)
		if resolveErr == nil {
			resolvedCurrent = filepath.Clean(resolvedCurrent)
			for i := len(missing) - 1; i >= 0; i-- {
				resolvedCurrent = filepath.Join(resolvedCurrent, missing[i])
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

func normalizePath(p string) string {
	p = strings.Trim(p, "\"' \t\n")
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" && len(p) >= 2 && p[1] == ':' {
		p = strings.ToUpper(p[:1]) + p[1:]
	}

	return p
}

func pathsEqual(first, second string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(first, second)
	}
	return first == second
}

func pathHasPrefix(path, prefix string) bool {
	if runtime.GOOS == "windows" {
		return len(path) >= len(prefix) && strings.EqualFold(path[:len(prefix)], prefix)
	}
	return strings.HasPrefix(path, prefix)
}
