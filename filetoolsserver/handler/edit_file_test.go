// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleEditFile_SimpleReplacement(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello World"), 0644)

	input := EditFileInput{
		Path:  testFile,
		Edits: []EditOperation{{OldText: "World", NewText: "Go"}},
	}

	result, output, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if !strings.Contains(output.Diff, "-Hello World") || !strings.Contains(output.Diff, "+Hello Go") {
		t.Errorf("expected diff to show change, got %q", output.Diff)
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "Hello Go" {
		t.Errorf("file should be modified, got %q", content)
	}
}

func TestHandleEditFile_DryRun(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	originalContent := "Hello World"
	os.WriteFile(testFile, []byte(originalContent), 0644)

	input := EditFileInput{
		Path:   testFile,
		Edits:  []EditOperation{{OldText: "World", NewText: "Go"}},
		DryRun: true,
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != originalContent {
		t.Errorf("file should NOT be modified in dry run, got %q", content)
	}
}

func TestHandleEditFile_MultipleEdits(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("foo bar baz"), 0644)

	input := EditFileInput{
		Path: testFile,
		Edits: []EditOperation{
			{OldText: "foo", NewText: "FOO"},
			{OldText: "bar", NewText: "BAR"},
		},
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "FOO BAR baz" {
		t.Errorf("edits should be applied, got %q", content)
	}
}

func TestHandleEditFile_WhitespaceFlexible(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("    indented line"), 0644)

	input := EditFileInput{
		Path:  testFile,
		Edits: []EditOperation{{OldText: "indented line", NewText: "modified line"}},
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success with flexible whitespace matching")
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "    modified line" {
		t.Errorf("indentation should be preserved, got %q", content)
	}
}

func TestHandleEditFile_NoMatch(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello World"), 0644)

	input := EditFileInput{
		Path:  testFile,
		Edits: []EditOperation{{OldText: "Nonexistent", NewText: "New"}},
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Errorf("expected error when oldText not found")
	}
}

func TestHandleEditFile_MultiLine(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3"), 0644)

	input := EditFileInput{
		Path:  testFile,
		Edits: []EditOperation{{OldText: "line1\nline2", NewText: "new1\nnew2"}},
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "new1\nnew2\nline3" {
		t.Errorf("multi-line edit should be applied, got %q", content)
	}
}

func TestHandleEditFile_CRLFPreservation(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	// Write file with CRLF line endings
	os.WriteFile(testFile, []byte("line1\r\nline2\r\nline3"), 0644)

	input := EditFileInput{
		Path:  testFile,
		Edits: []EditOperation{{OldText: "line1\nline2", NewText: "new1\nnew2"}},
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success with CRLF normalization")
	}

	// Verify CRLF line endings are preserved
	content, _ := os.ReadFile(testFile)
	if string(content) != "new1\r\nnew2\r\nline3" {
		t.Errorf("CRLF line endings should be preserved, got %q", content)
	}
}

func TestHandleEditFile_ValidationErrors(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	tests := []struct {
		name  string
		input EditFileInput
	}{
		{"empty path", EditFileInput{Path: "", Edits: []EditOperation{{OldText: "a", NewText: "b"}}}},
		{"empty edits", EditFileInput{Path: filepath.Join(tempDir, "f.txt"), Edits: []EditOperation{}}},
		{"outside allowed", EditFileInput{Path: "/random/path", Edits: []EditOperation{{OldText: "a", NewText: "b"}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := h.HandleEditFile(context.Background(), nil, tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if !result.IsError {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestAdjustRelativeIndent(t *testing.T) {
	tests := []struct {
		name       string
		oldLines   []string
		newLine    string
		lineIndex  int
		baseIndent string
		want       string
	}{
		{
			name:       "zero relative indent",
			oldLines:   []string{"    old content"},
			newLine:    "    new content",
			lineIndex:  0,
			baseIndent: "        ",
			want:       "        new content",
		},
		{
			name:       "positive relative indent",
			oldLines:   []string{"    old content"},
			newLine:    "        new content",
			lineIndex:  0,
			baseIndent: "    ",
			want:       "        new content",
		},
		{
			name:       "negative relative indent",
			oldLines:   []string{"        old content"},
			newLine:    "    new content",
			lineIndex:  0,
			baseIndent: "        ",
			want:       "    new content",
		},
		{
			name:       "negative indent exceeds base",
			oldLines:   []string{"        old content"},
			newLine:    "new content",
			lineIndex:  0,
			baseIndent: "    ",
			want:       "new content",
		},
		{
			name:       "line index out of range",
			oldLines:   []string{},
			newLine:    "    new content",
			lineIndex:  0,
			baseIndent: "        ",
			want:       "    new content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := adjustRelativeIndent(tt.oldLines, tt.newLine, tt.lineIndex, tt.baseIndent)
			if got != tt.want {
				t.Errorf("adjustRelativeIndent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleEditFile_NegativeRelativeIndent(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// File has a block indented at 8 spaces
	original := "func main() {\n        if true {\n            fmt.Println(\"hello\")\n        }\n}\n"
	testFile := filepath.Join(tempDir, "test.go")
	os.WriteFile(testFile, []byte(original), 0644)

	// oldText has 8-space if block, newText dedents the body to match the if level
	input := EditFileInput{
		Path: testFile,
		Edits: []EditOperation{{
			OldText: "        if true {\n            fmt.Println(\"hello\")\n        }",
			NewText: "        if true {\n        fmt.Println(\"hello\")\n        }",
		}},
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Error("expected success")
	}

	content, _ := os.ReadFile(testFile)
	expected := "func main() {\n        if true {\n        fmt.Println(\"hello\")\n        }\n}\n"
	if string(content) != expected {
		t.Errorf("negative indent not applied.\ngot:  %q\nwant: %q", string(content), expected)
	}
}

func TestEditFile_CP1251Encoding(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// CP1251 encoded Cyrillic text: "Невалиден тип." (Invalid type.)
	// In CP1251: Н=0xCD, е=0xE5, в=0xE2, а=0xE0, л=0xEB, и=0xE8, д=0xE4, н=0xED
	cp1251Content := []byte{
		0xCD, 0xE5, 0xE2, 0xE0, 0xEB, 0xE8, 0xE4, 0xE5, 0xED, // Невалиден
		0x20,             // space
		0xF2, 0xE8, 0xEF, // тип
		0x2E, // .
	}

	testFile := filepath.Join(tempDir, "cyrillic.txt")
	if err := os.WriteFile(testFile, cp1251Content, 0644); err != nil {
		t.Fatal(err)
	}

	// Edit using UTF-8 search text (what Claude sends)
	input := EditFileInput{
		Path:     testFile,
		Edits:    []EditOperation{{OldText: "Невалиден тип.", NewText: "Типът е невалиден."}},
		Encoding: "cp1251",
	}

	result, output, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("edit failed: %v", output)
	}

	// Verify the file was modified correctly
	modifiedData, _ := os.ReadFile(testFile)

	// Expected CP1251: "Типът е невалиден." (The type is invalid.)
	expectedCP1251 := []byte{
		0xD2, 0xE8, 0xEF, 0xFA, 0xF2, // Типът
		0x20,                                                 // space
		0xE5,                                                 // е
		0x20,                                                 // space
		0xED, 0xE5, 0xE2, 0xE0, 0xEB, 0xE8, 0xE4, 0xE5, 0xED, // невалиден
		0x2E, // .
	}

	if string(modifiedData) != string(expectedCP1251) {
		t.Errorf("file content mismatch.\ngot bytes: %v\nwant bytes: %v", modifiedData, expectedCP1251)
	}
}

// Regression: a pure-ASCII file gives no signal about its real legacy encoding,
// so edit_file must fall back to MCP_DEFAULT_ENCODING instead of hardcoding utf-8.
func TestEditFile_InconclusiveDetection_UsesConfiguredDefault(t *testing.T) {
	t.Setenv("MCP_DEFAULT_ENCODING", "cp1251")

	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "legacy.pas")
	asciiContent := "unit Main;\n\n// old english comment\n"
	if err := os.WriteFile(testFile, []byte(asciiContent), 0644); err != nil {
		t.Fatal(err)
	}

	// No explicit encoding - mirrors how edit_file is normally called.
	input := EditFileInput{
		Path: testFile,
		Edits: []EditOperation{
			{OldText: "// old english comment", NewText: "// нов коментар"},
		},
	}

	result, output, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("edit failed: %v", output)
	}

	modifiedData, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	// "нов коментар" in CP1251
	expectedCP1251Comment := []byte{0xED, 0xEE, 0xE2, 0x20, 0xEA, 0xEE, 0xEC, 0xE5, 0xED, 0xF2, 0xE0, 0xF0}
	if !bytes.Contains(modifiedData, expectedCP1251Comment) {
		t.Errorf("expected CP1251-encoded comment in file, got bytes: %v", modifiedData)
	}
	if bytes.HasPrefix(modifiedData, []byte{0xEF, 0xBB, 0xBF}) {
		t.Errorf("file should not have gained a UTF-8 BOM: %v", modifiedData)
	}
}

func TestLongestMatchingBlock(t *testing.T) {
	content := "alpha\nbeta\ngamma\ndelta"

	tests := []struct {
		name      string
		oldText   string
		wantLine  int
		wantCount int
	}{
		{"middle block", "beta\ngamma", 1, 2},
		{"whitespace-insensitive", "  beta  \n\tgamma", 1, 2},
		{"single line", "delta", 3, 1},
		{"no match", "zzz\nyyy", -1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, count := longestMatchingBlock(content, tt.oldText)
			if line != tt.wantLine || count != tt.wantCount {
				t.Errorf("longestMatchingBlock = (%d, %d), want (%d, %d)", line, count, tt.wantLine, tt.wantCount)
			}
		})
	}
}

func TestApplyEdits_NoMatchHint(t *testing.T) {
	content := "func main() {\n\tfmt.Println(\"hi\")\n}\n"

	// oldText whose first line is wrong but the rest matches, so a partial block exists.
	_, err := applyEdits(content, []EditOperation{{
		OldText: "func run() {\n\tfmt.Println(\"hi\")",
		NewText: "x",
	}})
	if err == nil {
		t.Fatal("expected an error for non-matching edit")
	}

	msg := err.Error()
	if !strings.Contains(msg, "HINT") {
		t.Errorf("error should include a hint, got: %s", msg)
	}
	if !strings.Contains(msg, "fmt.Println(\"hi\")") {
		t.Errorf("hint should quote the closest matching file content, got: %s", msg)
	}
}

func TestApplyEdits_NoMatchNoHintWhenNothingMatches(t *testing.T) {
	_, err := applyEdits("a\nb\nc\n", []EditOperation{{OldText: "totally\ndifferent", NewText: "x"}})
	if err == nil {
		t.Fatal("expected an error for non-matching edit")
	}
	if strings.Contains(err.Error(), "HINT") {
		t.Errorf("no hint expected when no lines match, got: %s", err.Error())
	}
}

func TestApplyEdits_FuzzyMatch(t *testing.T) {
	threshold := 0.66
	got, err := applyEdits("func f() {\n\t// current comment\n\treturn 1\n}\n", []EditOperation{{
		OldText: "func f() {\n\t// old comment\n\treturn 1", NewText: "func f() {\n\treturn 2", Similarity: &threshold,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "return 2") {
		t.Fatalf("fuzzy edit not applied: %q", got)
	}
}

func TestApplyEdits_FuzzyMissReportsScore(t *testing.T) {
	threshold := 0.8
	_, err := applyEdits("a\nb\nx\nd\n", []EditOperation{{
		OldText: "a\nb\nc\nd", NewText: "changed", Similarity: &threshold,
	}})
	if err == nil || !strings.Contains(err.Error(), "Best candidate scored 0.75, threshold 0.8") {
		t.Fatalf("expected score and threshold, got %v", err)
	}
}

func TestApplyEdits_FuzzyThresholdBoundary(t *testing.T) {
	threshold := 0.75
	got, err := applyEdits("a\nb\nx\nd\n", []EditOperation{{
		OldText: "a\nb\nc\nd", NewText: "changed", Similarity: &threshold,
	}})
	if err != nil || !strings.HasPrefix(got, "changed") {
		t.Fatalf("boundary match failed: %q, %v", got, err)
	}
}

func TestApplyEdits_FuzzyDoesNotSpanExtraLines(t *testing.T) {
	threshold := 0.8
	content := "func wanted() {\n\ta\n\tb\n\tc\n}\n\nfunc other() {\n\ta\n\textra\n\tb\n\tc\n}\n"
	got, err := applyEdits(content, []EditOperation{{
		OldText: "func other() {\n\ta\n\tb\n\tc", NewText: "wrong", Similarity: &threshold,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "func wanted()") || !strings.Contains(got, "\nwrong\n}") {
		t.Fatalf("fuzzy match chose the wrong function: %q", got)
	}
}

func TestHandleEditFile_Patch(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("alpha\nbeta\ngamma\n"), 0644)

	patch := "--- test.txt\n+++ test.txt\n@@ -1,3 +1,3 @@\n alpha\n-beta\n+BETA\n gamma\n"
	result, output, err := h.HandleEditFile(context.Background(), nil, EditFileInput{Path: testFile, Patch: patch})
	if err != nil || result.IsError {
		t.Fatalf("patch failed: %v", err)
	}
	if !strings.Contains(output.Diff, "+BETA") {
		t.Fatalf("missing returned diff: %q", output.Diff)
	}
	content, _ := os.ReadFile(testFile)
	if string(content) != "alpha\nBETA\ngamma\n" {
		t.Fatalf("unexpected content: %q", content)
	}
}

func TestApplyPatch_RoundTripEditFileDiff(t *testing.T) {
	original := "one\ntwo\nthree\n"
	modified := "one\nTWO\nthree\n"
	diff := createUnifiedDiff(original, modified, "test.txt")
	got, err := applyPatch(original, diff)
	if err != nil {
		t.Fatal(err)
	}
	if got != modified {
		t.Fatalf("round trip mismatch: %q", got)
	}
}

func TestApplyPatch_WhitespaceFlexible(t *testing.T) {
	patch := "--- test.go\n+++ test.go\n@@ -1,2 +1,2 @@\n if ok {\n-    old()\n+    new()\n"
	got, err := applyPatch("\tif ok {\n\t\told()\n", patch)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "new()") {
		t.Fatalf("flexible hunk not applied: %q", got)
	}
}

func TestApplyPatch_FailingHunkHasHint(t *testing.T) {
	patch := "--- test.txt\n+++ test.txt\n@@ -1,2 +1,2 @@\n func run() {\n-old\n+new\n"
	_, err := applyPatch("func main() {\nold\n", patch)
	if err == nil || !strings.Contains(err.Error(), "patch hunk 1 failed") || !strings.Contains(err.Error(), "HINT") {
		t.Fatalf("expected hunk number and hint, got %v", err)
	}
}

func TestHandleEditFile_RejectsPatchAndEdits(t *testing.T) {
	h := NewHandler([]string{t.TempDir()})
	result, _, err := h.HandleEditFile(context.Background(), nil, EditFileInput{
		Path: "unused", Patch: "patch", Edits: []EditOperation{{OldText: "a", NewText: "b"}},
	})
	if err != nil || !result.IsError {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestHandleEditFile_PatchDryRun(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("old\n"), 0644)
	patch := "--- test.txt\n+++ test.txt\n@@ -1 +1 @@\n-old\n+new\n"

	result, _, err := h.HandleEditFile(context.Background(), nil, EditFileInput{Path: testFile, Patch: patch, DryRun: true})
	if err != nil || result.IsError {
		t.Fatalf("dry run failed: %v", err)
	}
	content, _ := os.ReadFile(testFile)
	if string(content) != "old\n" {
		t.Fatalf("dry run wrote the file: %q", content)
	}
}

func TestHandleEditFile_PatchCP1251(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "legacy.txt")
	original := []byte{0xCD, 0xE5, 0xE2, 0xE0, 0xEB, 0xE8, 0xE4, 0xE5, 0xED}
	os.WriteFile(testFile, original, 0644)
	patch := "--- legacy.txt\n+++ legacy.txt\n@@ -1 +1 @@\n-Невалиден\n+Валиден\n"

	result, _, err := h.HandleEditFile(context.Background(), nil, EditFileInput{Path: testFile, Patch: patch, Encoding: "cp1251"})
	if err != nil || result.IsError {
		t.Fatalf("CP1251 patch failed: %v", err)
	}
	content, _ := os.ReadFile(testFile)
	expected := []byte{0xC2, 0xE0, 0xEB, 0xE8, 0xE4, 0xE5, 0xED}
	if !bytes.Equal(content, expected) {
		t.Fatalf("CP1251 round trip failed: %v", content)
	}
}

func TestApplyPatch_RejectsMultipleFiles(t *testing.T) {
	patch := "--- a\n+++ a\n@@ -1 +1 @@\n-a\n+A\n--- b\n+++ b\n@@ -1 +1 @@\n-b\n+B\n"
	_, err := applyPatch("a\n", patch)
	if err == nil || !strings.Contains(err.Error(), "multiple files") {
		t.Fatalf("expected multi-file rejection, got %v", err)
	}
}
