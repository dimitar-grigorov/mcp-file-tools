// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package workpool

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunOrdered_CommitsInInputOrder(t *testing.T) {
	items := []int{0, 1, 2, 3}
	gates := make([]chan struct{}, len(items))
	for i := range gates {
		gates[i] = make(chan struct{})
	}
	started := make(chan int, len(items))
	var committed []int
	done := make(chan Stats, 1)

	go func() {
		done <- RunOrdered(context.Background(), items, Options{Workers: len(items)},
			func(_ context.Context, index, item int) int {
				started <- index
				<-gates[index]
				return item
			},
			func(_ int, res int) bool {
				committed = append(committed, res)
				return true
			})
	}()

	waitForSignals(t, started, len(items))
	// Finish in reverse order; commits must still come out in input order.
	for i := len(gates) - 1; i >= 0; i-- {
		close(gates[i])
	}

	stats := waitForStats(t, done)
	if !reflect.DeepEqual(committed, items) {
		t.Fatalf("committed = %v, want %v", committed, items)
	}
	if stats.Dispatched != len(items) || stats.Completed != len(items) || stats.Committed != len(items) {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestRunOrdered_BoundsConcurrency(t *testing.T) {
	const workers = 3
	items := make([]int, 64)
	started := make(chan int, len(items))
	release := make(chan struct{})
	var active, peak atomic.Int32
	done := make(chan Stats, 1)

	go func() {
		done <- RunOrdered(context.Background(), items, Options{Workers: workers},
			func(_ context.Context, index, item int) int {
				current := active.Add(1)
				for {
					observed := peak.Load()
					if current <= observed || peak.CompareAndSwap(observed, current) {
						break
					}
				}
				started <- index
				<-release
				active.Add(-1)
				return item
			},
			func(_ int, _ int) bool { return true })
	}()

	waitForSignals(t, started, workers)
	if got := active.Load(); got != workers {
		t.Fatalf("active workers = %d, want %d", got, workers)
	}
	close(release)

	stats := waitForStats(t, done)
	if got := peak.Load(); got > workers {
		t.Fatalf("peak concurrency = %d, want <= %d", got, workers)
	}
	if stats.Committed != len(items) {
		t.Fatalf("committed = %d, want %d", stats.Committed, len(items))
	}
}

func TestRunOrdered_StopsDispatchAfterCancel(t *testing.T) {
	const workers = 4
	items := make([]int, 32)
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan int, workers)
	done := make(chan Stats, 1)

	go func() {
		done <- RunOrdered(ctx, items, Options{Workers: workers},
			func(ctx context.Context, index, item int) int {
				started <- index
				<-ctx.Done()
				return item
			},
			func(_ int, _ int) bool { return true })
	}()

	waitForSignals(t, started, workers)
	cancel()

	stats := waitForStats(t, done)
	if !stats.Cancelled {
		t.Fatalf("cancelled = false, want true: %+v", stats)
	}
	if stats.Dispatched != workers {
		t.Fatalf("dispatched = %d, want %d", stats.Dispatched, workers)
	}
	if stats.Committed != workers {
		t.Fatalf("committed = %d, want %d", stats.Committed, workers)
	}
}

func TestRunOrdered_DispatchAfterCancelFillsEveryPosition(t *testing.T) {
	items := make([]int, 20)
	for i := range items {
		items[i] = i
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var committed []int
	var sawLiveContext atomic.Bool

	stats := RunOrdered(ctx, items, Options{Workers: 3, DispatchAfterCancel: true},
		func(ctx context.Context, _ int, item int) int {
			if ctx.Err() == nil {
				sawLiveContext.Store(true)
			}
			return item
		},
		func(_ int, res int) bool {
			committed = append(committed, res)
			return true
		})

	if sawLiveContext.Load() {
		t.Fatal("worker context should inherit the cancelled parent")
	}
	if !reflect.DeepEqual(committed, items) {
		t.Fatalf("committed = %v, want %v", committed, items)
	}
	if stats.Dispatched != len(items) || stats.Committed != len(items) {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestRunOrdered_EarlyStopBoundsDispatch(t *testing.T) {
	items := make([]int, 20)
	stats := RunOrdered(context.Background(), items, Options{Workers: 4},
		func(_ context.Context, index, _ int) int { return index },
		func(index int, _ int) bool { return index < 2 })

	if !stats.StoppedEarly {
		t.Fatalf("stoppedEarly = false, want true: %+v", stats)
	}
	if stats.Committed != 3 {
		t.Fatalf("committed = %d, want 3", stats.Committed)
	}
	// 4 primed + one refill per successful commit.
	if stats.Dispatched > 6 {
		t.Fatalf("dispatched = %d, want <= 6", stats.Dispatched)
	}
}

func TestRunOrdered_NoItems(t *testing.T) {
	stats := RunOrdered(context.Background(), []int(nil), Options{},
		func(_ context.Context, _ int, item int) int { t.Fatal("work should not run"); return item },
		func(_ int, _ int) bool { t.Fatal("commit should not run"); return true })
	if stats != (Stats{}) {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func waitForSignals(t *testing.T, signals <-chan int, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		select {
		case <-signals:
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for %d worker signals", count)
		}
	}
}

func waitForStats(t *testing.T, done <-chan Stats) Stats {
	t.Helper()
	select {
	case stats := <-done:
		return stats
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for RunOrdered")
		return Stats{}
	}
}
