// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type noticeOut struct {
	OK bool `json:"ok"`
}

// The notice rides once on the next good result, after the JSON block clients parse.
func TestAppendUpdateNotice(t *testing.T) {
	h := NewHandler([]string{t.TempDir()})
	server := mcp.NewServer(&mcp.Implementation{Name: "t", Version: "0"}, nil)
	server.AddReceivingMiddleware(h.AppendUpdateNotice)
	mcp.AddTool(server, &mcp.Tool{Name: "ok"}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, noticeOut, error) {
		return nil, noticeOut{OK: true}, nil
	})
	mcp.AddTool(server, &mcp.Tool{Name: "fail"}, func(context.Context, *mcp.CallToolRequest, struct{}) (*mcp.CallToolResult, any, error) {
		return errorResult("boom"), nil, nil
	})

	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := server.Connect(context.Background(), t1, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "c", Version: "0"}, nil).Connect(context.Background(), t2, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	texts := func(tool string) []string {
		t.Helper()
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: tool})
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, c := range res.Content {
			out = append(out, c.(*mcp.TextContent).Text)
		}
		return out
	}

	if got := texts("ok"); len(got) != 1 {
		t.Fatalf("nothing pending, got %q", got)
	}
	h.setUpdateNotice("v2 is out")
	if got := texts("fail"); len(got) != 1 {
		t.Fatalf("an error result keeps its message alone, got %q", got)
	}
	if got := texts("ok"); len(got) != 2 || got[0] != `{"ok":true}` || got[1] != "Tell the user once: v2 is out" {
		t.Fatalf("got %q, want the JSON block then the notice", got)
	}
	if got := texts("ok"); len(got) != 1 {
		t.Fatalf("notice repeated: %q", got)
	}
	h.setUpdateNotice("late startup check")
	if got := texts("ok"); len(got) != 1 {
		t.Fatalf("a check after delivery repeated it: %q", got)
	}
}

// check_for_updates answers the question itself, so the startup notice must not follow it.
func TestCheckForUpdatesSettlesNotice(t *testing.T) {
	h := NewHandler([]string{t.TempDir()})
	if _, _, err := h.NewCheckUpdateHandler("dev")(context.Background(), &mcp.CallToolRequest{}, CheckUpdateInput{}); err != nil {
		t.Fatal(err)
	}
	h.setUpdateNotice("v2 is out")
	if h.updateNotice.Load() != &noticeDelivered {
		t.Fatal("startup notice queued after check_for_updates already ran")
	}
}
