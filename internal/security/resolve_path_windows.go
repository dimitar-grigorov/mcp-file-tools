//go:build windows

package security

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

const maxFinalPathUTF16Units = 1 << 16

// resolveExistingPath canonicalizes an existing path, resolving junctions too.
func resolveExistingPath(path string) (string, error) {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}

	handle, err := windows.CreateFile(
		pathPtr,
		0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)

	bufferSize := uint32(512)
	for bufferSize <= maxFinalPathUTF16Units {
		buffer := make([]uint16, bufferSize)
		length, err := windows.GetFinalPathNameByHandle(handle, &buffer[0], bufferSize, 0)
		if err != nil {
			return "", err
		}
		if length < bufferSize {
			return filepath.Clean(normalizeWindowsFinalPath(windows.UTF16ToString(buffer[:length]))), nil
		}
		bufferSize = length + 1
	}
	return "", fmt.Errorf("resolved path exceeds %d UTF-16 code units", maxFinalPathUTF16Units)
}

func normalizeWindowsFinalPath(path string) string {
	switch {
	case strings.HasPrefix(path, `\\?\UNC\`):
		return `\\` + strings.TrimPrefix(path, `\\?\UNC\`)
	case strings.HasPrefix(path, `\\?\`):
		return strings.TrimPrefix(path, `\\?\`)
	case strings.HasPrefix(path, `\??\UNC\`):
		return `\\` + strings.TrimPrefix(path, `\??\UNC\`)
	case strings.HasPrefix(path, `\??\`):
		return strings.TrimPrefix(path, `\??\`)
	default:
		return path
	}
}
