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

// Helper to extract text from MCP content
func extractTextFromResultRead(content []mcp.Content) string {
	for _, c := range content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func TestHandleReadTextFile_UTF8(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")
	content := "Hello, World!"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	input := ReadTextFileInput{
		Path:     testFile,
		Encoding: "utf-8",
	}

	result, output, err := h.HandleReadTextFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	if output.Content != content {
		t.Errorf("expected %q, got %q", content, output.Content)
	}
}

func TestHandleReadTextFile_CP1251(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	// CP1251 bytes for "Здравей свят!" (Bulgarian "Hello world!")
	// Encode "Здравей свят!" in CP1251 first
	enc, _ := encoding.Get("cp1251")
	encoder := enc.NewEncoder()
	cp1251Bytes, err := encoder.Bytes([]byte("Здравей свят!"))
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(testFile, cp1251Bytes, 0644); err != nil {
		t.Fatal(err)
	}

	input := ReadTextFileInput{
		Path:     testFile,
		Encoding: "cp1251",
	}

	result, output, err := h.HandleReadTextFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	if !strings.Contains(output.Content, "Здравей свят!") {
		t.Errorf("expected 'Здравей свят!', got %q", output.Content)
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
	input := ReadTextFileInput{
		Path: testFile,
	}

	result, output, err := h.HandleReadTextFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	// No auto-detection message anymore - we default to UTF-8 for compatibility
	// Just verify content is correct
	if output.Content != content {
		t.Errorf("expected %q, got %q", content, output.Content)
	}
}

func TestHandleReadTextFile_InvalidEncoding(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ReadTextFileInput{
		Path:     testFile,
		Encoding: "invalid-encoding",
	}

	result, _, err := h.HandleReadTextFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for invalid encoding")
	}

	text := extractTextFromResultRead(result.Content)
	if !strings.Contains(text, "unsupported encoding") {
		t.Errorf("expected 'unsupported encoding' message, got %q", text)
	}
}

func TestHandleReadTextFile_FileNotFound(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Try to access a file outside allowed directories
	input := ReadTextFileInput{
		Path:     filepath.Join(tempDir, "..", "..", "nonexistent", "file.txt"),
		Encoding: "utf-8",
	}

	result, _, err := h.HandleReadTextFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for file outside allowed directories")
	}

	text := extractTextFromResultRead(result.Content)
	// Path validation happens first, so we get "access denied" not "failed to read file"
	if !strings.Contains(text, "access denied") {
		t.Errorf("expected 'access denied' message, got %q", text)
	}
}

func TestHandleReadTextFile_EmptyPath(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	input := ReadTextFileInput{
		Path: "",
	}

	result, _, err := h.HandleReadTextFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for empty path")
	}

	text := extractTextFromResultRead(result.Content)
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
	input := ReadTextFileInput{
		Path: testFile,
		Head: &head,
	}

	result, output, err := h.HandleReadTextFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	expected := "line1\nline2\nline3"
	if output.Content != expected {
		t.Errorf("expected %q, got %q", expected, output.Content)
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
	input := ReadTextFileInput{
		Path: testFile,
		Tail: &tail,
	}

	result, output, err := h.HandleReadTextFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	expected := "line5\n"
	if output.Content != expected {
		t.Errorf("expected %q, got %q", expected, output.Content)
	}
}

func TestHandleReadTextFile_HeadCP1251(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	cyrillicText := "Здравей\nСвят\nГо\n"
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
	input := ReadTextFileInput{
		Path:     testFile,
		Encoding: "cp1251",
		Head:     &head,
	}

	result, output, err := h.HandleReadTextFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	expected := "Здравей\nСвят"
	if output.Content != expected {
		t.Errorf("expected %q, got %q", expected, output.Content)
	}
}
