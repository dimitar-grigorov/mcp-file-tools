// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"log/slog"
	"os"
	"sync"
	"sync/atomic"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/config"
	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/encoding"
	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/security"
)

// Default permissions for new files and directories
const (
	DefaultFileMode os.FileMode = 0644
	DefaultDirMode  os.FileMode = 0755
)

// Handler carries the allowed-directory set and config every tool call resolves against.
type Handler struct {
	config      *config.Config
	cliDirs     []string // immutable baseline from CLI args; always allowed
	allowedDirs []string
	mu          sync.RWMutex

	utf8NoticeOnce sync.Once // TODO(2027-01): remove with the utf-8 default transition notice
	plainUTF8Seen  sync.Map  // path -> struct{}, for the built-in-tooling hint
	plainUTF8Count atomic.Int64
}

// plainUTF8HintCap bounds the seen-set: a session past this has had the hint.
const plainUTF8HintCap = 1024

// Phrased as an instruction because models relay instructions and ignore trivia.
const plainUTF8Hint = "This file is plain utf-8 with no BOM, so your built-in file tools handle it correctly — " +
	"prefer them for it. These encoding-aware tools are for non-utf-8 files."

// Returns the hint the first time a path looks like plain utf-8, "" after that.
func (h *Handler) plainUTF8HintFor(path, encodingName string, hasBOM bool) string {
	if hasBOM || !encoding.IsUTF8(encodingName) {
		return ""
	}
	if h.plainUTF8Count.Load() >= plainUTF8HintCap {
		return ""
	}
	if _, seen := h.plainUTF8Seen.LoadOrStore(path, struct{}{}); seen {
		return ""
	}
	h.plainUTF8Count.Add(1)
	return plainUTF8Hint
}

// TODO(2027-01): drop the transition notice — it runs to the end of 2026. Phrased as an instruction because models relay instructions and ignore trivia.
const utf8DefaultNotice = " Tell the user once: new files now default to utf-8. " +
	"Before 2.0.0 the default was cp1251. Existing files still keep their own encoding. " +
	"To restore the old behaviour, set MCP_DEFAULT_ENCODING=cp1251."

// utf8TransitionNotice returns the notice once, on the first new file taking the built-in default. Silent if the user set it.
func (h *Handler) utf8TransitionNotice() string {
	if h.config.DefaultEncodingFromEnv {
		return ""
	}
	var notice string
	h.utf8NoticeOnce.Do(func() { notice = utf8DefaultNotice })
	return notice
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

// NewHandler creates a Handler for allowedDirs; without WithConfig the config comes from the environment.
func NewHandler(allowedDirs []string, opts ...Option) *Handler {
	cliDirs := normalizeAllowedDirs(allowedDirs)

	h := &Handler{
		config:      config.Load(),
		cliDirs:     cliDirs,
		allowedDirs: append([]string(nil), cliDirs...),
	}

	for _, opt := range opts {
		opt(h)
	}

	// Detection is package state in encoding, far below any handler; applied here so both config.Load and WithConfig reach it.
	if err := encoding.SetDetectionCandidates(h.config.DetectionCandidates); err != nil {
		slog.Warn("invalid "+config.EnvDetectionCandidates+", detection stays unrestricted", "error", err)
	}

	return h
}

func (h *Handler) GetAllowedDirectories() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	dirs := make([]string, len(h.allowedDirs))
	copy(dirs, h.allowedDirs)
	return dirs
}

// HasCLIDirs reports whether any directory came from CLI args rather than roots.
func (h *Handler) HasCLIDirs() bool {
	return len(h.cliDirs) > 0
}

// ResolvedAllowedDirs returns allowed directories with symlinks resolved.
func (h *Handler) ResolvedAllowedDirs() []string {
	return security.ResolveAllowedDirs(h.GetAllowedDirectories())
}

// UpdateAllowedDirectories updates the allowed directories (for MCP Roots protocol)
func (h *Handler) UpdateAllowedDirectories(newDirs []string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.allowedDirs = normalizeAllowedDirs(newDirs)
}

// normalizeAllowedDirs canonicalizes each dir so validations compare like with like (a Windows 8.3 root never matches a long path); per dir, so one bad path doesn't discard the rest.
func normalizeAllowedDirs(dirs []string) []string {
	normalized := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if one, err := security.NormalizeAllowedDir(dir); err == nil {
			normalized = append(normalized, one)
			continue
		}
		normalized = append(normalized, dir)
	}
	return normalized
}

// MergeAllowedDirectories sets the dirs to the deduped union of CLI baseline and newDirs — roots augment, not replace.
func (h *Handler) MergeAllowedDirectories(newDirs []string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	normalizedNew := normalizeAllowedDirs(newDirs)
	seen := make(map[string]struct{}, len(h.cliDirs)+len(normalizedNew))
	merged := make([]string, 0, len(h.cliDirs)+len(normalizedNew))
	for _, dirs := range [][]string{h.cliDirs, normalizedNew} {
		for _, dir := range dirs {
			if _, ok := seen[dir]; ok {
				continue
			}
			seen[dir] = struct{}{}
			merged = append(merged, dir)
		}
	}
	h.allowedDirs = merged

	result := make([]string, len(merged))
	copy(result, merged)
	return result
}

// validatePath resolves a path and rejects it if it lands outside the allowed dirs.
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
