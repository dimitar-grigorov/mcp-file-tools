// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package filesystem

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/security"
)

func TestWalk_LexicalOrderAndMetadata(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "a-dir"))
	mkdir(t, filepath.Join(root, "c-dir"))
	write(t, filepath.Join(root, "a-dir", "z.txt"))
	write(t, filepath.Join(root, "a-dir", "a.txt"))
	write(t, filepath.Join(root, "b.txt"))
	write(t, filepath.Join(root, "c-dir", "nested.txt"))

	var rels, paths []string
	var depths []int
	err := Walk(context.Background(), root, Options{AllowedDirs: allowed(root)}, func(e Entry) (Action, error) {
		rels = append(rels, e.RelPath)
		paths = append(paths, e.Path)
		depths = append(depths, e.Depth)
		return Continue, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	wantRels := []string{"a-dir", "a-dir/a.txt", "a-dir/z.txt", "b.txt", "c-dir", "c-dir/nested.txt"}
	if !reflect.DeepEqual(rels, wantRels) {
		t.Fatalf("relative paths = %v, want %v", rels, wantRels)
	}
	if want := []int{1, 2, 2, 1, 1, 2}; !reflect.DeepEqual(depths, want) {
		t.Fatalf("depths = %v, want %v", depths, want)
	}
	for i, p := range paths {
		if want := filepath.Join(root, filepath.FromSlash(wantRels[i])); p != want {
			t.Fatalf("path = %q, want %q", p, want)
		}
	}
}

func TestWalk_MaxDepth(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "a", "b", "c"))
	write(t, filepath.Join(root, "a", "b", "c", "deep.txt"))

	var rels []string
	err := Walk(context.Background(), root, Options{AllowedDirs: allowed(root), MaxDepth: 2}, func(e Entry) (Action, error) {
		rels = append(rels, e.RelPath)
		return Continue, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a", "a/b"}; !reflect.DeepEqual(rels, want) {
		t.Fatalf("visited = %v, want %v", rels, want)
	}
}

func TestWalk_SkipDirPrunesSubtree(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "skip", "inner"))
	write(t, filepath.Join(root, "skip", "secret.txt"))
	write(t, filepath.Join(root, "keep.txt"))

	var rels []string
	err := Walk(context.Background(), root, Options{AllowedDirs: allowed(root)}, func(e Entry) (Action, error) {
		rels = append(rels, e.RelPath)
		if e.Name() == "skip" {
			return SkipDir, nil
		}
		return Continue, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"keep.txt", "skip"}; !reflect.DeepEqual(rels, want) {
		t.Fatalf("visited = %v, want %v", rels, want)
	}
}

func TestWalk_StopEndsWalkWithoutError(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "a.txt"))
	write(t, filepath.Join(root, "b.txt"))

	var rels []string
	err := Walk(context.Background(), root, Options{AllowedDirs: allowed(root)}, func(e Entry) (Action, error) {
		rels = append(rels, e.RelPath)
		return Stop, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a.txt"}; !reflect.DeepEqual(rels, want) {
		t.Fatalf("visited = %v, want %v", rels, want)
	}
}

func TestWalk_VisitorErrorPropagates(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "a.txt"))
	sentinel := errors.New("boom")

	err := Walk(context.Background(), root, Options{AllowedDirs: allowed(root)}, func(Entry) (Action, error) {
		return Continue, sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want %v", err, sentinel)
	}
}

