package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dimitar-grigorov/mcp-file-tools/filetoolsserver"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// version is set at build time via ldflags
var version = "dev"

func main() {
	// Set version from build
	filetoolsserver.Version = version

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
