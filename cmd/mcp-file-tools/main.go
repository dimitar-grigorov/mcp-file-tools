// SPDX-License-Identifier: GPL-3.0-only
// mcp-file-tools - MCP server for file operations with non-UTF-8 encoding support.
// Copyright (C) 2026 Dimitar Grigorov <https://github.com/dimitar-grigorov/mcp-file-tools>
//
// Free software under the GNU General Public License version 3, distributed
// WITHOUT ANY WARRANTY. See LICENSE or <https://www.gnu.org/licenses/>.

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
	// Logs go to stderr; stdout is reserved for the MCP stdio protocol.
	// MCP_LOG_LEVEL (debug/warn/error) sets verbosity; defaults to info.
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

	// nil logger disables logging middleware (recovery stays on); nil config loads
	// MCP_DEFAULT_ENCODING/MCP_MEMORY_THRESHOLD from the environment.
	server := filetoolsserver.NewServer(normalized, nil, nil)

	// Run server on stdio transport
	ctx := context.Background()
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
