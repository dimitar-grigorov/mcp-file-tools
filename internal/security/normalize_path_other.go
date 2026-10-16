// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

//go:build !windows

package security

import "strings"

// expandShortPath is a no-op: 8.3 short names are Windows-only.
func expandShortPath(path string) string {
	return path
}

func pathsEqual(first, second string) bool {
	return first == second
}

func pathHasPrefix(path, prefix string) bool {
	return strings.HasPrefix(path, prefix)
}
