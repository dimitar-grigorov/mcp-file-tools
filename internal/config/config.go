// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

// Package config provides configuration management for MCP file tools server.
package config

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
)

const (
	// Environment variable names
	EnvDefaultEncoding = "MCP_DEFAULT_ENCODING"
	EnvMemoryThreshold = "MCP_MEMORY_THRESHOLD"

	// Default values
	DefaultEncoding = "utf-8"
	DefaultMaxSize  = int64(64 * 1024 * 1024) // 64MB - files smaller than this are loaded into memory
)

// Config holds server configuration loaded from environment variables.
type Config struct {
	// DefaultEncoding is the fallback for write_file/edit_file: new files, and
	// existing files with inconclusive detection (e.g. pure ASCII).
	// Set via MCP_DEFAULT_ENCODING. Default: "utf-8" ("cp1251" for legacy codebases).
	DefaultEncoding string

	// DefaultEncodingFromEnv reports whether MCP_DEFAULT_ENCODING set DefaultEncoding.
	DefaultEncodingFromEnv bool

	// MemoryThreshold is the threshold for loading files into memory vs streaming.
	// Files smaller than this are loaded entirely into memory for better performance.
	// Files larger use streaming I/O to reduce memory usage.
	// Also used as threshold for encoding detection mode (full vs sample).
	// Set via MCP_MEMORY_THRESHOLD environment variable.
	// Default: 67108864 (64MB)
	MemoryThreshold int64
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	cfg := &Config{
		DefaultEncoding: DefaultEncoding,
		MemoryThreshold: DefaultMaxSize,
	}

	// Load default encoding from environment
	if enc := os.Getenv(EnvDefaultEncoding); enc != "" {
		if _, ok := encoding.Get(enc); ok {
			cfg.DefaultEncoding = enc
			cfg.DefaultEncodingFromEnv = true
		} else {
			slog.Warn("invalid MCP_DEFAULT_ENCODING, using default", "value", enc, "fallback", DefaultEncoding)
		}
	}

	// Load memory threshold from environment
	if sizeStr := os.Getenv(EnvMemoryThreshold); sizeStr != "" {
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil && size > 0 {
			cfg.MemoryThreshold = size
		}
	}

	return cfg
}
