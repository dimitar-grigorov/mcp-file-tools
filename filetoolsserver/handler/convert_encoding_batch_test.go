// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeUTF8 creates a UTF-8 file and returns its path.
func writeUTF8(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestConvertEncoding_DryRunLeavesFileAlone(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	f := writeUTF8(t, dir, "a.txt", "Привет мир")
	before, _ := os.ReadFile(f)

	result, output, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path:   f,
		From:   "utf-8",
		To:     "cp1251",
		DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if !output.DryRun {
		t.Error("DryRun = false, want true")
	}
	if !output.Changed {
		t.Error("Changed = false, want true (it would convert)")
	}
	after, _ := os.ReadFile(f)
	if string(after) != string(before) {
		t.Error("dry run modified the file")
	}
	if !strings.Contains(output.Message, "Would convert") {
		t.Errorf("Message = %q, want it to say it would convert", output.Message)
	}
}

func TestConvertEncoding_DryRunSkipsBackup(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	f := writeUTF8(t, dir, "a.txt", "Привет")

	_, _, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: f, From: "utf-8", To: "cp1251", Backup: true, DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f + ".bak"); err == nil {
		t.Error("dry run created a backup file")
	}
}

func TestConvertEncoding_BatchConvertsAll(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	a := writeUTF8(t, dir, "a.txt", "Привет")
	b := writeUTF8(t, dir, "b.txt", "мир")

	result, output, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Paths: []string{a, b}, From: "utf-8", To: "cp1251",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if len(output.Results) != 2 {
		t.Fatalf("len(Results) = %d, want 2", len(output.Results))
	}
	if output.ErrorCount != 0 {
		t.Errorf("ErrorCount = %d, want 0 (%v)", output.ErrorCount, output.Errors)
	}
	if output.SuccessCount != 2 {
		t.Errorf("SuccessCount = %d, want 2", output.SuccessCount)
	}
	for _, r := range output.Results {
		if !r.Changed {
			t.Errorf("%s: Changed = false, want true", r.Path)
		}
	}
}

// A file that cannot be encoded must not abort the rest of the batch.
func TestConvertEncoding_BatchContinuesPastFailure(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	good := writeUTF8(t, dir, "good.txt", "Привет")
	bad := writeUTF8(t, dir, "bad.txt", "Bäcker Grüße") // umlauts are not in cp1251

	_, output, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Paths: []string{bad, good}, From: "utf-8", To: "cp1251",
	})
	if err != nil {
		t.Fatal(err)
	}
	if output.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", output.ErrorCount)
	}
	if output.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", output.SuccessCount)
	}

	var badResult *ConvertFileResult
	for i := range output.Results {
		if output.Results[i].Path == bad {
			badResult = &output.Results[i]
		}
	}
	if badResult == nil {
		t.Fatal("no result for the failing file")
	}
	// The whole point: the offenders come back as data, not just prose.
	if badResult.UnsupportedCount != 3 {
		t.Errorf("UnsupportedCount = %d, want 3 (ä, ü, ß)", badResult.UnsupportedCount)
	}
	if len(badResult.Unsupported) == 0 {
		t.Fatal("Unsupported is empty, want the offending runes")
	}
	if badResult.Unsupported[0].Char != "ä" {
		t.Errorf("first offender = %q, want ä", badResult.Unsupported[0].Char)
	}
	if badResult.Unsupported[0].Line != 1 {
		t.Errorf("first offender line = %d, want 1", badResult.Unsupported[0].Line)
	}

	// The good file must still be on disk converted.
	if got, _ := os.ReadFile(good); string(got) == "Привет" {
		t.Error("good file was not converted")
	}
}

func TestConvertEncoding_DryRunBatchReportsWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	a := writeUTF8(t, dir, "a.txt", "Привет")
	bad := writeUTF8(t, dir, "bad.txt", "Straße")
	beforeA, _ := os.ReadFile(a)

	_, output, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Paths: []string{a, bad}, From: "utf-8", To: "cp1251", DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if output.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", output.ErrorCount)
	}
	if !strings.Contains(output.Message, "would convert") {
		t.Errorf("Message = %q, want 'would convert'", output.Message)
	}
	if afterA, _ := os.ReadFile(a); string(afterA) != string(beforeA) {
		t.Error("dry run modified a file")
	}
}

func TestConvertEncoding_BatchCountsUnchanged(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	// Already utf-8, so converting to utf-8 is a no-op.
	a := writeUTF8(t, dir, "a.txt", "plain ascii")

	_, output, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Paths: []string{a}, To: "utf-8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if output.Results[0].Changed {
		t.Error("Changed = true, want false for a no-op")
	}
	if !strings.Contains(output.Message, "already utf-8") {
		t.Errorf("Message = %q, want it to mention the file is already utf-8", output.Message)
	}
}

func TestConvertEncoding_RejectsPathAndPathsTogether(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	f := writeUTF8(t, dir, "a.txt", "x")

	result, _, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: f, Paths: []string{f}, To: "utf-8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected an error when both path and paths are given")
	}
}

func TestConvertEncoding_RejectsNeitherPathNorPaths(t *testing.T) {
	h := NewHandler([]string{t.TempDir()})
	result, _, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{To: "utf-8"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected an error when no path is given")
	}
}

// A single path must keep failing as a tool error, not silently as a result entry.
func TestConvertEncoding_SinglePathStillErrorsHard(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	f := writeUTF8(t, dir, "bad.txt", "Straße")

	result, _, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: f, From: "utf-8", To: "cp1251",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected a tool error for a lossy single-file conversion")
	}
	text := extractTextFromResult(result.Content)
	for _, want := range []string{"ß", "U+00DF", "line 1", "utf-8"} {
		if !strings.Contains(text, want) {
			t.Errorf("error text = %q, missing %q", text, want)
		}
	}
}
