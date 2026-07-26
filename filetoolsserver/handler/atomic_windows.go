//go:build windows

package handler

// syncDir is a no-op: Windows has no portable directory fsync, and NTFS journals
// the rename itself.
func syncDir(string) error { return nil }
