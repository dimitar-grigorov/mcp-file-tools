// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
)

// Locks in the per-file codes read_multiple reports, so the mapper can't reclassify silently.
func TestReadSingleFileErrorCodes(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	subDir := filepath.Join(tempDir, "sub")
	if err := os.Mkdir(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		path     string
		wantCode string
		wantMsg  string
	}{
		{name: "empty path", path: "", wantCode: ErrCodeInvalidPath, wantMsg: ErrPathRequired.Error()},
		{name: "missing file", path: filepath.Join(tempDir, "missing.txt"), wantCode: ErrCodeNotFound, wantMsg: "file not found: "},
		{name: "outside allowed", path: filepath.Join(tempDir, "..", "..", "etc", "passwd"), wantCode: ErrCodeAccessDenied, wantMsg: "access denied"},
		// Missing suffix is re-projected, so validation passes and the read fails.
		{name: "missing parent", path: filepath.Join(tempDir, "nope", "deep", "file.txt"), wantCode: ErrCodeNotFound, wantMsg: "file not found: "},
		{name: "directory", path: subDir, wantCode: ErrCodeIO, wantMsg: "failed to read file: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.readSingleFile(tt.path, "")
			if got.ErrorCode != tt.wantCode {
				t.Errorf("ErrorCode = %q, want %q (error: %q)", got.ErrorCode, tt.wantCode, got.Error)
			}
			if !strings.Contains(got.Error, tt.wantMsg) {
				t.Errorf("Error = %q, want it to contain %q", got.Error, tt.wantMsg)
			}
		})
	}
}

func TestSecuritySentinelsKeepMessagesAndMatching(t *testing.T) {
	sentinels := []struct {
		err  error
		want string
	}{
		{security.ErrNoAllowedDirs, "no allowed directories configured - please provide directories via CLI arguments or MCP roots protocol"},
		{security.ErrPathDenied, "access denied - path outside allowed directories"},
		{security.ErrSymlinkDenied, "access denied - symlink target outside allowed directories"},
		{security.ErrParentDirDenied, "access denied - parent directory outside allowed directories"},
		{security.ErrParentNotExists, "parent directory does not exist"},
		{security.ErrNotDirectory, "path is not a directory"},
		{ErrPathRequired, "path is required and must be a non-empty string"},
		{ErrPatternRequired, "pattern is required and must be a non-empty string"},
		{ErrEditsRequired, "edits array is required and must not be empty"},
		{ErrPathMustBeDirectory, "path must be a directory"},
		{ErrEncodingUnsupported, "unsupported encoding"},
		{ErrEditNoMatch, "could not find exact match for edit"},
		{ErrOldTextEmpty, "edit old_text cannot be empty"},
	}

	for _, s := range sentinels {
		if got := s.err.Error(); got != s.want {
			t.Errorf("message = %q, want %q", got, s.want)
		}
		if wrapped := fmt.Errorf("%w: extra", s.err); !errors.Is(wrapped, s.err) {
			t.Errorf("errors.Is broken for %q", s.want)
		}
	}
}

func TestMapOperationErrorFallbacks(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		path     string
		fallback string
		wantMsg  string
		wantCode string
	}{
		{
			name: "nil error", err: nil, fallback: ErrCodeIO,
			wantMsg: "", wantCode: ErrCodeNone,
		},
		{
			name: "unclassified uses fallback", err: errors.New("disk on fire"), fallback: ErrCodeIO,
			wantMsg: "disk on fire", wantCode: ErrCodeIO,
		},
		{
			name: "not found rewrites message when path given",
			err:  fmt.Errorf("open x: %w", fs.ErrNotExist), path: "x.txt", fallback: ErrCodeIO,
			wantMsg: "file not found: x.txt", wantCode: ErrCodeNotFound,
		},
		{
			name: "not found keeps message without path",
			err:  fmt.Errorf("open x: %w", fs.ErrNotExist), fallback: ErrCodeIO,
			wantMsg: "open x: " + fs.ErrNotExist.Error(), wantCode: ErrCodeNotFound,
		},
		{
			name: "cancelled", err: context.Canceled, fallback: ErrCodeIO,
			wantMsg: "operation cancelled", wantCode: ErrCodeOperationFailed,
		},
		{
			name: "symlink escape", err: fmt.Errorf("%w: /out", security.ErrSymlinkDenied), fallback: ErrCodeInvalidPath,
			wantMsg: "access denied - symlink target outside allowed directories: /out", wantCode: ErrCodeSymlinkEscape,
		},
		{
			name: "unsupported encoding", err: fmt.Errorf("%w: bogus", ErrEncodingUnsupported), fallback: ErrCodeIO,
			wantMsg: "unsupported encoding: bogus", wantCode: ErrCodeEncoding,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, code := mapOperationError(tt.err, tt.path, tt.fallback)
			if msg != tt.wantMsg || code != tt.wantCode {
				t.Errorf("mapOperationError() = (%q, %q), want (%q, %q)", msg, code, tt.wantMsg, tt.wantCode)
			}
		})
	}
}
