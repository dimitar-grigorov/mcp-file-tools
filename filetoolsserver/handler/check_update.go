package handler

import (
	"context"
	"time"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/updater"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CheckUpdateInput is the input for check_for_updates.
type CheckUpdateInput struct {
	Force bool `json:"force,omitempty"`
}

// CheckUpdateOutput returns current and latest version info.
type CheckUpdateOutput struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	UpdateMessage  string `json:"updateMessage,omitempty"`
}

// NewCheckUpdateHandler returns a handler that checks for newer versions.
// Uses cached result by default (max 1 GitHub API call per 30 min).
// Set force=true to bypass cache.
func NewCheckUpdateHandler(version string) mcp.ToolHandlerFor[CheckUpdateInput, CheckUpdateOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input CheckUpdateInput) (*mcp.CallToolResult, CheckUpdateOutput, error) {
		msg := updater.Check(ctx, version, input.Force)
		latest := updater.CachedLatestVersion()
		if latest == "" {
			latest = version
		}

		return &mcp.CallToolResult{}, CheckUpdateOutput{
			CurrentVersion: version,
			LatestVersion:  latest,
			UpdateMessage:  msg,
		}, nil
	}
}

// CheckForUpdatesAsync checks for updates in the background and notifies via MCP logging.
// Called once on server initialization, before any tool calls.
func CheckForUpdatesAsync(session *mcp.ServerSession, version string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if msg := updater.Check(ctx, version, false); msg != "" {
		_ = session.Log(ctx, &mcp.LoggingMessageParams{
			Level:  "notice",
			Logger: "update-checker",
			Data:   msg,
		})
	}
}
