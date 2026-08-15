// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"time"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/install"
	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/updater"
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
	InstallMethod  string `json:"installMethod"`
}

// NewCheckUpdateHandler returns a handler that checks for newer versions.
// Uses cached result by default (max 1 GitHub API call per 30 min).
// Set force=true to bypass cache.
func (h *Handler) NewCheckUpdateHandler(version string) mcp.ToolHandlerFor[CheckUpdateInput, CheckUpdateOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input CheckUpdateInput) (*mcp.CallToolResult, CheckUpdateOutput, error) {
		env := h.installEnv(req.Session)
		msg := updater.Check(ctx, version, input.Force, env)
		latest := updater.CachedLatestVersion()
		if latest == "" {
			latest = version
		}

		return &mcp.CallToolResult{}, CheckUpdateOutput{
			CurrentVersion: version,
			LatestVersion:  latest,
			UpdateMessage:  msg,
			InstallMethod:  string(env.Method),
		}, nil
	}
}

// installEnv describes this setup so update steps match it.
func (h *Handler) installEnv(session *mcp.ServerSession) install.Env {
	env := install.Env{
		Method:    install.DetectMethod(),
		RootsOnly: !h.HasCLIDirs(),
	}
	if session != nil {
		if params := session.InitializeParams(); params != nil && params.ClientInfo != nil {
			env.Client = install.DetectClient(params.ClientInfo.Name)
		}
	}
	return env
}

// CheckForUpdatesAsync checks for updates in the background and notifies via MCP logging.
// Called once on server initialization, before any tool calls.
func (h *Handler) CheckForUpdatesAsync(session *mcp.ServerSession, version string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if msg := updater.Check(ctx, version, false, h.installEnv(session)); msg != "" {
		//lint:ignore SA1019 logging deprecated in 2026-07-28, still reaches older clients
		_ = session.Log(ctx, &mcp.LoggingMessageParams{
			Level:  "notice",
			Logger: "update-checker",
			Data:   msg,
		})
	}
}
