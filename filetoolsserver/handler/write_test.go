package handler

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHandleWriteFile_UTF8(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()
	testFile := filepath.Join(dir, "output.txt")
	content := "Hello, World!"

	params := &mcp.CallToolParamsFor[WriteFileInput]{
		Arguments: WriteFileInput{
			Path:     testFile,
			Content:  content,
			Encoding: "utf-8",
		},
	}

	result, err := h.HandleWriteFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	// Verify file content
	written, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(written) != content {
		t.Errorf("expected %q, got %q", content, string(written))
	}
}

func TestHandleWriteFile_CP1251(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()
	testFile := filepath.Join(dir, "output.txt")
	content := "Привет" // Russian "Hello"

	params := &mcp.CallToolParamsFor[WriteFileInput]{
		Arguments: WriteFileInput{
			Path:     testFile,
			Content:  content,
			Encoding: "cp1251",
		},
	}

	result, err := h.HandleWriteFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	// Verify CP1251 bytes were written
	written, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	// Expected CP1251 bytes for "Привет"
	expectedCP1251 := []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2}
	if !bytes.Equal(written, expectedCP1251) {
		t.Errorf("expected CP1251 bytes %v, got %v", expectedCP1251, written)
	}
}

func TestHandleWriteFile_WarningUTF8ToCP1251(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()
	testFile := filepath.Join(dir, "output.txt")

	// First write UTF-8 file
	if err := os.WriteFile(testFile, []byte("UTF-8 content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Now write with CP1251 - should warn
	params := &mcp.CallToolParamsFor[WriteFileInput]{
		Arguments: WriteFileInput{
			Path:     testFile,
			Content:  "New content",
			Encoding: "cp1251",
		},
	}

	result, err := h.HandleWriteFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "Warning") {
		t.Errorf("expected warning about UTF-8 to CP1251, got %q", text)
	}
}

func TestHandleWriteFile_InvalidEncoding(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()
	testFile := filepath.Join(dir, "output.txt")

	params := &mcp.CallToolParamsFor[WriteFileInput]{
		Arguments: WriteFileInput{
			Path:     testFile,
			Content:  "test",
			Encoding: "invalid-encoding",
		},
	}

	result, err := h.HandleWriteFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for invalid encoding")
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "unsupported encoding") {
		t.Errorf("expected 'unsupported encoding' message, got %q", text)
	}
}

func TestHandleWriteFile_EmptyPath(t *testing.T) {
	h := NewHandler()

	params := &mcp.CallToolParamsFor[WriteFileInput]{
		Arguments: WriteFileInput{
			Path:    "",
			Content: "test",
		},
	}

	result, err := h.HandleWriteFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for empty path")
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "path is required") {
		t.Errorf("expected 'path is required' message, got %q", text)
	}
}

func TestHandleWriteFile_DefaultEncoding(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()
	testFile := filepath.Join(dir, "output.txt")
	content := "Тест" // Russian "Test"

	// No encoding specified - should use default (cp1251)
	params := &mcp.CallToolParamsFor[WriteFileInput]{
		Arguments: WriteFileInput{
			Path:    testFile,
			Content: content,
		},
	}

	result, err := h.HandleWriteFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "cp1251") {
		t.Errorf("expected default encoding cp1251 in message, got %q", text)
	}
}
