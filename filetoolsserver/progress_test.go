// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package filetoolsserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// waitUntil blocks until the notifications satisfy done or within elapses, returning whatever arrived.
type waitUntil func(within time.Duration, done func([]*mcp.ProgressNotificationParams) bool) []*mcp.ProgressNotificationParams

// connectWithProgress collects a session's progress notifications. Waiting beats reading them straight
// after a call: the SDK retires a response on the read path but queues notifications for another goroutine.
func connectWithProgress(t *testing.T, dir string) (*mcp.ClientSession, waitUntil) {
	t.Helper()
	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	server := NewServer([]string{dir}, nil, nil)
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { serverSession.Close() })

	var mu sync.Mutex
	var seen []*mcp.ProgressNotificationParams
	arrived := make(chan struct{}, 1)
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, &mcp.ClientOptions{
		ProgressNotificationHandler: func(_ context.Context, req *mcp.ProgressNotificationClientRequest) {
			mu.Lock()
			seen = append(seen, req.Params)
			mu.Unlock()
			select {
			case arrived <- struct{}{}:
			default:
			}
		},
	})
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { clientSession.Close() })

	return clientSession, func(within time.Duration, done func([]*mcp.ProgressNotificationParams) bool) []*mcp.ProgressNotificationParams {
		deadline := time.After(within)
		for {
			mu.Lock()
			got := slices.Clone(seen)
			mu.Unlock()
			if done(got) {
				return got
			}
			select {
			case <-arrived:
			case <-deadline:
				return got
			}
		}
	}
}

// cp1251 files to convert, so the batch has real work to do.
func writeCP1251Files(t *testing.T, dir string, n int) []any {
	t.Helper()
	paths := make([]any, 0, n)
	for i := range n {
		path := filepath.Join(dir, fmt.Sprintf("unit%d.pas", i))
		// "Настройки" in cp1251.
		if err := os.WriteFile(path, []byte("\xCD\xE0\xF1\xF2\xF0\xEE\xE9\xEA\xE8\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	return paths
}

func callConvert(t *testing.T, session *mcp.ClientSession, paths []any, withToken bool) {
	t.Helper()
	params := &mcp.CallToolParams{
		Name:      "convert_encoding",
		Arguments: map[string]any{"paths": paths, "to": "utf-8", "dryRun": true},
	}
	if withToken {
		params.SetProgressToken("batch-1")
	}
	res, err := session.CallTool(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("convert_encoding failed: %v", res.Content)
	}
}

func TestBatchConvertReportsProgress(t *testing.T) {
	dir := t.TempDir()
	paths := writeCP1251Files(t, dir, 3)
	session, notifications := connectWithProgress(t, dir)

	callConvert(t, session, paths, true)

	seen := notifications(5*time.Second, func(s []*mcp.ProgressNotificationParams) bool { return len(s) >= len(paths) })
	if len(seen) != len(paths) {
		t.Fatalf("got %d notifications, want %d: %+v", len(seen), len(paths), seen)
	}
	for i, params := range seen {
		if params.Progress != float64(i+1) || params.Total != float64(len(paths)) {
			t.Errorf("notification %d = %v/%v", i, params.Progress, params.Total)
		}
		if params.Message != fmt.Sprintf("unit%d.pas", i) {
			t.Errorf("notification %d names %q", i, params.Message)
		}
	}
}

// No token means no notifications: the client never asked.
func TestBatchConvertQuietWithoutToken(t *testing.T) {
	dir := t.TempDir()
	paths := writeCP1251Files(t, dir, 3)
	session, notifications := connectWithProgress(t, dir)

	callConvert(t, session, paths, false)

	// Give one a chance to arrive rather than checking before it could have.
	seen := notifications(200*time.Millisecond, func(s []*mcp.ProgressNotificationParams) bool { return len(s) > 0 })
	if len(seen) != 0 {
		t.Errorf("got %d unrequested notifications: %+v", len(seen), seen)
	}
}

// Past the cap the steps thin out, and the last one still arrives.
func TestBatchConvertThinsLargeBatches(t *testing.T) {
	dir := t.TempDir()
	paths := writeCP1251Files(t, dir, 250)
	session, notifications := connectWithProgress(t, dir)

	callConvert(t, session, paths, true)

	seen := notifications(5*time.Second, func(s []*mcp.ProgressNotificationParams) bool {
		return len(s) > 0 && s[len(s)-1].Progress == 250
	})
	if len(seen) == 0 || len(seen) > 101 {
		t.Fatalf("got %d notifications for 250 files, want at most 101", len(seen))
	}
	if last := seen[len(seen)-1]; last.Progress != 250 {
		t.Errorf("last notification is %v/250, want the final step", last.Progress)
	}
}
