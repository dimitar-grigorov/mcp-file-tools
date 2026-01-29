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

	// Parse allowed directories from CLI arguments
	allowedDirs := os.Args[1:]

	// Require at least one allowed directory
	if len(allowedDirs) == 0 {
		fmt.Fprintf(os.Stderr, "Error: At least one allowed directory required\n")
		fmt.Fprintf(os.Stderr, "Usage: mcp-file-tools <directory1> [directory2] ...\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  mcp-file-tools /home/user/project\n")
		fmt.Fprintf(os.Stderr, "  mcp-file-tools D:\\Projects C:\\Users\\user\\Documents\n")
		os.Exit(1)
	}

	// Normalize and validate allowed directories
	normalized, err := security.NormalizeAllowedDirs(allowedDirs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Create MCP server with allowed directories
	server := filetoolsserver.NewServer(normalized)

	// Create stdio transport
	transport := mcp.NewStdioTransport()

	// Run server
	ctx := context.Background()
	if err := server.Run(ctx, transport); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
