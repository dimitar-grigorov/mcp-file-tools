package filetoolsserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dimitar-grigorov/mcp-file-tools/filetoolsserver/handler"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func createInitializedHandler(h *handler.Handler) func(context.Context, *mcp.ServerSession, *mcp.InitializedParams) {
	return func(ctx context.Context, ss *mcp.ServerSession, params *mcp.InitializedParams) {
		result, err := ss.ListRoots(ctx, &mcp.ListRootsParams{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to request roots from client: %v\n", err)
			return
		}

		if len(result.Roots) > 0 {
			updateAllowedDirectoriesFromRoots(h, result.Roots)
		} else {
			currentDirs := h.GetAllowedDirectories()
			if len(currentDirs) == 0 {
				fmt.Fprintf(os.Stderr, "Warning: No allowed directories configured. File operations will fail.\n")
				fmt.Fprintf(os.Stderr, "Provide directories via CLI arguments or ensure MCP client supports roots protocol.\n")
			}
		}
	}
}

func createRootsListChangedHandler(h *handler.Handler) func(context.Context, *mcp.ServerSession, *mcp.RootsListChangedParams) {
	return func(ctx context.Context, ss *mcp.ServerSession, params *mcp.RootsListChangedParams) {
		result, err := ss.ListRoots(ctx, &mcp.ListRootsParams{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to request updated roots from client: %v\n", err)
			return
		}

		updateAllowedDirectoriesFromRoots(h, result.Roots)
	}
}

func updateAllowedDirectoriesFromRoots(h *handler.Handler, roots []*mcp.Root) {
	validatedDirs := make([]string, 0, len(roots))

	for _, root := range roots {
		// Extract path from file:// URI
		rootPath := root.URI
		if len(rootPath) > 8 && rootPath[:8] == "file:///" {
			rootPath = rootPath[8:]
			// Windows: file:///C:/path -> C:/path
			if len(rootPath) > 2 && rootPath[1] == ':' {
				rootPath = filepath.FromSlash(rootPath)
			}
		}

		normalized, err := security.NormalizeAllowedDirs([]string{rootPath})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to normalize root directory %s: %v\n", rootPath, err)
			continue
		}

		if len(normalized) > 0 {
			validatedDirs = append(validatedDirs, normalized[0])
		}
	}

	if len(validatedDirs) > 0 {
		h.UpdateAllowedDirectories(validatedDirs)
		fmt.Fprintf(os.Stderr, "Updated allowed directories from MCP roots: %d directories\n", len(validatedDirs))
		for _, dir := range validatedDirs {
			fmt.Fprintf(os.Stderr, "  - %s\n", dir)
		}
	} else {
		fmt.Fprintf(os.Stderr, "No valid root directories provided by client\n")
	}
}
