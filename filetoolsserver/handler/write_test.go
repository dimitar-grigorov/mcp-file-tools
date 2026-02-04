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

func TestHandleWriteFile_DefaultEncoding_NewFile(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "new_file.txt")
	content := "Тест" // Russian "Test"

	// No encoding specified for NEW file - should use configured default (cp1251)
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

	// Verify CP1251 bytes were written
	written, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	// Expected CP1251 bytes for "Тест"
	expectedCP1251 := []byte{0xD2, 0xE5, 0xF1, 0xF2}
	if !bytes.Equal(written, expectedCP1251) {
		t.Errorf("expected CP1251 bytes %v, got %v", expectedCP1251, written)
	}
}

func TestHandleWriteFile_PreservesExistingEncoding(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "existing.txt")

	// Create an existing file with UTF-8 content
	utf8Content := "Hello, мир!" // Mixed English and Russian in UTF-8
	if err := os.WriteFile(testFile, []byte(utf8Content), 0644); err != nil {
		t.Fatal(err)
	}

	// Write new content WITHOUT specifying encoding - should preserve UTF-8
	newContent := "Goodbye, мир!"
	input := WriteFileInput{
		Path:    testFile,
		Content: newContent,
	}

	result, output, err := h.HandleWriteFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		text := extractTextFromResultWrite(result.Content)
		t.Errorf("expected success, got error: %s", text)
	}

	// Should preserve UTF-8 encoding
	if !strings.Contains(output.Message, "utf-8") && !strings.Contains(output.Message, "UTF-8") {
		t.Errorf("expected preserved UTF-8 encoding in message, got %q", output.Message)
	}

	// Verify UTF-8 bytes were written (not CP1251)
	written, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(written) != newContent {
		t.Errorf("expected UTF-8 content %q, got %q", newContent, string(written))
	}
}

func TestHandleWriteFile_PreservesExistingCP1251(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "cyrillic.txt")

	// Create an existing file with CP1251 content (Russian text)
	// "Привет мир" in CP1251
	cp1251Content := []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2, 0x20, 0xEC, 0xE8, 0xF0}
	if err := os.WriteFile(testFile, cp1251Content, 0644); err != nil {
		t.Fatal(err)
	}

	// Write new content WITHOUT specifying encoding - should preserve CP1251
	newContent := "Пока" // "Bye" in Russian
	input := WriteFileInput{
		Path:    testFile,
		Content: newContent,
	}

	result, output, err := h.HandleWriteFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		text := extractTextFromResultWrite(result.Content)
		t.Errorf("expected success, got error: %s", text)
	}

	// Should preserve CP1251 encoding (may be detected as "windows-1251" alias)
	if !strings.Contains(output.Message, "cp1251") && !strings.Contains(output.Message, "windows-1251") {
		t.Errorf("expected preserved CP1251/windows-1251 encoding in message, got %q", output.Message)
	}

	// Verify CP1251 bytes were written
	written, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	// Expected CP1251 bytes for "Пока"
	expectedCP1251 := []byte{0xCF, 0xEE, 0xEA, 0xE0}
	if !bytes.Equal(written, expectedCP1251) {
		t.Errorf("expected CP1251 bytes %v, got %v", expectedCP1251, written)
	}
}

func TestHandleWriteFile_ExplicitEncodingOverridesExisting(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "override.txt")

	// Create an existing file with UTF-8 content
	if err := os.WriteFile(testFile, []byte("Hello UTF-8"), 0644); err != nil {
		t.Fatal(err)
	}

	// Write new content WITH explicit CP1251 encoding - should use CP1251, not UTF-8
	newContent := "Тест" // Russian "Test"
	input := WriteFileInput{
		Path:     testFile,
		Content:  newContent,
		Encoding: "cp1251", // Explicit override
	}

	result, output, err := h.HandleWriteFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		text := extractTextFromResultWrite(result.Content)
		t.Errorf("expected success, got error: %s", text)
	}

	// Should use explicit CP1251
	if !strings.Contains(output.Message, "cp1251") {
		t.Errorf("expected explicit CP1251 encoding in message, got %q", output.Message)
	}

	// Verify CP1251 bytes were written
	written, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	// Expected CP1251 bytes for "Тест"
	expectedCP1251 := []byte{0xD2, 0xE5, 0xF1, 0xF2}
	if !bytes.Equal(written, expectedCP1251) {
		t.Errorf("expected CP1251 bytes %v, got %v", expectedCP1251, written)
	}
}
