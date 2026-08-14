// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Cap on notifications per call, so a 5,000-file batch does not flood the client.
const maxProgressNotifications = 100

// progressReporter reports batch progress, but only to a client that asked for
// it with a token. A nil reporter is the no-token case and ignores every call.
type progressReporter struct {
	session *mcp.ServerSession
	token   any
	total   int
	stride  int
}

func newProgressReporter(req *mcp.CallToolRequest, total int) *progressReporter {
	if req == nil || req.Session == nil || req.Params == nil || total <= 0 {
		return nil
	}
	token := req.Params.GetProgressToken()
	if token == nil {
		return nil
	}
	return &progressReporter{
		session: req.Session,
		token:   token,
		total:   total,
		// Rounded up, so the count stays inside the cap rather than just near it.
		stride: (total + maxProgressNotifications - 1) / maxProgressNotifications,
	}
}

// step reports that done items of total are finished. Intermediate steps are
// thinned to the stride; the last one always goes out.
func (p *progressReporter) step(ctx context.Context, done int, message string) {
	if p == nil || (done%p.stride != 0 && done != p.total) {
		return
	}
	err := p.session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
		ProgressToken: p.token,
		Progress:      float64(done),
		Total:         float64(p.total),
		Message:       message,
	})
	if err != nil {
		slog.Debug("progress notification failed", "done", done, "total", p.total, "err", err)
	}
}
