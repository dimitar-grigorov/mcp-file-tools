// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func newLineEndingsFixture(t *testing.T, body string) (string, *Handler) {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return p, NewHandler([]string{dir})
}

func TestManageLineEndings_Detect(t *testing.T) {
	p, h := newLineEndingsFixture(t, "a\r\nb\nc\r\n")
	res, out, err := h.HandleManageLineEndings(context.Background(), nil, ManageLineEndingsInput{
		Path: p, Action: "detect",
	})
	if err != nil || res.IsError {
		t.Fatalf("detect failed: err=%v result=%+v", err, res)
	}
	if out.Style != LineEndingMixed {
		t.Errorf("style = %q, want mixed", out.Style)
	}
	if len(out.InconsistentLines) == 0 {
		t.Error("expected inconsistentLines for a mixed file")
	}
}

func TestManageLineEndings_Convert(t *testing.T) {
	p, h := newLineEndingsFixture(t, "a\r\nb\r\n")
	res, out, err := h.HandleManageLineEndings(context.Background(), nil, ManageLineEndingsInput{
		Path: p, Action: "convert", Style: "lf",
	})
	if err != nil || res.IsError {
		t.Fatalf("convert failed: err=%v result=%+v", err, res)
	}
	if out.Style != "lf" || out.OriginalStyle != "crlf" {
		t.Errorf("got style=%q originalStyle=%q, want lf/crlf", out.Style, out.OriginalStyle)
	}
	if !out.Changed {
		t.Error("expected changed=true")
	}
	if got, _ := os.ReadFile(p); string(got) != "a\nb\n" {
		t.Errorf("file = %q, want %q", got, "a\nb\n")
	}
}

// action="convert" is a no-op when the file already matches.
func TestManageLineEndings_ConvertNoOp(t *testing.T) {
	p, h := newLineEndingsFixture(t, "a\nb\n")
	_, out, _ := h.HandleManageLineEndings(context.Background(), nil, ManageLineEndingsInput{
		Path: p, Action: "convert", Style: "lf",
	})
	if out.Changed {
		t.Error("expected changed=false when the style already matches")
	}
}

func TestManageLineEndings_ConvertRequiresStyle(t *testing.T) {
	p, h := newLineEndingsFixture(t, "a\r\n")
	res, _, err := h.HandleManageLineEndings(context.Background(), nil, ManageLineEndingsInput{
		Path: p, Action: "convert",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("expected an error result when style is missing")
	}
}

func TestManageLineEndings_UnknownAction(t *testing.T) {
	p, h := newLineEndingsFixture(t, "a\r\n")
	res, _, err := h.HandleManageLineEndings(context.Background(), nil, ManageLineEndingsInput{
		Path: p, Action: "normalise",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Error("expected an error result for an unknown action")
	}
}
