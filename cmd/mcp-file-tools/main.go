package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dimitar-grigorov/mcp-file-tools/filetoolsserver"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// version is set at build time via ldflags
var version = "dev"

func main() {
	// Set version from build
	filetoolsserver.Version = version

	// Create MCP server
	server := filetoolsserver.NewServer()

	// Create stdio transport
	transport := mcp.NewStdioTransport()

	// Run server
	ctx := context.Background()
	if err := server.Run(ctx, transport); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
