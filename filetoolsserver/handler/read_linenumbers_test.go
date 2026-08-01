// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReadTextFile_LineNumbers(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	f := filepath.Join(dir, "a.txt")
	os.WriteFile(f, []byte("alpha\nbeta\ngamma\n"), 0644)

	_, out, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{
		Path: f, LineNumbers: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "1\talpha\n2\tbeta\n3\tgamma\n"
	if out.Content != want {
		t.Errorf("Content = %q, want %q", out.Content, want)
	}
}

// A paged read must number from the file's absolute line, not from 1.
func TestReadTextFile_LineNumbersWithOffset(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	f := filepath.Join(dir, "a.txt")
	os.WriteFile(f, []byte("l1\nl2\nl3\nl4\nl5\n"), 0644)

	off, lim := 3, 2
	_, out, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{
		Path: f, LineNumbers: true, Offset: &off, Limit: &lim,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "3\tl3\n4\tl4\n"
	if out.Content != want {
		t.Errorf("Content = %q, want %q", out.Content, want)
	}
}

func TestReadTextFile_LineNumbersOffByDefault(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	f := filepath.Join(dir, "a.txt")
	os.WriteFile(f, []byte("alpha\n"), 0644)

	_, out, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{Path: f})
	if err != nil {
		t.Fatal(err)
	}
	if out.Content != "alpha\n" {
		t.Errorf("Content = %q, want unprefixed", out.Content)
	}
}

func TestAddLineNumbers_NoTrailingNewline(t *testing.T) {
	if got := addLineNumbers("a\nb", 1); got != "1\ta\n2\tb" {
		t.Errorf("got %q", got)
	}
	if got := addLineNumbers("", 5); got != "" {
		t.Errorf("got %q for empty content", got)
	}
}
