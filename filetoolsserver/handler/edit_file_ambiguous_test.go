// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const twoForms = `procedure TFormA.Load;
begin
  Caption := 'Настройки';
end;

procedure TFormB.Load;
begin
  Caption := 'Настройки';
end;
`

// Silently editing the first of several identical blocks picks the wrong one
// half the time, and the tool tells the model not to re-read and check.
func TestApplyEdits_AmbiguousIsRejected(t *testing.T) {
	_, _, err := applyEdits(twoForms, []EditOperation{{
		OldText: "  Caption := 'Настройки';",
		NewText: "  Caption := 'Options';",
	}})
	if err == nil {
		t.Fatal("expected an error for an oldText matching twice")
	}
	if !errors.Is(err, ErrEditAmbiguous) {
		t.Fatalf("wrong error: %v", err)
	}
	for _, want := range []string{"2 places", "lines 3, 8", "NOTHING was changed", "replaceAll"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error is missing %q:\n%s", want, err)
		}
	}
}

func TestApplyEdits_ReplaceAllChangesEvery(t *testing.T) {
	got, n, err := applyEdits(twoForms, []EditOperation{{
		OldText:    "  Caption := 'Настройки';",
		NewText:    "  Caption := 'Options';",
		ReplaceAll: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("replacements = %d, want 2", n)
	}
	if strings.Contains(got, "Настройки") {
		t.Errorf("an occurrence survived:\n%s", got)
	}
	if strings.Count(got, "Options") != 2 {
		t.Errorf("want 2 replacements:\n%s", got)
	}
}

// A unique oldText keeps working exactly as before.
func TestApplyEdits_UniqueStillApplies(t *testing.T) {
	got, n, err := applyEdits("a\nb\nc\n", []EditOperation{{OldText: "b", NewText: "B"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "a\nB\nc\n" || n != 1 {
		t.Errorf("got %q n=%d", got, n)
	}
}

// Ambiguity is caught on the whitespace-flexible path too, not just exact matches.
func TestApplyEdits_AmbiguousFlexibleMatch(t *testing.T) {
	content := "if x then\n    DoIt;\nend;\n\nif y then\n\tDoIt;\nend;\n"
	_, _, err := applyEdits(content, []EditOperation{{OldText: "DoIt;", NewText: "DoThat;"}})
	if !errors.Is(err, ErrEditAmbiguous) {
		t.Fatalf("expected ambiguity across differing indentation, got: %v", err)
	}

	got, n, err := applyEdits(content, []EditOperation{{OldText: "DoIt;", NewText: "DoThat;", ReplaceAll: true}})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 || strings.Contains(got, "DoIt;") {
		t.Errorf("n=%d got:\n%s", n, got)
	}
	// Each site keeps its own indentation.
	if !strings.Contains(got, "    DoThat;") || !strings.Contains(got, "\tDoThat;") {
		t.Errorf("indentation not preserved per site:\n%q", got)
	}
}

// The whole path through the tool: error result, file untouched.
func TestHandleEditFile_AmbiguousLeavesFileAlone(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "u.pas")
	if err := os.WriteFile(p, []byte(twoForms), 0644); err != nil {
		t.Fatal(err)
	}
	h := NewHandler([]string{dir})

	res, _, err := h.HandleEditFile(context.Background(), nil, EditFileInput{
		Path:  p,
		Edits: []EditOperation{{OldText: "  Caption := 'Настройки';", NewText: "  Caption := 'Options';"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected an error result")
	}
	after, _ := os.ReadFile(p)
	if string(after) != twoForms {
		t.Errorf("file was modified despite the error:\n%s", string(after))
	}
}

// replaceAll reports the count so the model can pass it on.
func TestHandleEditFile_ReplaceAllReportsCount(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "u.pas")
	if err := os.WriteFile(p, []byte(twoForms), 0644); err != nil {
		t.Fatal(err)
	}
	h := NewHandler([]string{dir})

	res, out, err := h.HandleEditFile(context.Background(), nil, EditFileInput{
		Path: p,
		Edits: []EditOperation{{
			OldText: "  Caption := 'Настройки';", NewText: "  Caption := 'Options';", ReplaceAll: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("unexpected error: %+v", res.Content)
	}
	if out.Replacements != 2 {
		t.Errorf("Replacements = %d, want 2", out.Replacements)
	}
	after, _ := os.ReadFile(p)
	if strings.Contains(string(after), "Настройки") {
		t.Errorf("an occurrence survived:\n%s", string(after))
	}
}
