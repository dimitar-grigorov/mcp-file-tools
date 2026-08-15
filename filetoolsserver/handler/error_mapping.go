// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"fmt"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/errkind"
)

// mapOperationError turns an error into a per-file message and code.
// path (if set) is used for not-found/permission messages; fallback covers unkinded errors.
func mapOperationError(err error, path, fallback string) (message, code string) {
	if err == nil {
		return "", ErrCodeNone
	}

	message, code = err.Error(), fallback
	switch errkind.Of(err) {
	case errkind.InvalidPath:
		code = ErrCodeInvalidPath
	case errkind.AccessDenied:
		code = ErrCodeAccessDenied
	case errkind.SymlinkEscape:
		code = ErrCodeSymlinkEscape
	case errkind.NotFound:
		code = ErrCodeNotFound
		if path != "" {
			message = fmt.Sprintf("file not found: %s", path)
		}
	case errkind.Permission:
		code = ErrCodePermission
		if path != "" {
			message = fmt.Sprintf("permission denied: %s", path)
		}
	case errkind.Encoding:
		code = ErrCodeEncoding
	case errkind.Cancelled:
		message, code = "operation cancelled", ErrCodeOperationFailed
	case errkind.InvalidInput:
		code = ErrCodeOperationFailed
	case errkind.Filesystem:
		code = ErrCodeIO
	}
	return message, code
}
