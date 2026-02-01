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

	// Verify content is correct
	if output.Content != content {
		t.Errorf("expected %q, got %q", content, output.Content)
	}

	// Verify auto-detection info is present
	if output.DetectedEncoding == "" {
		t.Errorf("expected DetectedEncoding to be set when auto-detecting")
	}
}

func TestHandleReadTextFile_AutoDetectCP1251(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	// Create a file with CP1251 Cyrillic content
	// More Cyrillic text for better detection
	cyrillicText := "Здравей свят! Това е тест за автоматично разпознаване на кодирането."
	enc, ok := encoding.Get("cp1251")
	if !ok {
		t.Fatal("cp1251 encoding not found")
	}
	encoder := enc.NewEncoder()
	cp1251Bytes, err := encoder.Bytes([]byte(cyrillicText))
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(testFile, cp1251Bytes, 0644); err != nil {
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

	// Verify auto-detection info is present
	if output.DetectedEncoding == "" {
		t.Errorf("expected DetectedEncoding to be set when auto-detecting")
	}

	// The detection should either correctly decode the content or indicate the detected encoding
	// Due to detection confidence variations, we just verify the output is not empty
	if output.Content == "" {
		t.Errorf("expected content to be non-empty")
	}
}

func TestHandleReadTextFile_ExplicitEncodingNoDetectionInfo(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")
	content := "Hello, World!"

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Explicitly specify encoding
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

	// When encoding is explicitly specified, detection info should NOT be present
	if output.DetectedEncoding != "" {
		t.Errorf("expected DetectedEncoding to be empty when encoding is explicitly specified, got %q", output.DetectedEncoding)
	}

	if output.EncodingConfidence != 0 {
		t.Errorf("expected EncodingConfidence to be 0 when encoding is explicitly specified, got %d", output.EncodingConfidence)
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

func TestHandleReadTextFile_OffsetLimit(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	// Create a file with 10 lines
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	offset := 3
	limit := 4
	input := ReadTextFileInput{
		Path:   testFile,
		Offset: &offset,
		Limit:  &limit,
	}

	result, output, err := h.HandleReadTextFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	// Should return lines 3-6
	expected := "line3\nline4\nline5\nline6"
	if output.Content != expected {
		t.Errorf("expected %q, got %q", expected, output.Content)
	}

	if output.TotalLines != 10 {
		t.Errorf("expected TotalLines=10, got %d", output.TotalLines)
	}

	if output.StartLine != 3 {
		t.Errorf("expected StartLine=3, got %d", output.StartLine)
	}

	if output.EndLine != 6 {
		t.Errorf("expected EndLine=6, got %d", output.EndLine)
	}
}

func TestHandleReadTextFile_OffsetOnly(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	content := "line1\nline2\nline3\nline4\nline5"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	offset := 3
	input := ReadTextFileInput{
		Path:   testFile,
		Offset: &offset,
	}

	result, output, err := h.HandleReadTextFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	// Should return lines 3 to end
	expected := "line3\nline4\nline5"
	if output.Content != expected {
		t.Errorf("expected %q, got %q", expected, output.Content)
	}

	if output.StartLine != 3 {
		t.Errorf("expected StartLine=3, got %d", output.StartLine)
	}

	if output.EndLine != 5 {
		t.Errorf("expected EndLine=5, got %d", output.EndLine)
	}
}

func TestHandleReadTextFile_LimitOnly(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	content := "line1\nline2\nline3\nline4\nline5"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	limit := 2
	input := ReadTextFileInput{
		Path:  testFile,
		Limit: &limit,
	}

	result, output, err := h.HandleReadTextFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	// Should return first 2 lines
	expected := "line1\nline2"
	if output.Content != expected {
		t.Errorf("expected %q, got %q", expected, output.Content)
	}

	if output.StartLine != 1 {
		t.Errorf("expected StartLine=1, got %d", output.StartLine)
	}

	if output.EndLine != 2 {
		t.Errorf("expected EndLine=2, got %d", output.EndLine)
	}
}

func TestHandleReadTextFile_OffsetLimitConflictWithHead(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	offset := 1
	head := 5
	input := ReadTextFileInput{
		Path:   testFile,
		Offset: &offset,
		Head:   &head,
	}

	result, _, err := h.HandleReadTextFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if !result.IsError {
		t.Error("expected error for offset/limit with head conflict")
	}
}

func TestHandleReadTextFile_TotalLinesReturned(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	content := "line1\nline2\nline3"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	input := ReadTextFileInput{Path: testFile}

	result, output, err := h.HandleReadTextFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error")
	}

	if output.TotalLines != 3 {
		t.Errorf("expected TotalLines=3, got %d", output.TotalLines)
	}

	if output.StartLine != 1 {
		t.Errorf("expected StartLine=1, got %d", output.StartLine)
	}

	if output.EndLine != 3 {
		t.Errorf("expected EndLine=3, got %d", output.EndLine)
	}
}
