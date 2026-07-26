// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

//go:build windows

package security

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

const maxLongPathUTF16Units = 1 << 16

// expandShortPath expands 8.3 components ("PROGRA~1") to long form so the lexical
// containment gate compares like with like. Falls back to the input on any failure.
func expandShortPath(path string) string {
	current := filepath.Clean(path)
	var missing []string

	for {
		expanded, err := longPathName(current)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				expanded = filepath.Join(expanded, missing[i])
			}
			return filepath.Clean(expanded)
		}
		if !os.IsNotExist(err) {
			return path
		}

		// GetLongPathName only works on existing paths: expand the nearest
		// existing ancestor and re-project the missing suffix onto it.
		parent := filepath.Dir(current)
		if parent == current {
			return path
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func longPathName(path string) (string, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}

	bufferSize := uint32(windows.MAX_PATH)
	for bufferSize <= maxLongPathUTF16Units {
		buffer := make([]uint16, bufferSize)
		length, err := windows.GetLongPathName(pathPtr, &buffer[0], bufferSize)
		if err != nil {
			return "", err
		}
		if length < bufferSize {
			return filepath.Clean(windows.UTF16ToString(buffer[:length])), nil
		}
		bufferSize = length + 1
	}
	return "", fmt.Errorf("long path exceeds %d UTF-16 code units", maxLongPathUTF16Units)
}
