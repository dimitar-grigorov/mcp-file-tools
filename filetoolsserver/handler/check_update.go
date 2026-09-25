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

// NewCheckUpdateHandler: cached result by default (max 1 GitHub API call per 30 min); force=true bypasses it.
func (h *Handler) NewCheckUpdateHandler(version string) mcp.ToolHandlerFor[CheckUpdateInput, CheckUpdateOutput] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input CheckUpdateInput) (*mcp.CallToolResult, CheckUpdateOutput, error) {
		h.updateNotice.Store(&noticeDelivered) // this result tells it
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
		RootsOnly: !h.HasExplicitDirs(),
	}
	if session != nil {
		if params := session.InitializeParams(); params != nil && params.ClientInfo != nil {
			env.Client = install.DetectClient(params.ClientInfo.Name)
		}
	}
	return env
}

// CheckForUpdatesAsync runs the startup check. Call once on initialization.
func (h *Handler) CheckForUpdatesAsync(session *mcp.ServerSession, version string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if msg := updater.Check(ctx, version, false, h.installEnv(session)); msg != "" {
		h.setUpdateNotice(msg)
	}
}

// noticeDelivered marks the notice as given, so a late startup check cannot repeat it.
var noticeDelivered string

// Phrased as an instruction because models relay instructions and ignore trivia.
func (h *Handler) setUpdateNotice(msg string) {
	notice := "Tell the user once: " + msg
	h.updateNotice.CompareAndSwap(nil, &notice)
}

// AppendUpdateNotice adds a pending update notice to the next successful tool result.
// MCP log messages reach no model; a text block after the SDK's JSON one leaves that intact.
func (h *Handler) AppendUpdateNotice(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		res, err := next(ctx, method, req)
		if r, ok := res.(*mcp.CallToolResult); ok && err == nil && !r.IsError {
			if p := h.updateNotice.Load(); p != nil && p != &noticeDelivered && h.updateNotice.CompareAndSwap(p, &noticeDelivered) {
				r.Content = append(r.Content, &mcp.TextContent{Text: *p})
			}
		}
		return res, err
	}
}
