// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

//go:build !windows

package security

import "testing"

func TestExpandShortPath_Passthrough(t *testing.T) {
	for _, path := range []string{"", "/tmp/PROGRA~1/file.txt", "relative/path"} {
		if got := expandShortPath(path); got != path {
			t.Errorf("expandShortPath(%q) = %q, want unchanged", path, got)
		}
	}
}
