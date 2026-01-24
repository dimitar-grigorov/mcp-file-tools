package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHandleReadFile_UTF8(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	content := "Hello, World!"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	params := &mcp.CallToolParamsFor[ReadFileInput]{
		Arguments: ReadFileInput{
			Path:     testFile,
			Encoding: "utf-8",
		},
	}

	result, err := h.HandleReadFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	text := extractText(result.Content)
	if text != content {
		t.Errorf("expected %q, got %q", content, text)
	}
}

func TestHandleReadFile_UTF8WithBOM(t *testing.T) {
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

	// Auto-detect (no encoding specified)
	params := &mcp.CallToolParamsFor[ReadFileInput]{
		Arguments: ReadFileInput{
			Path: testFile,
		},
	}

	result, err := h.HandleReadFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "UTF-8 with BOM") {
		t.Errorf("expected BOM detection message, got %q", text)
	}
	if !strings.Contains(text, content) {
		t.Errorf("expected content %q in result", content)
	}
}

func TestHandleReadFile_CP1251(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")

	// CP1251 bytes for "Привет" (Russian "Hello")
	cp1251Bytes := []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2}

	if err := os.WriteFile(testFile, cp1251Bytes, 0644); err != nil {
		t.Fatal(err)
	}

	params := &mcp.CallToolParamsFor[ReadFileInput]{
		Arguments: ReadFileInput{
			Path:     testFile,
			Encoding: "cp1251",
		},
	}

	result, err := h.HandleReadFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "Привет") {
		t.Errorf("expected 'Привет', got %q", text)
	}
}

func TestHandleReadFile_AutoDetectUTF8(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	// Use plain ASCII content - chardet will detect as "Ascii" which we map to UTF-8
	content := "Hello, this is plain ASCII content for testing auto-detection."

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// No encoding specified - should auto-detect
	params := &mcp.CallToolParamsFor[ReadFileInput]{
		Arguments: ReadFileInput{
			Path: testFile,
		},
	}

	result, err := h.HandleReadFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		text := extractText(result.Content)
		t.Errorf("expected success, got error: %s", text)
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "Auto-detected") {
		t.Errorf("expected auto-detection message, got %q", text)
	}
	if !strings.Contains(text, content) {
		t.Errorf("expected content in result")
	}
}

func TestHandleReadFile_InvalidEncoding(t *testing.T) {
	h := NewHandler()
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")

	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	params := &mcp.CallToolParamsFor[ReadFileInput]{
		Arguments: ReadFileInput{
			Path:     testFile,
			Encoding: "invalid-encoding",
		},
	}

	result, err := h.HandleReadFile(context.Background(), nil, params)
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

func TestHandleReadFile_FileNotFound(t *testing.T) {
	h := NewHandler()

	params := &mcp.CallToolParamsFor[ReadFileInput]{
		Arguments: ReadFileInput{
			Path:     "/nonexistent/file.txt",
			Encoding: "utf-8",
		},
	}

	result, err := h.HandleReadFile(context.Background(), nil, params)
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

func TestHandleReadFile_EmptyPath(t *testing.T) {
	h := NewHandler()

	params := &mcp.CallToolParamsFor[ReadFileInput]{
		Arguments: ReadFileInput{
			Path: "",
		},
	}

	result, err := h.HandleReadFile(context.Background(), nil, params)
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

// Helper to extract text from MCP content
func extractText(content []mcp.Content) string {
	for _, c := range content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}
