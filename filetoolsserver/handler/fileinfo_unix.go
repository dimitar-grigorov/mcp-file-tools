//go:build !windows

package handler

import (
	"os"
	"syscall"
	"time"
)

// getFileTimes returns created, accessed, and modified times for a file on Unix
// Note: Unix doesn't have a true creation time, so we use ctime (status change time) as a fallback
func getFileTimes(stat os.FileInfo) (created, accessed, modified time.Time) {
	modified = stat.ModTime()

	sys := stat.Sys().(*syscall.Stat_t)
	if sys != nil {
		// Use atime for accessed time
		accessed = time.Unix(sys.Atim.Sec, sys.Atim.Nsec)
		// Unix doesn't have birthtime, use ctime (status change time) as fallback
		// Note: ctime is NOT creation time, it's the time the inode was last changed
		created = time.Unix(sys.Ctim.Sec, sys.Ctim.Nsec)
	} else {
		// Fallback if syscall data is not available
		created = modified
		accessed = modified
	}

	return created, accessed, modified
}
