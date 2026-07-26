//go:build !windows

package handler

import "os"

// syncDir flushes a directory entry so a completed rename survives a crash.
func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
