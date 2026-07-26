// Package workpool runs bounded concurrent work and commits results in input order.
package workpool

import (
	"context"
	"runtime"
	"sync"
)

type Options struct {
	Workers             int  // <= 0 means runtime.NumCPU(), capped at len(items)
	DispatchAfterCancel bool // keep dispatching once ctx is cancelled, so every item gets a result
}

type Stats struct {
	Dispatched   int
	Completed    int
	Committed    int
	StoppedEarly bool
	Cancelled    bool
}

type job[T any] struct {
	index int
	value T
}

type result[T any] struct {
	index int
	value T
}

// RunOrdered runs work on items with bounded concurrency and calls commit serially in
// input order. commit returns false to stop; running workers are cancelled and drained.
func RunOrdered[In, Out any](
	ctx context.Context,
	items []In,
	opts Options,
	work func(context.Context, int, In) Out,
	commit func(int, Out) bool,
) Stats {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(items) == 0 {
		return Stats{Cancelled: ctx.Err() != nil}
	}
	workers := opts.Workers
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if workers < 1 {
		workers = 1
	}
	if workers > len(items) {
		workers = len(items)
	}

	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	jobs := make(chan job[In])
	results := make(chan result[Out], workers) // never blocks a worker: in-flight <= workers

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := range jobs {
				results <- result[Out]{index: j.index, value: work(workerCtx, j.index, j.value)}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	var stats Stats
	nextDispatch, nextCommit := 0, 0
	dispatching, committing := true, true

	stopDispatch := func() {
		if !dispatching {
			return
		}
		close(jobs)
		dispatching = false
	}
	dispatch := func() {
		jobs <- job[In]{index: nextDispatch, value: items[nextDispatch]}
		nextDispatch++
		stats.Dispatched++
		if nextDispatch == len(items) {
			stopDispatch()
		}
	}
	// Stop feeding a cancelled run, unless the caller wants every item dispatched.
	cancelCheck := func() {
		if dispatching && !opts.DispatchAfterCancel && ctx.Err() != nil {
			stats.Cancelled = true
			stopDispatch()
		}
	}

	for i := 0; i < workers && dispatching; i++ {
		dispatch()
	}

	pending := make(map[int]Out, workers)
	for r := range results {
		stats.Completed++
		pending[r.index] = r.value
		cancelCheck()

		for committing {
			value, ok := pending[nextCommit]
			if !ok {
				break
			}
			delete(pending, nextCommit)
			keepGoing := commit(nextCommit, value)
			stats.Committed++
			nextCommit++
			if !keepGoing {
				stats.StoppedEarly = true
				committing = false
				stopDispatch()
				cancelWorkers()
				break
			}
			cancelCheck()
			if dispatching {
				dispatch()
			}
		}
	}

	if ctx.Err() != nil {
		stats.Cancelled = true
	}
	return stats
}
