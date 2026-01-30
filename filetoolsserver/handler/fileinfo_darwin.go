//go:build darwin

package handler

import (
	"os"
	"syscall"
	"time"
)

// getFileTimes returns created, accessed, and modified times for a file on macOS
func getFileTimes(stat os.FileInfo) (created, accessed, modified time.Time) {
	modified = stat.ModTime()

	sys := stat.Sys().(*syscall.Stat_t)
	if sys != nil {
		accessed = time.Unix(sys.Atimespec.Sec, sys.Atimespec.Nsec)
		// macOS has birthtime but it's not in the standard Stat_t, use ctime as fallback
		created = time.Unix(sys.Ctimespec.Sec, sys.Ctimespec.Nsec)
	} else {
		created = modified
		accessed = modified
	}

	return created, accessed, modified
}
