package handler

import (
	"os"
	"sync"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/config"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
)

// Default permissions for new files and directories
const (
	DefaultFileMode os.FileMode = 0644
	DefaultDirMode  os.FileMode = 0755
)

// Handler handles all file tool operations
type Handler struct {
	config      *config.Config
	cliDirs     []string // immutable baseline from CLI args; always allowed
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
	// Normalize the CLI baseline so dedup against (already-normalized) MCP roots
	// is reliable regardless of how the caller passed the paths. Best-effort:
	// fall back to the raw paths if normalization fails.
	cliDirs, err := security.NormalizeAllowedDirs(allowedDirs)
	if err != nil {
		cliDirs = make([]string, len(allowedDirs))
		copy(cliDirs, allowedDirs)
	}

	h := &Handler{
		config:      config.Load(), // Load defaults from environment
		cliDirs:     cliDirs,
		allowedDirs: allowedDirs,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// GetAllowedDirectories returns a copy of the allowed directories.
func (h *Handler) GetAllowedDirectories() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	dirs := make([]string, len(h.allowedDirs))
	copy(dirs, h.allowedDirs)
	return dirs
}

// ResolvedAllowedDirs returns allowed directories with symlinks resolved.
func (h *Handler) ResolvedAllowedDirs() []string {
	return security.ResolveAllowedDirs(h.GetAllowedDirectories())
}

// UpdateAllowedDirectories updates the allowed directories (for MCP Roots protocol)
func (h *Handler) UpdateAllowedDirectories(newDirs []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.allowedDirs = newDirs
}

// MergeAllowedDirectories sets allowed directories to the union of the immutable
// CLI baseline and newDirs (deduped, baseline first). Used for the MCP Roots
// protocol so client-provided roots augment, rather than replace, CLI args.
func (h *Handler) MergeAllowedDirectories(newDirs []string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	seen := make(map[string]struct{}, len(h.cliDirs)+len(newDirs))
	merged := make([]string, 0, len(h.cliDirs)+len(newDirs))
	for _, dir := range append(append([]string{}, h.cliDirs...), newDirs...) {
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		merged = append(merged, dir)
	}

	h.allowedDirs = merged

	result := make([]string, len(merged))
	copy(result, merged)
	return result
}

// validatePath validates a path against allowed directories
func (h *Handler) validatePath(path string) (string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return security.ValidatePath(path, h.allowedDirs)
}

// getFileMode returns the file's current permissions, or DefaultFileMode if file doesn't exist.
func getFileMode(path string) os.FileMode {
	info, err := os.Stat(path)
	if err != nil {
		return DefaultFileMode
	}
	return info.Mode().Perm()
}
