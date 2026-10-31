// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package filetoolsserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/filetoolsserver/handler"
	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/config"
	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/security"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const rootsTimeout = 5 * time.Second

// listRootURIs isolates the deprecated roots API to one deletable function.
func listRootURIs(ctx context.Context, session *mcp.ServerSession) ([]string, error) {
	// A client that never answers would otherwise hold this handler forever.
	p := session.InitializeParams()
	//lint:ignore SA1019 roots kept for pre-2026-07-28 clients
	if p == nil || p.Capabilities == nil || p.Capabilities.RootsV2 == nil {
		return nil, errors.New("client does not support roots")
	}
	ctx, cancel := context.WithTimeout(ctx, rootsTimeout)
	defer cancel()
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
		h.StartUpdateCheck(req.Session, Version)

		// 2026-07-28 clients cannot be asked at all (SEP-2322), so failure is routine.
		uris, err := listRootURIs(ctx, req.Session)
		if err != nil {
			slog.Debug("client did not answer roots/list; keeping the startup baseline", "error", err)
		} else if len(uris) > 0 {
			updateAllowedDirectoriesFromRoots(h, uris)
		}

		if len(h.GetAllowedDirectories()) == 0 {
			fmt.Fprintf(os.Stderr, "Warning: no allowed directories - pass them as arguments or set %s.\n",
				config.EnvAllowedDirs)
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

		normalized, err := security.NormalizeAllowedDir(rootPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to normalize root directory %s: %v\n", rootPath, err)
			continue
		}
		validatedDirs = append(validatedDirs, normalized)
	}

	// Merge unconditionally: an empty or invalid list revokes earlier roots, leaving the baseline.
	merged := h.MergeAllowedDirectories(validatedDirs)
	slog.Debug("merged allowed directories from MCP roots",
		"roots", validatedDirs, "merged", merged)
}
