package security

import "github.com/dimitar-grigorov/mcp-file-tools/internal/errkind"

// Sentinel errors for path validation and security operations.
// Match with errors.Is(); the attached Kind is what callers classify on.

var (
	// ErrNoAllowedDirs is returned when no allowed directories are configured.
	ErrNoAllowedDirs = errkind.New(errkind.AccessDenied, "no allowed directories configured - please provide directories via CLI arguments or MCP roots protocol")

	// ErrPathDenied is returned when a path is outside all allowed directories.
	ErrPathDenied = errkind.New(errkind.AccessDenied, "access denied - path outside allowed directories")

	// ErrSymlinkDenied is returned when a symlink target is outside allowed directories.
	ErrSymlinkDenied = errkind.New(errkind.SymlinkEscape, "access denied - symlink target outside allowed directories")

	// ErrParentDirDenied is returned when a parent directory is outside allowed directories.
	ErrParentDirDenied = errkind.New(errkind.AccessDenied, "access denied - parent directory outside allowed directories")

	// ErrParentNotExists is returned when the parent directory does not exist.
	ErrParentNotExists = errkind.New(errkind.InvalidPath, "parent directory does not exist")

	// ErrNotDirectory is returned when a path is expected to be a directory but is not.
	ErrNotDirectory = errkind.New(errkind.InvalidPath, "path is not a directory")
)
