package security

import "errors"

// Sentinel errors for path validation and security operations.
// Use errors.Is() to check for specific error types.

var (
	// ErrNoAllowedDirs is returned when no allowed directories are configured.
	ErrNoAllowedDirs = errors.New("no allowed directories configured - please provide directories via CLI arguments or MCP roots protocol")

	// ErrPathDenied is returned when a path is outside all allowed directories.
	ErrPathDenied = errors.New("access denied - path outside allowed directories")

	// ErrSymlinkDenied is returned when a symlink target is outside allowed directories.
	ErrSymlinkDenied = errors.New("access denied - symlink target outside allowed directories")

	// ErrParentDirDenied is returned when a parent directory is outside allowed directories.
	ErrParentDirDenied = errors.New("access denied - parent directory outside allowed directories")

	// ErrParentNotExists is returned when the parent directory does not exist.
	ErrParentNotExists = errors.New("parent directory does not exist")

	// ErrNotDirectory is returned when a path is expected to be a directory but is not.
	ErrNotDirectory = errors.New("path is not a directory")
)
