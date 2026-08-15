// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

// Package config provides configuration management for MCP file tools server.
package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/encoding"
)

const (
	// Environment variable names
	EnvDefaultEncoding     = "MCP_DEFAULT_ENCODING"
	EnvMemoryThreshold     = "MCP_MEMORY_THRESHOLD"
	EnvDefaultLineEnding   = "MCP_DEFAULT_LINE_ENDINGS"
	EnvDetectionCandidates = "MCP_DETECTION_CANDIDATES"

	// Default values
	DefaultEncoding = "utf-8"
	DefaultMaxSize  = int64(64 * 1024 * 1024) // 64MB - files smaller than this are loaded into memory
)

// Config holds server configuration loaded from environment variables.
type Config struct {
	// DefaultEncoding is the write fallback for new files and inconclusive detection.
	DefaultEncoding string

	// DefaultEncodingFromEnv reports whether MCP_DEFAULT_ENCODING set DefaultEncoding.
	DefaultEncodingFromEnv bool

	// MemoryThreshold: files below load fully, above stream; also picks full vs sample detection.
	MemoryThreshold int64

	// DefaultLineEndings ("crlf"/"lf") is the write fallback; empty writes content unchanged.
	DefaultLineEndings string

	// DetectionCandidates pins what detection may answer, in priority order; empty is unrestricted.
	DetectionCandidates []string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	cfg := &Config{
		DefaultEncoding: DefaultEncoding,
		MemoryThreshold: DefaultMaxSize,
	}

	if enc := os.Getenv(EnvDefaultEncoding); enc != "" {
		if _, ok := encoding.Get(enc); ok {
			cfg.DefaultEncoding = enc
			cfg.DefaultEncodingFromEnv = true
		} else {
			slog.Warn("invalid MCP_DEFAULT_ENCODING, using default", "value", enc, "fallback", DefaultEncoding)
		}
	}

	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvDefaultLineEnding))) {
	case "crlf":
		cfg.DefaultLineEndings = "crlf"
	case "lf":
		cfg.DefaultLineEndings = "lf"
	case "":
	default:
		slog.Warn("invalid MCP_DEFAULT_LINE_ENDINGS, ignoring", "value", os.Getenv(EnvDefaultLineEnding))
	}

	cfg.DetectionCandidates = parseDetectionCandidates(os.Getenv(EnvDetectionCandidates))

	if sizeStr := os.Getenv(EnvMemoryThreshold); sizeStr != "" {
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil && size > 0 {
			cfg.MemoryThreshold = size
		}
	}

	return cfg
}

// parseDetectionCandidates canonicalizes a comma-separated list, skipping names the registry does not know.
func parseDetectionCandidates(raw string) []string {
	var candidates []string
	for name := range strings.SplitSeq(raw, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if canonical, ok := encoding.Canonical(name); ok {
			candidates = append(candidates, canonical)
		} else {
			slog.Warn("unknown encoding in "+EnvDetectionCandidates+", skipping", "value", name)
		}
	}
	return candidates
}
