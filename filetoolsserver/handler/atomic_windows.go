// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

//go:build windows

package handler

// syncDir is a no-op: Windows has no portable directory fsync, and NTFS journals
// the rename itself.
func syncDir(string) error { return nil }
