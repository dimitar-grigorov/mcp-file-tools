package handler

import (
	"sync"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
)

const (
	// DefaultEncoding is the default encoding used when none is specified
	DefaultEncoding = "cp1251"
)

// Handler handles all file tool operations
type Handler struct {
	defaultEncoding string
	allowedDirs     []string
	mu              sync.RWMutex
}

// NewHandler creates a new Handler with allowed directories
func NewHandler(allowedDirs []string) *Handler {
	return &Handler{
		defaultEncoding: DefaultEncoding,
		allowedDirs:     allowedDirs,
	}
}

// GetAllowedDirectories returns a copy of the allowed directories
func (h *Handler) GetAllowedDirectories() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	dirs := make([]string, len(h.allowedDirs))
	copy(dirs, h.allowedDirs)
	return dirs
}

// UpdateAllowedDirectories updates the allowed directories (for MCP Roots protocol)
func (h *Handler) UpdateAllowedDirectories(newDirs []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.allowedDirs = newDirs
}

// validatePath validates a path against allowed directories
func (h *Handler) validatePath(path string) (string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return security.ValidatePath(path, h.allowedDirs)
}
