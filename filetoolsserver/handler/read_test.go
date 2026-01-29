package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHandleReadTextFile_UTF8(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")
	content := "Hello, World!"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	params := &mcp.CallToolParamsFor[ReadTextFileInput]{
		Arguments: ReadTextFileInput{
			Path:     testFile,
			Encoding: "utf-8",
		},
	}

	result, err := h.HandleReadTextFile(context.Background(), nil, params)
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

func TestHandleReadTextFile_UTF8WithBOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	// UTF-8 BOM + content
	bom := []byte{0xEF, 0xBB, 0xBF}
	content := "Hello with BOM"
	data := append(bom, []byte(content)...)

	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Auto-detect (no encoding specified)
	params := &mcp.CallToolParamsFor[ReadTextFileInput]{
		Arguments: ReadTextFileInput{
			Path: testFile,
		},
	}

	result, err := h.HandleReadTextFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	text := extractText(result.Content)
	// BOM should be stripped, just verify content is correct
	if !strings.Contains(text, content) {
		t.Errorf("expected content %q in result, got %q", content, text)
	}
}

func TestHandleReadTextFile_CP1251(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	// CP1251 bytes for "Привет" (Russian "Hello")
	cp1251Bytes := []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2}

	if err := os.WriteFile(testFile, cp1251Bytes, 0644); err != nil {
		t.Fatal(err)
	}

	params := &mcp.CallToolParamsFor[ReadTextFileInput]{
		Arguments: ReadTextFileInput{
			Path:     testFile,
			Encoding: "cp1251",
		},
	}

	result, err := h.HandleReadTextFile(context.Background(), nil, params)
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

func TestHandleReadTextFile_AutoDetectUTF8(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")
	// Use plain ASCII content - chardet will detect as "Ascii" which we map to UTF-8
	content := "Hello, this is plain ASCII content for testing auto-detection."

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// No encoding specified - should auto-detect
	params := &mcp.CallToolParamsFor[ReadTextFileInput]{
		Arguments: ReadTextFileInput{
			Path: testFile,
		},
	}

	result, err := h.HandleReadTextFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		text := extractText(result.Content)
		t.Errorf("expected success, got error: %s", text)
	}

	text := extractText(result.Content)
	// No auto-detection message anymore - we default to UTF-8 for compatibility
	// Just verify content is correct
	if text != content {
		t.Errorf("expected %q, got %q", content, text)
	}
}

func TestHandleReadTextFile_InvalidEncoding(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	params := &mcp.CallToolParamsFor[ReadTextFileInput]{
		Arguments: ReadTextFileInput{
			Path:     testFile,
			Encoding: "invalid-encoding",
		},
	}

	result, err := h.HandleReadTextFile(context.Background(), nil, params)
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

func TestHandleReadTextFile_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Try to access a file outside allowed directories
	params := &mcp.CallToolParamsFor[ReadTextFileInput]{
		Arguments: ReadTextFileInput{
			Path:     filepath.Join(tempDir, "..", "..", "nonexistent", "file.txt"),
			Encoding: "utf-8",
		},
	}

	result, err := h.HandleReadTextFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for file outside allowed directories")
	}

	text := extractText(result.Content)
	// Path validation happens first, so we get "access denied" not "failed to read file"
	if !strings.Contains(text, "access denied") {
		t.Errorf("expected 'access denied' message, got %q", text)
	}
}

func TestHandleReadTextFile_EmptyPath(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	params := &mcp.CallToolParamsFor[ReadTextFileInput]{
		Arguments: ReadTextFileInput{
			Path: "",
		},
	}

	result, err := h.HandleReadTextFile(context.Background(), nil, params)
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

func TestHandleReadTextFile_Head(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	head := 3
	params := &mcp.CallToolParamsFor[ReadTextFileInput]{
		Arguments: ReadTextFileInput{
			Path: testFile,
			Head: &head,
		},
	}

	result, err := h.HandleReadTextFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		text := extractText(result.Content)
		t.Errorf("expected success, got error: %s", text)
	}

	text := extractText(result.Content)
	expected := "line1\nline2\nline3"
	if text != expected {
		t.Errorf("expected %q, got %q", expected, text)
	}
}

func TestHandleReadTextFile_Tail(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	tail := 2
	params := &mcp.CallToolParamsFor[ReadTextFileInput]{
		Arguments: ReadTextFileInput{
			Path: testFile,
			Tail: &tail,
		},
	}

	result, err := h.HandleReadTextFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		text := extractText(result.Content)
		t.Errorf("expected success, got error: %s", text)
	}

	text := extractText(result.Content)
	expected := "line5\n"
	if text != expected {
		t.Errorf("expected %q, got %q", expected, text)
	}
}

func TestHandleReadTextFile_HeadAndTail(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	head := 1
	tail := 1
	params := &mcp.CallToolParamsFor[ReadTextFileInput]{
		Arguments: ReadTextFileInput{
			Path: testFile,
			Head: &head,
			Tail: &tail,
		},
	}

	result, err := h.HandleReadTextFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error when both head and tail are specified")
	}

	text := extractText(result.Content)
	if !strings.Contains(text, "cannot specify both") {
		t.Errorf("expected 'cannot specify both' message, got %q", text)
	}
}

func TestHandleReadTextFile_HeadCP1251(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	cyrillicText := "Привет\nМир\nГо\n"
	enc, ok := encoding.Get("cp1251")
	if !ok {
		t.Fatal("cp1251 encoding not found")
	}
	encoder := enc.NewEncoder()
	encoded, err := encoder.Bytes([]byte(cyrillicText))
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(testFile, encoded, 0644); err != nil {
		t.Fatal(err)
	}

	head := 2
	params := &mcp.CallToolParamsFor[ReadTextFileInput]{
		Arguments: ReadTextFileInput{
			Path:     testFile,
			Encoding: "cp1251",
			Head:     &head,
		},
	}

	result, err := h.HandleReadTextFile(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		text := extractText(result.Content)
		t.Errorf("expected success, got error: %s", text)
	}

	text := extractText(result.Content)
	expected := "Привет\nМир"
	if text != expected {
		t.Errorf("expected %q, got %q", expected, text)
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
