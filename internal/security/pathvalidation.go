package security

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func IsPathWithinAllowedDirectories(absolutePath string, allowedDirs []string) bool {
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

		if normalized == cleanAllowed {
			return true
		}

		sep := string(filepath.Separator)
		if strings.HasPrefix(normalized, cleanAllowed+sep) {
			return true
		}
	}

	return false
}

// ValidatePath resolves a path and ensures it's within allowed directories.
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
			parentDir := filepath.Dir(absolute)
			realParent, err := filepath.EvalSymlinks(parentDir)
			if err != nil {
				if os.IsNotExist(err) {
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

	normalizedReal := normalizePath(realPath)
	if !IsPathWithinAllowedDirectories(normalizedReal, resolvedAllowedDirs) {
		return "", fmt.Errorf("%w: %s", ErrSymlinkDenied, realPath)
	}

	return realPath, nil
}

func normalizePath(p string) string {
	p = strings.Trim(p, "\"' \t\n")
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" && len(p) >= 2 && p[1] == ':' {
		p = strings.ToUpper(p[:1]) + p[1:]
	}

	return p
}

// ResolveAllowedDirs resolves symlinks in allowed directories once.
func ResolveAllowedDirs(allowedDirs []string) []string {
	resolved := make([]string, 0, len(allowedDirs))
	for _, dir := range allowedDirs {
		resolvedDir, err := filepath.EvalSymlinks(dir)
		if err == nil {
			resolved = append(resolved, normalizePath(resolvedDir))
		} else {
			resolved = append(resolved, normalizePath(dir))
		}
	}
	return resolved
}

// IsPathSafeResolved checks if a path (after resolving symlinks) is within pre-resolved allowed dirs.
func IsPathSafeResolved(path string, resolvedAllowedDirs []string) bool {
	if path == "" || len(resolvedAllowedDirs) == 0 {
		return false
	}

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}

	return IsPathWithinAllowedDirectories(resolved, resolvedAllowedDirs)
}

func ExpandHome(path string) string {
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
