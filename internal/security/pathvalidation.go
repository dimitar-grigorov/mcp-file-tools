package security

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// IsPathWithinAllowedDirectories checks if the given absolute path is within any of the allowed directories.
func IsPathWithinAllowedDirectories(absolutePath string, allowedDirs []string) bool {
	if absolutePath == "" || len(allowedDirs) == 0 {
		return false
	}

	// Prevent null byte injection
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

		if normalized == cleanAllowed {
			return true
		}

		// Prevent prefix attacks: /project vs /project2
		sep := string(filepath.Separator)
		if strings.HasPrefix(normalized, cleanAllowed+sep) {
			return true
		}
	}

	return false
}

// ValidatePath validates and resolves a path, ensuring it's within allowed directories.
// Returns the validated absolute path or an error if access is denied.
func ValidatePath(requestedPath string, allowedDirs []string) (string, error) {
	if len(allowedDirs) == 0 {
		return "", ErrNoAllowedDirs
	}

	expanded := ExpandHome(requestedPath)

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

	if !IsPathWithinAllowedDirectories(normalized, allowedDirs) {
		return "", fmt.Errorf("%w: %s", ErrPathDenied, absolute)
	}

	// Resolve allowed directories for symlink comparison
	resolvedAllowedDirs := make([]string, 0, len(allowedDirs))
	for _, dir := range allowedDirs {
		resolvedDir, err := filepath.EvalSymlinks(dir)
		if err == nil {
			resolvedAllowedDirs = append(resolvedAllowedDirs, normalizePath(resolvedDir))
		} else {
			resolvedAllowedDirs = append(resolvedAllowedDirs, normalizePath(dir))
		}
	}

	realPath, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - validate parent directory
			parentDir := filepath.Dir(absolute)
			realParent, err := filepath.EvalSymlinks(parentDir)
			if err != nil {
				if os.IsNotExist(err) {
					// Parent also doesn't exist - check if path would be valid
					if IsPathWithinAllowedDirectories(normalized, resolvedAllowedDirs) {
						return absolute, nil
					}
					return "", fmt.Errorf("%w: %s", ErrParentNotExists, parentDir)
				}
				return "", fmt.Errorf("failed to resolve parent directory: %w", err)
			}
			normalizedParent := normalizePath(realParent)
			if !IsPathWithinAllowedDirectories(normalizedParent, resolvedAllowedDirs) {
				return "", fmt.Errorf("%w: %s", ErrParentDirDenied, realParent)
			}
			return absolute, nil
		}
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Validate symlink target
	normalizedReal := normalizePath(realPath)
	if !IsPathWithinAllowedDirectories(normalizedReal, resolvedAllowedDirs) {
		return "", fmt.Errorf("%w: %s", ErrSymlinkDenied, realPath)
	}

	return realPath, nil
}

// normalizePath normalizes a path for cross-platform comparison.
func normalizePath(p string) string {
	// Remove quotes and trim
	p = strings.Trim(p, "\"' \t\n")

	// Normalize separators
	p = filepath.Clean(p)

	// On Windows: uppercase drive letter for consistent comparison
	if runtime.GOOS == "windows" && len(p) >= 2 && p[1] == ':' {
		p = strings.ToUpper(p[:1]) + p[1:]
	}

	return p
}

// IsPathSafe checks if a path (resolving symlinks) is within allowed directories.
func IsPathSafe(path string, allowedDirs []string) bool {
	if path == "" || len(allowedDirs) == 0 {
		return false
	}

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}

	resolvedAllowedDirs := make([]string, 0, len(allowedDirs))
	for _, dir := range allowedDirs {
		resolvedDir, err := filepath.EvalSymlinks(dir)
		if err == nil {
			resolvedAllowedDirs = append(resolvedAllowedDirs, normalizePath(resolvedDir))
		} else {
			resolvedAllowedDirs = append(resolvedAllowedDirs, normalizePath(dir))
		}
	}

	return IsPathWithinAllowedDirectories(resolved, resolvedAllowedDirs)
}

// ExpandHome expands the ~ prefix to the user's home directory.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path // Return unchanged if home dir cannot be determined
		}
		if path == "~" {
			return home
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// NormalizeAllowedDirs normalizes and validates a list of allowed directories.
func NormalizeAllowedDirs(dirs []string) ([]string, error) {
	var normalized []string
	for _, dir := range dirs {
		expanded := ExpandHome(dir)

		absolute, err := filepath.Abs(expanded)
		if err != nil {
			return nil, fmt.Errorf("invalid directory %s: %w", dir, err)
		}

		resolved, err := filepath.EvalSymlinks(absolute)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("cannot resolve directory %s: %w", dir, err)
		}
		if os.IsNotExist(err) {
			resolved = absolute
		} else {
			info, err := os.Stat(resolved)
			if err != nil {
				return nil, fmt.Errorf("cannot stat directory %s: %w", resolved, err)
			}
			if !info.IsDir() {
				return nil, fmt.Errorf("%w: %s", ErrNotDirectory, resolved)
			}
		}

		normalized = append(normalized, normalizePath(filepath.Clean(resolved)))
	}
	return normalized, nil
}
