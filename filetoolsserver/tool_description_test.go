// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package filetoolsserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Claude Code silently truncates a tool description at 2KB.
const maxDescriptionBytes = 2048

func listTools(t *testing.T) []*mcp.Tool {
	t.Helper()
	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	server := NewServer([]string{t.TempDir()}, nil, nil)
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()

	res, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	return res.Tools
}

func TestToolDescriptionsFitClientLimit(t *testing.T) {
	tools := listTools(t)
	if len(tools) == 0 {
		t.Fatal("no tools registered")
	}
	for _, tool := range tools {
		if n := len(tool.Description); n > maxDescriptionBytes {
			t.Errorf("%s: description is %d bytes, over the %d byte limit", tool.Name, n, maxDescriptionBytes)
		} else {
			t.Logf("%-28s %4d bytes", tool.Name, n)
		}
	}
}
