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

	"github.com/dimitar-grigorov/mcp-file-tools/v4/filetoolsserver"
	"github.com/dimitar-grigorov/mcp-file-tools/v4/filetoolsserver/handler"
	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/config"
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

	filetoolsserver.Version = version

	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(version)
		return
	}

	// Args are the allowed dirs; without them the baseline comes from env or the working directory.
	baseline, err := config.ResolveBaseline(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	logBaseline(baseline)

	// nil logger drops logging middleware, recovery stays; nil config reads env.
	server := filetoolsserver.NewServer(baseline.Dirs, nil, nil, handler.WithExplicitDirs(baseline.Explicit))

	ctx := context.Background()
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

// logBaseline puts what the server may reach, and why, in the client's log.
func logBaseline(b config.Baseline) {
	if len(b.Dirs) > 0 {
		fmt.Fprintf(os.Stderr, "mcp-file-tools: allowed directories from %s: %s\n",
			b.Reason, strings.Join(b.Dirs, ", "))
		slog.Debug("resolved allowed directories", "source", b.Source, "dirs", b.Dirs)
		return
	}

	fmt.Fprintf(os.Stderr, "mcp-file-tools: no allowed directories - %s. Pass them as arguments or set %s.\n",
		b.Reason, config.EnvAllowedDirs)
}
