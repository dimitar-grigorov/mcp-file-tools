// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

//go:build linux

package handler

import (
	"os"
	"syscall"
	"time"
)

// getFileTimes returns created, accessed, and modified times for a file on Linux.
func getFileTimes(stat os.FileInfo) (created, accessed, modified time.Time) {
	modified = stat.ModTime()

	sys := stat.Sys().(*syscall.Stat_t)
	if sys != nil {
		accessed = time.Unix(sys.Atim.Sec, sys.Atim.Nsec)
		// Linux doesn't have birthtime, use ctime (status change time) as fallback
		created = time.Unix(sys.Ctim.Sec, sys.Ctim.Nsec)
	} else {
		created = modified
		accessed = modified
	}

	return created, accessed, modified
}
