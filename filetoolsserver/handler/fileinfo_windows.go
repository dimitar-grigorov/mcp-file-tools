//go:build windows

package handler

import (
	"os"
	"syscall"
	"time"
)

// getFileTimes returns created, accessed, and modified times for a file on Windows
func getFileTimes(stat os.FileInfo) (created, accessed, modified time.Time) {
	modified = stat.ModTime()

	sys := stat.Sys().(*syscall.Win32FileAttributeData)
	if sys != nil {
		created = time.Unix(0, sys.CreationTime.Nanoseconds())
		accessed = time.Unix(0, sys.LastAccessTime.Nanoseconds())
	} else {
		// Fallback if syscall data is not available
		created = modified
		accessed = modified
	}

	return created, accessed, modified
}
