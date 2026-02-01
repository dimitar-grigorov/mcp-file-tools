// Package config provides configuration management for MCP file tools server.
package config

import (
	"os"
	"strconv"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
)

const (
	// Environment variable names
	EnvDefaultEncoding = "MCP_DEFAULT_ENCODING"
	EnvMaxFileSize     = "MCP_MAX_FILE_SIZE"

	// Default values
	DefaultEncoding = "cp1251"
	DefaultMaxSize  = int64(10 * 1024 * 1024) // 10MB
)

// Config holds server configuration loaded from environment variables.
type Config struct {
	// DefaultEncoding is the default encoding for write_file when none is specified.
	// Set via MCP_DEFAULT_ENCODING environment variable.
	// Default: "cp1251" (for backward compatibility with legacy codebases)
	DefaultEncoding string

	// MaxFileSize is the maximum file size in bytes for read/write operations.
	// Set via MCP_MAX_FILE_SIZE environment variable.
	// Default: 10485760 (10MB)
	MaxFileSize int64
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	cfg := &Config{
		DefaultEncoding: DefaultEncoding,
		MaxFileSize:     DefaultMaxSize,
	}

	// Load default encoding from environment
	if enc := os.Getenv(EnvDefaultEncoding); enc != "" {
		// Validate encoding exists
		if _, ok := encoding.Get(enc); ok {
			cfg.DefaultEncoding = enc
		}
		// If invalid encoding, silently use default (cp1251)
	}

	// Load max file size from environment
	if sizeStr := os.Getenv(EnvMaxFileSize); sizeStr != "" {
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil && size > 0 {
			cfg.MaxFileSize = size
		}
	}

	return cfg
}
