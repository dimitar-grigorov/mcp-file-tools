// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

//go:build !windows

package security

// expandShortPath is a no-op: 8.3 short names are Windows-only.
func expandShortPath(path string) string {
	return path
}
