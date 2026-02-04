package filetoolsserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dimitar-grigorov/mcp-file-tools/filetoolsserver/handler"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
	"github.com/dimitar-grigorov/mcp-file-tools/internal/updater"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func createInitializedHandler(h *handler.Handler) func(context.Context, *mcp.InitializedRequest) {
	return func(ctx context.Context, req *mcp.InitializedRequest) {
		result, err := req.Session.ListRoots(ctx, &mcp.ListRootsParams{})
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

		// Async update check - doesn't block initialization
		go checkForUpdatesAsync(req.Session)
	}
}

// checkForUpdatesAsync checks for updates in the background and notifies the client
func checkForUpdatesAsync(session *mcp.ServerSession) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if msg := updater.Check(ctx, Version); msg != "" {
		_ = session.Log(ctx, &mcp.LoggingMessageParams{
			Level:  "notice",
			Logger: "update-checker",
			Data:   msg,
		})
	}
}

func createRootsListChangedHandler(h *handler.Handler) func(context.Context, *mcp.RootsListChangedRequest) {
	return func(ctx context.Context, req *mcp.RootsListChangedRequest) {
		result, err := req.Session.ListRoots(ctx, &mcp.ListRootsParams{})
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
