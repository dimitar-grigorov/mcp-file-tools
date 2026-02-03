package handler

import (
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// shouldLoadEntireFile returns true if the file is small enough to load into memory.
// Returns (shouldLoad, fileSize). On stat error, defaults to true (load into memory).
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

// ValidateSourceDest validates both source and destination paths.
func (h *Handler) ValidateSourceDest(source, destination string) (PathValidationResult, PathValidationResult) {
	srcResult := h.validateSourcePath(source)
	if !srcResult.Ok() {
		return srcResult, PathValidationResult{}
	}
	return srcResult, h.validateDestPath(destination)
}

func (h *Handler) validateSourcePath(path string) PathValidationResult {
	if path == "" {
		return PathValidationResult{
			Result: errorResult("source is required and must be a non-empty string"),
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

func (h *Handler) validateDestPath(path string) PathValidationResult {
	if path == "" {
		return PathValidationResult{
			Result: errorResult("destination is required and must be a non-empty string"),
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
