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

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/encoding"
)

func TestHandleConvertEncoding_UTF8ToCP1251(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	// UTF-8 content with Cyrillic
	utf8Content := "Привет мир" // "Hello world" in Russian
	os.WriteFile(testFile, []byte(utf8Content), 0644)

	result, output, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: testFile,
		From: "utf-8",
		To:   "cp1251",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if output.SourceEncoding != "utf-8" {
		t.Errorf("expected source encoding utf-8, got %s", output.SourceEncoding)
	}
	if output.TargetEncoding != "cp1251" {
		t.Errorf("expected target encoding cp1251, got %s", output.TargetEncoding)
	}

	// Verify file was converted (CP1251 bytes are different from UTF-8)
	converted, _ := os.ReadFile(testFile)
	if string(converted) == utf8Content {
		t.Error("file content should be different after conversion")
	}
}

func TestHandleConvertEncoding_CP1251ToUTF8(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	// CP1251 bytes for "Привет" (Russian "Hello")
	cp1251Bytes := []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2}
	os.WriteFile(testFile, cp1251Bytes, 0644)

	result, output, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: testFile,
		From: "cp1251",
		To:   "utf-8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if output.TargetEncoding != "utf-8" {
		t.Errorf("expected target encoding utf-8, got %s", output.TargetEncoding)
	}

	// Verify file is now valid UTF-8
	converted, _ := os.ReadFile(testFile)
	expected := "Привет"
	if string(converted) != expected {
		t.Errorf("expected %q, got %q", expected, string(converted))
	}
}

func TestHandleConvertEncoding_WithBackup(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	originalContent := []byte("Привет") // non-ASCII, so the conversion really changes bytes
	os.WriteFile(testFile, originalContent, 0644)

	result, output, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path:   testFile,
		From:   "utf-8",
		To:     "cp1251",
		Backup: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if output.BackupPath == "" {
		t.Error("expected backup path to be set")
	}

	// Verify backup file exists with original content
	backupContent, err := os.ReadFile(output.BackupPath)
	if err != nil {
		t.Errorf("backup file should exist: %v", err)
	}
	if string(backupContent) != string(originalContent) {
		t.Error("backup should contain original content")
	}
}

func TestHandleConvertEncoding_MissingTo(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	result, _, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: testFile,
		From: "utf-8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing 'to' parameter")
	}
}

func TestHandleConvertEncoding_OutsideAllowed(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	result, _, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: "/some/random/file.txt",
		To:   "utf-8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for path outside allowed directories")
	}
}

func TestHandleConvertEncoding_GBKRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	const chinese = "你好，世界" // "Hello, world"
	testFile := filepath.Join(tempDir, "zh.txt")
	if err := os.WriteFile(testFile, []byte(chinese), 0644); err != nil {
		t.Fatal(err)
	}

	// UTF-8 -> GBK
	_, out, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: testFile, From: "utf-8", To: "gbk",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.TargetEncoding != "gbk" {
		t.Errorf("target encoding = %q, want gbk", out.TargetEncoding)
	}
	if encoded, _ := os.ReadFile(testFile); string(encoded) == chinese {
		t.Error("file should differ from UTF-8 after GBK encoding")
	}

	// GBK -> UTF-8 round-trips back to the original text
	if _, _, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: testFile, From: "gbk", To: "utf-8",
	}); err != nil {
		t.Fatal(err)
	}
	if back, _ := os.ReadFile(testFile); string(back) != chinese {
		t.Errorf("round-trip mismatch: got %q, want %q", back, chinese)
	}
}

func TestHandleConvertEncoding_GB2312AliasResolves(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "zh.txt")
	if err := os.WriteFile(testFile, []byte("编码"), 0644); err != nil {
		t.Fatal(err)
	}

	// gb2312 is an alias for gbk; conversion should succeed.
	result, _, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: testFile, From: "utf-8", To: "gb2312",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Error("expected gb2312 alias to resolve, got error")
	}
}

// lowConfidenceSource is bytes chardet cannot place.
func lowConfidenceSource(t *testing.T) []byte {
	t.Helper()
	data := make([]byte, 0, 512)
	for i := range 256 {
		data = append(data, byte(0x80+i%0x60), byte(0xC0+i%0x30))
	}
	if _, trusted := encoding.DetectSample(data); trusted {
		t.Skip("fixture is detected confidently, nothing to test")
	}
	return data
}

func TestHandleConvertEncoding_LowConfidenceNeedsConfirmation(t *testing.T) {
	write := func(t *testing.T) (*Handler, string) {
		t.Helper()
		dir := t.TempDir()
		path := filepath.Join(dir, "unknown.dat")
		if err := os.WriteFile(path, lowConfidenceSource(t), 0644); err != nil {
			t.Fatal(err)
		}
		return NewHandler([]string{dir}), path
	}
	ctx := context.Background()

	t.Run("refuses and names the guess", func(t *testing.T) {
		h, path := write(t)
		before, _ := os.ReadFile(path)

		res, _, err := h.HandleConvertEncoding(ctx, nil, ConvertEncodingInput{Path: path, To: "utf-8"})
		if err != nil {
			t.Fatal(err)
		}
		if !res.IsError {
			t.Fatal("expected a low-confidence conversion to be refused")
		}
		msg := extractTextFromResult(res.Content)
		for _, want := range []string{"confident", "from=", "allowLowConfidence"} {
			if !strings.Contains(msg, want) {
				t.Errorf("error should mention %q, got: %s", want, msg)
			}
		}
		if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
			t.Error("the file was modified despite the refusal")
		}
	})

	t.Run("allowLowConfidence converts", func(t *testing.T) {
		h, path := write(t)
		res, _, err := h.HandleConvertEncoding(ctx, nil, ConvertEncodingInput{
			Path: path, To: "utf-8", AllowLowConfidence: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.IsError {
			t.Errorf("allowLowConfidence should convert, got: %s", extractTextFromResult(res.Content))
		}
	})

	t.Run("explicit from converts", func(t *testing.T) {
		h, path := write(t)
		res, _, err := h.HandleConvertEncoding(ctx, nil, ConvertEncodingInput{
			Path: path, To: "utf-8", From: "cp1251",
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.IsError {
			t.Errorf("an explicit from should convert, got: %s", extractTextFromResult(res.Content))
		}
	})
}