func TestWalk_Cancellation(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "a.txt"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var visited int
	err := Walk(ctx, root, Options{AllowedDirs: allowed(root)}, func(Entry) (Action, error) {
		visited++
		return Continue, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if visited != 0 {
		t.Fatalf("visited %d entries after cancellation", visited)
	}
}

func TestWalk_UnsafeDirectoryLinkNotVisitedOrFollowed(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	write(t, filepath.Join(outside, "secret.txt"))
	dirLink(t, outside, filepath.Join(root, "escape"))
	write(t, filepath.Join(root, "keep.txt"))

	var rels []string
	err := Walk(context.Background(), root, Options{AllowedDirs: allowed(root)}, func(e Entry) (Action, error) {
		rels = append(rels, e.RelPath)
		return Continue, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"keep.txt"}; !reflect.DeepEqual(rels, want) {
		t.Fatalf("visited = %v, want the escaping link skipped", rels)
	}
}

func TestWalk_UnsafeFileLinkSkipped(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	write(t, secret)
	if err := os.Symlink(secret, filepath.Join(root, "leak.txt")); err != nil {
		t.Skipf("file symlinks are not supported in this environment: %v", err)
	}
	write(t, filepath.Join(root, "keep.txt"))

	var rels []string
	err := Walk(context.Background(), root, Options{AllowedDirs: allowed(root)}, func(e Entry) (Action, error) {
		rels = append(rels, e.RelPath)
		return Continue, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"keep.txt"}; !reflect.DeepEqual(rels, want) {
		t.Fatalf("visited = %v, want the escaping file link skipped", rels)
	}
}

func TestWalk_SafeInternalLinkVisited(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	write(t, target)
	if err := os.Symlink(target, filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("file symlinks are not supported in this environment: %v", err)
	}

	var rels []string
	err := Walk(context.Background(), root, Options{AllowedDirs: allowed(root)}, func(e Entry) (Action, error) {
		rels = append(rels, e.RelPath)
		return Continue, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"link.txt", "target.txt"}; !reflect.DeepEqual(rels, want) {
		t.Fatalf("visited = %v, want %v", rels, want)
	}
}

func TestWalk_UnsafeRootRejected(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()

	err := Walk(context.Background(), outside, Options{AllowedDirs: allowed(root)}, func(Entry) (Action, error) {
		t.Fatal("visitor must not run for an unsafe root")
		return Continue, nil
	})
	if err == nil {
		t.Fatal("expected an error for a root outside the allowed directories")
	}
}

func TestWalk_MissingRootReturnsStatError(t *testing.T) {
	root := t.TempDir()
	err := Walk(context.Background(), filepath.Join(root, "nope"), Options{AllowedDirs: allowed(root)}, func(Entry) (Action, error) {
		return Continue, nil
	})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error = %v, want os.ErrNotExist", err)
	}
}

func TestWalk_RequiresVisitorAndAllowedDirs(t *testing.T) {
	root := t.TempDir()
	if err := Walk(context.Background(), root, Options{AllowedDirs: allowed(root)}, nil); err == nil {
		t.Fatal("expected an error for a nil visitor")
	}
	if err := Walk(context.Background(), root, Options{}, func(Entry) (Action, error) { return Continue, nil }); err == nil {
		t.Fatal("expected an error for empty allowed directories")
	}
}

func TestWalk_OnErrorSkipsUnreadableSubtree(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "a-dir")
	mkdir(t, dir)
	write(t, filepath.Join(dir, "child.txt"))
	write(t, filepath.Join(root, "b.txt"))

	var rels []string
	var errPath string
	errDepth := -1
	err := Walk(context.Background(), root, Options{
		AllowedDirs: allowed(root),
		OnError: func(path string, depth int, err error) error {
			errPath, errDepth = path, depth
			return nil
		},
	}, func(e Entry) (Action, error) {
		rels = append(rels, e.RelPath)
		if e.RelPath == "a-dir" {
			// Replace the directory with a file so reading it fails.
			if err := os.RemoveAll(e.Path); err != nil {
				t.Fatal(err)
			}
			write(t, e.Path)
		}
		return Continue, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a-dir", "b.txt"}; !reflect.DeepEqual(rels, want) {
		t.Fatalf("visited = %v, want the walk to continue past the error", rels)
	}
	if errPath != dir || errDepth != 1 {
		t.Fatalf("OnError got (%q, %d), want (%q, 1)", errPath, errDepth, dir)
	}
}

func TestWalk_OnErrorCanAbortWalk(t *testing.T) {
	root := t.TempDir()
	mkdir(t, filepath.Join(root, "a-dir"))
	write(t, filepath.Join(root, "b.txt"))
	sentinel := errors.New("abort")

	err := Walk(context.Background(), root, Options{
		AllowedDirs: allowed(root),
		OnError:     func(string, int, error) error { return sentinel },
	}, func(e Entry) (Action, error) {
		if e.RelPath == "a-dir" {
			if err := os.RemoveAll(e.Path); err != nil {
				t.Fatal(err)
			}
			write(t, e.Path)
		}
		return Continue, nil
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("error = %v, want %v", err, sentinel)
	}
}

func allowed(dirs ...string) []string {
	return security.ResolveAllowedDirs(dirs)
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
}

// dirLink creates a directory symlink, falling back to a junction on Windows.
func dirLink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err == nil {
		return
	} else if runtime.GOOS != "windows" {
		t.Skipf("directory symlinks are not supported in this environment: %v", err)
	}
	output, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
	if err != nil {
		t.Skipf("directory junctions are not supported in this environment: %v (%s)", err, output)
	}
}
