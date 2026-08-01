// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"container/heap"
	"fmt"
	"io/fs"
	"sort"
)

// Sort modes for list_directory and search_files. Base order follows ls: name
// ascending, mtime newest first, size largest first. reverse flips each.
const (
	sortByName  = "name"
	sortByMtime = "mtime"
	sortBySize  = "size"
)

// sortEntry is one result plus the keys it may be ordered on. mtime and size
// stay zero when Info() failed, which puts the entry last in the base order.
type sortEntry struct {
	key   string // the returned string, and the tiebreaker
	value string // what the caller gets back
	mtime int64
	size  int64
}

// statFunc reads the sort keys off a directory entry. Both os.ReadDir entries
// and filesystem.Entry satisfy fs.DirEntry, so one signature covers both callers.
type statFunc func(fs.DirEntry) (mtime, size int64)

// statEntry reads mtime and size. On Windows FindNextFile already carried them,
// so Info() costs no syscall; elsewhere it is an lstat, hence the needsStat gate.
func statEntry(e fs.DirEntry) (int64, int64) {
	info, err := e.Info()
	if err != nil {
		return 0, 0 // keep the entry, sort it last
	}
	return info.ModTime().UnixNano(), info.Size()
}

// resolveSortBy defaults to name and rejects anything unknown.
func resolveSortBy(sortBy string) (string, error) {
	switch sortBy {
	case "":
		return sortByName, nil
	case sortByName, sortByMtime, sortBySize:
		return sortBy, nil
	}
	return "", fmt.Errorf("invalid sortBy %q: use %q, %q or %q", sortBy, sortByName, sortByMtime, sortBySize)
}

// needsStat reports whether the mode reads anything beyond the entry name.
func needsStat(sortBy string) bool { return sortBy != sortByName }

// sortLess reports whether a comes before b in the final output order.
// The name tiebreaker makes it a total order, so reverse is just less(b, a).
func sortLess(sortBy string, reverse bool) func(a, b sortEntry) bool {
	base := func(a, b sortEntry) bool {
		switch sortBy {
		case sortByMtime:
			if a.mtime != b.mtime {
				return a.mtime > b.mtime // newest first
			}
		case sortBySize:
			if a.size != b.size {
				return a.size > b.size // largest first
			}
		}
		return a.key < b.key
	}
	if !reverse {
		return base
	}
	return func(a, b sortEntry) bool { return base(b, a) }
}

// sortEntries orders a fully collected slice in place.
func sortEntries(entries []sortEntry, sortBy string, reverse bool) {
	less := sortLess(sortBy, reverse)
	sort.Slice(entries, func(i, j int) bool { return less(entries[i], entries[j]) })
}

// worstFirst is a heap whose root is the entry closest to being dropped.
type worstFirst struct {
	items []sortEntry
	less  func(a, b sortEntry) bool
}

func (h worstFirst) Len() int           { return len(h.items) }
func (h worstFirst) Less(i, j int) bool { return h.less(h.items[j], h.items[i]) }
func (h worstFirst) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *worstFirst) Push(x any)        { h.items = append(h.items, x.(sortEntry)) }
func (h *worstFirst) Pop() any {
	last := len(h.items) - 1
	item := h.items[last]
	h.items = h.items[:last]
	return item
}

// topN keeps the best limit entries seen, in bounded memory. Sorting after a
// cap would answer "the first N in walk order, sorted" — not the true top N.
type topN struct {
	h     worstFirst
	limit int
	seen  int
}

func newTopN(limit int, sortBy string, reverse bool) *topN {
	return &topN{h: worstFirst{less: sortLess(sortBy, reverse)}, limit: limit}
}

func (t *topN) add(e sortEntry) {
	t.seen++
	if len(t.h.items) < t.limit {
		heap.Push(&t.h, e)
		return
	}
	if t.h.less(e, t.h.items[0]) {
		t.h.items[0] = e
		heap.Fix(&t.h, 0)
	}
}

func (t *topN) truncated() bool { return t.seen > t.limit }

// values returns the kept entries in output order.
func (t *topN) values() []string {
	sort.Slice(t.h.items, func(i, j int) bool { return t.h.less(t.h.items[i], t.h.items[j]) })
	out := make([]string, len(t.h.items))
	for i, e := range t.h.items {
		out[i] = e.value
	}
	return out
}
