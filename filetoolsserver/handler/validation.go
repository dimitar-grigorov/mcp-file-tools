// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// cancelled returns an error result once the caller gave up; check it right before a mutating syscall.
func cancelled(ctx context.Context) *mcp.CallToolResult {
	select {
	case <-ctx.Done():
		return errorResult(ctx.Err().Error())
	default:
		return nil
	}
}

// shouldLoadEntireFile reports whether path fits under the memory threshold, and its size; true on a stat error.
func (h *Handler) shouldLoadEntireFile(path string) (bool, int64) {
	info, err := os.Stat(path)
	if err != nil {
		return true, 0
	}
	return info.Size() <= h.config.MemoryThreshold, info.Size()
}

// PathValidationResult holds the result of path validation.
type PathValidationResult struct {
	Path   string
	Result *mcp.CallToolResult
	Err    error
}

// Ok returns true if validation succeeded.
func (r PathValidationResult) Ok() bool {
	return r.Err == nil
}

// ValidatePath checks that a path is non-empty and within allowed directories.
func (h *Handler) ValidatePath(path string) PathValidationResult {
	if path == "" {
		return PathValidationResult{
			Result: errorResult(ErrPathRequired.Error()),
			Err:    ErrPathRequired,
		}
	}

	validatedPath, err := h.validatePath(path)
	if err != nil {
		return PathValidationResult{
			Result: errorResult(err.Error()),
			Err:    err,
		}
	}

	return PathValidationResult{Path: validatedPath}
}

// ValidateEntryPath is ValidatePath for delete and move: a symlink names itself, not its target.
func (h *Handler) ValidateEntryPath(path string) PathValidationResult {
	return h.validateNamedPath(path, "path", h.validateEntryPath)
}

// ValidateSourceDest leaves destination zero when source already failed.
func (h *Handler) ValidateSourceDest(source, destination string) (PathValidationResult, PathValidationResult) {
	return h.validateSourceDest(source, destination, h.validatePath)
}

// ValidateMoveSourceDest is ValidateSourceDest with the source taken as the entry itself, so a move renames a symlink.
func (h *Handler) ValidateMoveSourceDest(source, destination string) (PathValidationResult, PathValidationResult) {
	return h.validateSourceDest(source, destination, h.validateEntryPath)
}

func (h *Handler) validateSourceDest(source, destination string, validateSource func(string) (string, error)) (PathValidationResult, PathValidationResult) {
	srcResult := h.validateNamedPath(source, "source", validateSource)
	if !srcResult.Ok() {
		return srcResult, PathValidationResult{}
	}
	return srcResult, h.validateNamedPath(destination, "destination", h.validatePath)
}

// validateNamedPath names the parameter in the empty-path message, so a two-path tool says which one was missing.
func (h *Handler) validateNamedPath(path, param string, validate func(string) (string, error)) PathValidationResult {
	if path == "" {
		return PathValidationResult{
			Result: errorResult(param + " is required and must be a non-empty string"),
			Err:    ErrPathRequired,
		}
	}
	validated, err := validate(path)
	if err != nil {
		return PathValidationResult{Result: errorResult(err.Error()), Err: err}
	}
	return PathValidationResult{Path: validated}
}
