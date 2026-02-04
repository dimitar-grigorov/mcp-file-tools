package handler

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleCopyFile copies a file to a new location.
func (h *Handler) HandleCopyFile(ctx context.Context, req *mcp.CallToolRequest, input CopyFileInput) (*mcp.CallToolResult, CopyFileOutput, error) {
	src, dst := h.ValidateSourceDest(input.Source, input.Destination)
	if !src.Ok() {
		return src.Result, CopyFileOutput{}, nil
	}
	if !dst.Ok() {
		return dst.Result, CopyFileOutput{}, nil
	}

	srcInfo, err := os.Stat(src.Path)
	if os.IsNotExist(err) {
		return errorResult(fmt.Sprintf("source does not exist: %s", input.Source)), CopyFileOutput{}, nil
	}
	if err != nil {
		return errorResult(fmt.Sprintf("failed to access source: %v", err)), CopyFileOutput{}, nil
	}

	if srcInfo.IsDir() {
		return errorResult("source is a directory, only files can be copied"), CopyFileOutput{}, nil
	}

	if _, err := os.Stat(dst.Path); err == nil {
		return errorResult(fmt.Sprintf("destination already exists: %s", input.Destination)), CopyFileOutput{}, nil
	}

	// Copy file with source permissions and timestamps preserved
	if err := copyFile(src.Path, dst.Path, srcInfo.Mode().Perm(), srcInfo.ModTime()); err != nil {
		return errorResult(fmt.Sprintf("failed to copy file: %v", err)), CopyFileOutput{}, nil
	}

	message := fmt.Sprintf("Successfully copied %s to %s", input.Source, input.Destination)
	return &mcp.CallToolResult{}, CopyFileOutput{Message: message}, nil
}

func copyFile(src, dst string, mode os.FileMode, modTime time.Time) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	if err := dstFile.Sync(); err != nil {
		return err
	}

	return os.Chtimes(dst, modTime, modTime)
}
