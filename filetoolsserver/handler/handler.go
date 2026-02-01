package handler

import (
	"sync"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/config"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
)

// Handler handles all file tool operations
type Handler struct {
	config      *config.Config
	allowedDirs []string
	mu          sync.RWMutex
}

// Option is a functional option for configuring Handler
type Option func(*Handler)

// WithConfig sets the configuration for the handler
func WithConfig(cfg *config.Config) Option {
	return func(h *Handler) {
		if cfg != nil {
			h.config = cfg
		}
	}
}

// NewHandler creates a new Handler with allowed directories and optional configuration.
// If no config is provided via WithConfig, default configuration is used.
func NewHandler(allowedDirs []string, opts ...Option) *Handler {
	h := &Handler{
		config:      config.Load(), // Load defaults from environment
		allowedDirs: allowedDirs,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// GetConfig returns the handler's configuration
func (h *Handler) GetConfig() *config.Config {
	return h.config
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
