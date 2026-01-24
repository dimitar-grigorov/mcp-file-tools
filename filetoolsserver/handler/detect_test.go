package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHandleDetectEncoding_UTF8(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	content := "Hello, World! This is UTF-8 text."

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	params := &mcp.CallToolParamsFor[DetectEncodingInput]{
		Arguments: DetectEncodingInput{
			Path: testFile,
		},
	}

	result, err := h.HandleDetectEncoding(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "utf-8") && !strings.Contains(text, "ascii") {
		t.Errorf("expected UTF-8 or ASCII detection, got %q", text)
	}
}

func TestHandleDetectEncoding_UTF8WithBOM(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")

	// UTF-8 BOM + content
	bom := []byte{0xEF, 0xBB, 0xBF}
	content := "Hello with BOM"
	data := append(bom, []byte(content)...)

	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	params := &mcp.CallToolParamsFor[DetectEncodingInput]{
		Arguments: DetectEncodingInput{
			Path: testFile,
		},
	}

	result, err := h.HandleDetectEncoding(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "utf-8") {
		t.Errorf("expected UTF-8 detection, got %q", text)
	}
	if !strings.Contains(text, "BOM") {
		t.Errorf("expected BOM indicator, got %q", text)
	}
}

func TestHandleDetectEncoding_CP1251(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")

	// CP1251 bytes for "Привет" (Russian "Hello")
	cp1251Bytes := []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2}

	if err := os.WriteFile(testFile, cp1251Bytes, 0644); err != nil {
		t.Fatal(err)
	}

	params := &mcp.CallToolParamsFor[DetectEncodingInput]{
		Arguments: DetectEncodingInput{
			Path: testFile,
		},
	}

	result, err := h.HandleDetectEncoding(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	text := extractText(result.Content)
	// Chardet may detect as windows-1251, koi8-r, or iso-8859-5 for Cyrillic
	if !strings.Contains(strings.ToLower(text), "1251") &&
		!strings.Contains(strings.ToLower(text), "koi8") &&
		!strings.Contains(strings.ToLower(text), "iso-8859") {
		t.Errorf("expected Cyrillic encoding detection, got %q", text)
	}
}

func TestHandleDetectEncoding_FileNotFound(t *testing.T) {
	h := NewHandler()

	params := &mcp.CallToolParamsFor[DetectEncodingInput]{
		Arguments: DetectEncodingInput{
			Path: "/nonexistent/file.txt",
		},
	}

	result, err := h.HandleDetectEncoding(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for nonexistent file")
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "failed to read file") {
		t.Errorf("expected 'failed to read file' message, got %q", text)
	}
}

func TestHandleDetectEncoding_EmptyPath(t *testing.T) {
	h := NewHandler()

	params := &mcp.CallToolParamsFor[DetectEncodingInput]{
		Arguments: DetectEncodingInput{
			Path: "",
		},
	}

	result, err := h.HandleDetectEncoding(context.Background(), nil, params)
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
