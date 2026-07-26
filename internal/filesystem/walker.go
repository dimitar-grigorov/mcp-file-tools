// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

// Package filesystem holds the shared, containment-checked directory walk.
package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/security"
)

// Action tells Walk what to do next.
type Action uint8

const (
	Continue Action = iota
	SkipDir
	Stop
)

// Entry is one visited entry.
type Entry struct {
	fs.DirEntry
	Path    string // lexical path
	RelPath string // slash-separated, relative to the root
	Depth   int    // 1 for direct children of the root
}

// Options is the shared traversal policy.
type Options struct {
	AllowedDirs []string                                      // from security.ResolveAllowedDirs
	MaxDepth    int                                           // unlimited when <= 0
	OnError     func(path string, depth int, err error) error // nil aborts on a root error, skips deeper ones
}

// Visitor processes one entry and picks the next action.
type Visitor func(Entry) (Action, error)

var errStopped = errors.New("walk stopped")

// Walk traverses root in lexical order without following directory links.
// Regular files inherit their parent's containment, so they cost no syscall.
func Walk(ctx context.Context, root string, opts Options, visit Visitor) error {
	if visit == nil {
		return errors.New("walk: visitor is required")
	}
	if len(opts.AllowedDirs) == 0 {
		return errors.New("walk: resolved allowed directories are required")
	}
	if !security.IsPathSafeResolved(root, opts.AllowedDirs) {
		if _, err := os.Lstat(root); err != nil {
			return err
		}
		return fmt.Errorf("walk: root resolves outside allowed directories: %s", root)
	}
	if err := walkDir(ctx, root, "", 1, opts, visit); err != nil && !errors.Is(err, errStopped) {
		return err
	}
	return nil
}

func walkDir(ctx context.Context, dir, relDir string, depth int, opts Options, visit Visitor) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return handleError(opts, dir, depth-1, err)
	}
	for _, dirEntry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		path := filepath.Join(dir, dirEntry.Name())
		if needsCheck(dirEntry) && !security.IsPathSafeResolved(path, opts.AllowedDirs) {
			continue
		}
		relPath := dirEntry.Name()
		if relDir != "" {
			relPath = relDir + "/" + dirEntry.Name()
		}
		action, err := visit(Entry{DirEntry: dirEntry, Path: path, RelPath: relPath, Depth: depth})
		if err != nil {
			return err
		}
		if action == Stop {
			return errStopped
		}
		if !dirEntry.IsDir() || action == SkipDir {
			continue
		}
		if opts.MaxDepth > 0 && depth >= opts.MaxDepth {
			continue
		}
		if err := walkDir(ctx, path, relPath, depth+1, opts, visit); err != nil {
			return err
		}
	}
	return nil
}

// needsCheck reports whether an entry can point outside its parent directory.
func needsCheck(dirEntry fs.DirEntry) bool {
	return dirEntry.IsDir() || !dirEntry.Type().IsRegular()
}

func handleError(opts Options, path string, depth int, err error) error {
	if opts.OnError != nil {
		return opts.OnError(path, depth, err)
	}
	if depth == 0 {
		return err
	}
	return nil
}
