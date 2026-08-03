// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package filetoolsserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/filetoolsserver/handler"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// listRootURIs isolates the deprecated roots API to one deletable function.
func listRootURIs(ctx context.Context, session *mcp.ServerSession) ([]string, error) {
	//lint:ignore SA1019 roots kept for pre-2026-07-28 clients
	result, err := session.ListRoots(ctx, &mcp.ListRootsParams{})
	if err != nil {
		return nil, err
	}

	uris := make([]string, 0, len(result.Roots))
	for _, root := range result.Roots {
		uris = append(uris, root.URI)
	}
	return uris, nil
}

func createInitializedHandler(h *handler.Handler) func(context.Context, *mcp.InitializedRequest) {
	return func(ctx context.Context, req *mcp.InitializedRequest) {
		// Async update check — runs regardless of roots support.
		go h.CheckForUpdatesAsync(req.Session, Version)

		uris, err := listRootURIs(ctx, req.Session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to request roots from client: %v\n", err)
			return
		}

		if len(uris) > 0 {
			updateAllowedDirectoriesFromRoots(h, uris)
		} else {
			currentDirs := h.GetAllowedDirectories()
			if len(currentDirs) == 0 {
				fmt.Fprintf(os.Stderr, "Warning: No allowed directories configured. File operations will fail.\n")
				fmt.Fprintf(os.Stderr, "Provide directories via CLI arguments or ensure MCP client supports roots protocol.\n")
			}
		}
	}
}

func createRootsListChangedHandler(h *handler.Handler) func(context.Context, *mcp.RootsListChangedRequest) {
	return func(ctx context.Context, req *mcp.RootsListChangedRequest) {
		uris, err := listRootURIs(ctx, req.Session)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to request updated roots from client: %v\n", err)
			return
		}

		updateAllowedDirectoriesFromRoots(h, uris)
	}
}

// fileURIToPath converts a file:// URI to a local filesystem path.
func fileURIToPath(uri string) string {
	if !strings.HasPrefix(uri, "file://") {
		return uri
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	path := parsed.Path
	// Windows: url.Parse turns file:///C:/path into /C:/path — strip the leading slash
	if runtime.GOOS == "windows" && len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}

	return path
}

func updateAllowedDirectoriesFromRoots(h *handler.Handler, rootURIs []string) {
	validatedDirs := make([]string, 0, len(rootURIs))

	for _, uri := range rootURIs {
		rootPath := fileURIToPath(uri)

		normalized, err := security.NormalizeAllowedDirs([]string{rootPath})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to normalize root directory %s: %v\n", rootPath, err)
			continue
		}

		if len(normalized) > 0 {
			validatedDirs = append(validatedDirs, normalized[0])
		}
	}

	// Merge unconditionally: an empty or fully invalid list must revoke roots the
	// client granted earlier, leaving only the CLI baseline.
	merged := h.MergeAllowedDirectories(validatedDirs)
	slog.Debug("merged allowed directories from MCP roots",
		"roots", validatedDirs, "merged", merged)
}
