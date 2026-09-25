// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func hashResultText(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

func TestContentHash_MatchesSHA256Prefix(t *testing.T) {
	data := []byte("Hello World")
	full := sha256.Sum256(data)
	want := hex.EncodeToString(full[:])[:contentHashBytes*2]
	if got := contentHash(data); got != want {
		t.Errorf("contentHash = %q, want %q", got, want)
	}
}

func TestHashesMatch_AcceptsFullAndPrefixedForms(t *testing.T) {
	data := []byte("Hello World")
	short := contentHash(data)
	full := sha256.Sum256(data)
	fullHex := hex.EncodeToString(full[:])

	for _, expected := range []string{short, fullHex, strings.ToUpper(short), "sha256:" + fullHex} {
		if !hashesMatch(expected, short) {
			t.Errorf("hashesMatch(%q, %q) = false, want true", expected, short)
		}
	}
	if hashesMatch(contentHash([]byte("other")), short) {
		t.Error("a different file's hash must not match")
	}
}

// A stale hash must leave the file byte for byte as it was.
func TestHandleEditFile_ExpectedHashStaleLeavesFileUntouched(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello World"), 0644)

	result, _, err := h.HandleEditFile(context.Background(), nil, EditFileInput{
		Path:         testFile,
		Edits:        []EditOperation{{OldText: "World", NewText: "Go"}},
		ExpectedHash: contentHash([]byte("what the model read earlier")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error on a stale expectedHash")
	}
	if msg := hashResultText(result); !strings.Contains(msg, "changed since you read it") {
		t.Errorf("error should name the staleness, got %q", msg)
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "Hello World" {
		t.Errorf("file must be untouched, got %q", content)
	}
}

func TestHandleEditFile_ExpectedHashCurrentEditsAndReturnsNewHash(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello World"), 0644)

	result, output, err := h.HandleEditFile(context.Background(), nil, EditFileInput{
		Path:         testFile,
		Edits:        []EditOperation{{OldText: "World", NewText: "Go"}},
		ExpectedHash: contentHash([]byte("Hello World")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %q", hashResultText(result))
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "Hello Go" {
		t.Errorf("file should be modified, got %q", content)
	}
	if want := "contentHash: " + contentHash(content); !strings.Contains(hashResultText(result), want) {
		t.Errorf("result should carry the post-edit hash %q, got %q", want, hashResultText(result))
	}
	if output.ContentHash != contentHash(content) {
		t.Errorf("structured contentHash = %q, want %q", output.ContentHash, contentHash(content))
	}
}

// A rejected edit must not leave the read-only flag cleared behind it.
func TestHandleEditFile_ExpectedHashCheckedBeforeReadOnlyIsCleared(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello World"), 0644)
	if err := os.Chmod(testFile, 0444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(testFile, 0644) })

	force := true
	result, _, err := h.HandleEditFile(context.Background(), nil, EditFileInput{
		Path:          testFile,
		Edits:         []EditOperation{{OldText: "World", NewText: "Go"}},
		ExpectedHash:  contentHash([]byte("stale")),
		ForceWritable: &force,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error on a stale expectedHash")
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if !isReadOnly(info.Mode()) {
		t.Error("read-only flag must survive a rejected edit")
	}
}

func TestHandleEditFile_ExpectedHashTooShort(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("Hello World"), 0644)

	result, _, err := h.HandleEditFile(context.Background(), nil, EditFileInput{
		Path:         testFile,
		Edits:        []EditOperation{{OldText: "World", NewText: "Go"}},
		ExpectedHash: "abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error on a too-short expectedHash")
	}
}

func TestHandleWriteFile_ExpectedHashStaleLeavesFileUntouched(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("original"), 0644)

	result, _, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{
		Path:         testFile,
		Content:      "replacement",
		ExpectedHash: contentHash([]byte("stale")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error on a stale expectedHash")
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "original" {
		t.Errorf("file must be untouched, got %q", content)
	}
}

func TestHandleWriteFile_ExpectedHashOnMissingFileFails(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	result, _, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{
		Path:         filepath.Join(tempDir, "absent.txt"),
		Content:      "content",
		ExpectedHash: contentHash([]byte("anything")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expectedHash on a missing file should fail rather than create it")
	}
	if msg := hashResultText(result); !strings.Contains(msg, "does not exist") {
		t.Errorf("error should say the file is absent, got %q", msg)
	}
}

func TestHandleWriteFile_ExpectedHashCurrentWritesAndReturnsNewHash(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("original"), 0644)

	_, output, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{
		Path:         testFile,
		Content:      "replacement",
		ExpectedHash: contentHash([]byte("original")),
	})
	if err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "replacement" {
		t.Errorf("file should be rewritten, got %q", content)
	}
	if output.ContentHash != contentHash(content) {
		t.Errorf("contentHash = %q, want %q", output.ContentHash, contentHash(content))
	}
}

func TestHandleWriteFile_NoExpectedHashReportsNoHash(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	_, output, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{Path: testFile, Content: "new"})
	if err != nil {
		t.Fatal(err)
	}
	if output.ContentHash != "" {
		t.Errorf("contentHash should stay off unless expectedHash was used, got %q", output.ContentHash)
	}
}

// The hash is of the bytes on disk, so paging never changes it.
func TestHandleReadTextFile_ContentHashIsOfWholeFile(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	raw := []byte("one\ntwo\nthree\nfour\n")
	os.WriteFile(testFile, raw, 0644)

	_, full, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{Path: testFile})
	if err != nil {
		t.Fatal(err)
	}
	if full.ContentHash != contentHash(raw) {
		t.Errorf("contentHash = %q, want %q", full.ContentHash, contentHash(raw))
	}

	offset, limit := 2, 2
	_, paged, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{Path: testFile, Offset: &offset, Limit: &limit})
	if err != nil {
		t.Fatal(err)
	}
	if paged.ContentHash != full.ContentHash {
		t.Errorf("a paged read must report the whole file's hash: %q vs %q", paged.ContentHash, full.ContentHash)
	}
}

// Raw bytes, not decoded text, so a cp1251 read guards on what is on disk.
func TestHandleReadTextFile_ContentHashRoundTripsThroughEdit(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte{0xcf, 0xf0, 0xe8, 0xe2, 0xe5, 0xf2}, 0644) // "Привет" in cp1251

	_, read, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{Path: testFile, Encoding: "cp1251"})
	if err != nil {
		t.Fatal(err)
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, EditFileInput{
		Path:         testFile,
		Encoding:     "cp1251",
		Edits:        []EditOperation{{OldText: "Привет", NewText: "Здравей"}},
		ExpectedHash: read.ContentHash,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("the hash from read_text_file should satisfy edit_file, got %q", hashResultText(result))
	}
}
