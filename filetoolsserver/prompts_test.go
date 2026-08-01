// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package filetoolsserver

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func connectTestSession(t *testing.T) *mcp.ClientSession {
	t.Helper()
	server := NewServer([]string{t.TempDir()}, nil, nil)
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := server.Connect(context.Background(), t1, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := client.Connect(context.Background(), t2, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

func TestPromptsListedAndRendered(t *testing.T) {
	cs := connectTestSession(t)

	list, err := cs.ListPrompts(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, p := range list.Prompts {
		names[p.Name] = true
	}
	for _, want := range []string{"audit_encodings", "fix_mojibake", "migrate_to_utf8"} {
		if !names[want] {
			t.Errorf("prompt %q not listed (got %v)", want, names)
		}
	}

	res, err := cs.GetPrompt(context.Background(), &mcp.GetPromptParams{
		Name:      "migrate_to_utf8",
		Arguments: map[string]string{"path": `D:\legacy\proj`},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Messages[0].Content.(*mcp.TextContent).Text
	for _, want := range []string{`D:\legacy\proj`, "*.pas", "dryRun=true", "backup=true"} {
		if !strings.Contains(text, want) {
			t.Errorf("migrate_to_utf8 text missing %q", want)
		}
	}

	// The optional pattern argument must override the default.
	res, err = cs.GetPrompt(context.Background(), &mcp.GetPromptParams{
		Name:      "migrate_to_utf8",
		Arguments: map[string]string{"path": "x", "pattern": "*.dfm"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if text := res.Messages[0].Content.(*mcp.TextContent).Text; !strings.Contains(text, "*.dfm") {
		t.Errorf("pattern argument not applied: %s", text[:120])
	}
}
