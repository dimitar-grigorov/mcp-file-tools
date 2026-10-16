// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

//go:build windows

package security

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

const maxLongPathUTF16Units = 1 << 16

// NTFS folds case like CompareStringOrdinal; EqualFold and ToUpper differ on hundreds of runes (Kelvin sign, long s).
var procCompareStringOrdinal = windows.NewLazySystemDLL("kernel32.dll").NewProc("CompareStringOrdinal")

const cstrEqual = 2

func equalFoldUTF16(a, b []uint16) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	ret, _, _ := procCompareStringOrdinal.Call(
		uintptr(unsafe.Pointer(&a[0])), uintptr(len(a)),
		uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)),
		1, // bIgnoreCase
	)
	return ret == cstrEqual
}

func pathsEqual(first, second string) bool {
	return equalFoldUTF16(utf16.Encode([]rune(first)), utf16.Encode([]rune(second)))
}

func pathHasPrefix(path, prefix string) bool {
	p, pre := utf16.Encode([]rune(path)), utf16.Encode([]rune(prefix))
	return len(p) >= len(pre) && equalFoldUTF16(p[:len(pre)], pre)
}

// expandShortPath expands 8.3 components ("PROGRA~1") so the lexical gate compares like with like; input on failure.
func expandShortPath(path string) string {
	current := filepath.Clean(path)
	var missing []string

	for {
		expanded, err := longPathName(current)
		if err == nil {
			for _, m := range slices.Backward(missing) {
				expanded = filepath.Join(expanded, m)
			}
			return filepath.Clean(expanded)
		}
		if !os.IsNotExist(err) {
			return path
		}

		// GetLongPathName needs an existing path: expand the nearest existing ancestor, re-attach the rest.
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
