package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/filetoolsserver"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// version is set at build time via ldflags
var version = "dev"

func main() {
	// Configure default logger to write to stderr (stdout is reserved for the
	// MCP protocol on stdio). Level defaults to Info; set MCP_LOG_LEVEL=debug
	// to surface slog.Debug output (e.g. normalized allowed directories).
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("MCP_LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	// Set version from build
	filetoolsserver.Version = version

	// Handle --version flag
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(version)
		return
	}

	// Parse allowed directories from CLI arguments (optional)
	allowedDirs := os.Args[1:]

	// Normalize and validate allowed directories if provided
	var normalized []string
	var err error
	if len(allowedDirs) > 0 {
		normalized, err = security.NormalizeAllowedDirs(allowedDirs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		slog.Debug("normalized allowed directories", "dirs", normalized)
	}

	// Create MCP server with allowed directories (can be empty, directories can be added dynamically)
	// Pass nil for logger to disable logging middleware (recovery still active)
	// Pass nil for config to load from environment variables (MCP_DEFAULT_ENCODING, MCP_MEMORY_THRESHOLD)
	server := filetoolsserver.NewServer(normalized, nil, nil)

	// Run server on stdio transport
	ctx := context.Background()
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
