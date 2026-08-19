// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package security

import "github.com/dimitar-grigorov/mcp-file-tools/v4/internal/errkind"

// Sentinel errors for path validation and security operations.
// Match with errors.Is(); the attached Kind is what callers classify on.

var (
	// ErrNoAllowedDirs is returned when no allowed directories are configured.
	ErrNoAllowedDirs = errkind.New(errkind.AccessDenied, "no allowed directories configured - pass them as CLI arguments or set MCP_FILE_TOOLS_ALLOWED_DIRS")

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
