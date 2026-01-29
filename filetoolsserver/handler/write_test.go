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

// Helper to extract text from MCP content
func extractTextFromResultWrite(content []mcp.Content) string {
	for _, c := range content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}

func TestHandleWriteFile_UTF8(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "output.txt")
	content := "Hello, World!"

	input := WriteFileInput{
		Path:     testFile,
		Content:  content,
		Encoding: "utf-8",
	}

	result, output, err := h.HandleWriteFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	if !strings.Contains(strings.ToLower(output.Message), "success") && !strings.Contains(strings.ToLower(output.Message), "wrote") {
		t.Errorf("expected success message, got %q", output.Message)
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
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "output.txt")
	content := "Привет" // Russian "Hello"

	input := WriteFileInput{
		Path:     testFile,
		Content:  content,
		Encoding: "cp1251",
	}

	result, output, err := h.HandleWriteFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	if !strings.Contains(strings.ToLower(output.Message), "success") && !strings.Contains(strings.ToLower(output.Message), "wrote") {
		t.Errorf("expected success message, got %q", output.Message)
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

func TestHandleWriteFile_InvalidEncoding(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "output.txt")

	input := WriteFileInput{
		Path:     testFile,
		Content:  "test",
		Encoding: "invalid-encoding",
	}

	result, _, err := h.HandleWriteFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for invalid encoding")
	}

	text := extractTextFromResultWrite(result.Content)
	if !strings.Contains(text, "unsupported encoding") {
		t.Errorf("expected 'unsupported encoding' message, got %q", text)
	}
}

func TestHandleWriteFile_EmptyPath(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	input := WriteFileInput{
		Path:    "",
		Content: "test",
	}

	result, _, err := h.HandleWriteFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Errorf("expected error for empty path")
	}

	text := extractTextFromResultWrite(result.Content)
	if !strings.Contains(text, "path is required") {
		t.Errorf("expected 'path is required' message, got %q", text)
	}
}

func TestHandleWriteFile_DefaultEncoding(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "output.txt")
	content := "Тест" // Russian "Test"

	// No encoding specified - should use default (cp1251)
	input := WriteFileInput{
		Path:    testFile,
		Content: content,
	}

	result, output, err := h.HandleWriteFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	if !strings.Contains(output.Message, "cp1251") {
		t.Errorf("expected default encoding cp1251 in message, got %q", output.Message)
	}
}
