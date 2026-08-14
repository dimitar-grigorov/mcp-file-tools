// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleMoveFile moves or renames a file or directory.
func (h *Handler) HandleMoveFile(ctx context.Context, req *mcp.CallToolRequest, input MoveFileInput) (*mcp.CallToolResult, MoveFileOutput, error) {
	src, dst := h.ValidateSourceDest(input.Source, input.Destination)
	if !src.Ok() {
		return src.Result, MoveFileOutput{}, nil
	}
	if !dst.Ok() {
		return dst.Result, MoveFileOutput{}, nil
	}

	if _, err := os.Stat(src.Path); os.IsNotExist(err) {
		return errorResult(fmt.Sprintf("source does not exist: %s", input.Source)), MoveFileOutput{}, nil
	}

	// Lstat, so a dangling symlink counts as present. A destination created
	// between here and the rename is still clobbered.
	if _, err := os.Lstat(dst.Path); err == nil {
		return errorResult(fmt.Sprintf("destination already exists: %s", input.Destination)), MoveFileOutput{}, nil
	} else if !os.IsNotExist(err) {
		return errorResult(fmt.Sprintf("failed to inspect destination: %v", err)), MoveFileOutput{}, nil
	}

	if r := cancelled(ctx); r != nil {
		return r, MoveFileOutput{}, nil
	}

	if err := os.Rename(src.Path, dst.Path); err != nil {
		return errorResult(fmt.Sprintf("failed to move file: %v", err)), MoveFileOutput{}, nil
	}

	message := fmt.Sprintf("Successfully moved %s to %s", input.Source, input.Destination)
	return &mcp.CallToolResult{}, MoveFileOutput{Message: message}, nil
}
