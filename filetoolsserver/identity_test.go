// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package filetoolsserver

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServerIdentityReachesTheClient(t *testing.T) {
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

	info := clientSession.InitializeResult().ServerInfo
	if info.Title == "" || info.Description == "" || info.WebsiteURL == "" {
		t.Errorf("incomplete server info: %+v", info)
	}
	if len(info.Icons) != 1 {
		t.Fatalf("got %d icons, want 1", len(info.Icons))
	}
	// A remote URL would make a client's server list hit the network.
	if !strings.HasPrefix(info.Icons[0].Source, "data:image/svg+xml;base64,") {
		t.Errorf("icon is not a self-contained data URI: %q", info.Icons[0].Source)
	}
}
