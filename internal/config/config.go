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
	DefaultEncoding = "cp1251"
	DefaultMaxSize  = int64(64 * 1024 * 1024) // 64MB - files smaller than this are loaded into memory
)

// Config holds server configuration loaded from environment variables.
type Config struct {
	// DefaultEncoding is the default encoding for write_file when none is specified.
	// Set via MCP_DEFAULT_ENCODING environment variable.
	// Default: "cp1251" (for backward compatibility with legacy codebases)
	DefaultEncoding string

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
