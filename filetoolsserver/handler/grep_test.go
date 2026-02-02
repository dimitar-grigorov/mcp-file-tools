package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleGrep_SimpleMatch(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	content := "line one\nline two with pattern\nline three"
	os.WriteFile(testFile, []byte(content), 0644)

	result, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern: "pattern",
		Paths:   []string{testFile},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if output.TotalMatches != 1 {
		t.Errorf("expected 1 match, got %d", output.TotalMatches)
	}
	if output.Matches[0].Line != 2 {
		t.Errorf("expected match on line 2, got %d", output.Matches[0].Line)
	}
}

func TestHandleGrep_CaseInsensitive(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	content := "Hello World\nHELLO WORLD\nhello world"
	os.WriteFile(testFile, []byte(content), 0644)

	caseSensitive := false
	result, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern:       "hello",
		Paths:         []string{testFile},
		CaseSensitive: &caseSensitive,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if output.TotalMatches != 3 {
		t.Errorf("expected 3 matches (case-insensitive), got %d", output.TotalMatches)
	}
}

func TestHandleGrep_WithContext(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	content := "line 1\nline 2\nMATCH\nline 4\nline 5"
	os.WriteFile(testFile, []byte(content), 0644)

	result, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern:       "MATCH",
		Paths:         []string{testFile},
		ContextBefore: 2,
		ContextAfter:  2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if len(output.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(output.Matches))
	}

	match := output.Matches[0]
	if len(match.Before) != 2 {
		t.Errorf("expected 2 lines before, got %d", len(match.Before))
	}
	if len(match.After) != 2 {
		t.Errorf("expected 2 lines after, got %d", len(match.After))
	}
}

func TestHandleGrep_DirectorySearch(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create multiple files
	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("findme here"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("no match"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file3.txt"), []byte("also findme"), 0644)

	result, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern: "findme",
		Paths:   []string{tempDir},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if output.TotalMatches != 2 {
		t.Errorf("expected 2 matches across files, got %d", output.TotalMatches)
	}
	if output.FilesSearched != 3 {
		t.Errorf("expected 3 files searched, got %d", output.FilesSearched)
	}
	if output.FilesMatched != 2 {
		t.Errorf("expected 2 files matched, got %d", output.FilesMatched)
	}
}

func TestHandleGrep_IncludePattern(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	os.WriteFile(filepath.Join(tempDir, "file.txt"), []byte("findme"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file.pas"), []byte("findme"), 0644)

	result, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern: "findme",
		Paths:   []string{tempDir},
		Include: "*.pas",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if output.TotalMatches != 1 {
		t.Errorf("expected 1 match (only .pas), got %d", output.TotalMatches)
	}
}

func TestHandleGrep_ExcludePattern(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	os.WriteFile(filepath.Join(tempDir, "file.txt"), []byte("findme"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file.bak"), []byte("findme"), 0644)

	result, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern: "findme",
		Paths:   []string{tempDir},
		Exclude: "*.bak",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if output.TotalMatches != 1 {
		t.Errorf("expected 1 match (excluded .bak), got %d", output.TotalMatches)
	}
}

func TestHandleGrep_MaxMatches(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create file with many matches
	content := ""
	for i := 0; i < 100; i++ {
		content += "match line\n"
	}
	os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte(content), 0644)

	result, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern:    "match",
		Paths:      []string{tempDir},
		MaxMatches: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if output.TotalMatches != 10 {
		t.Errorf("expected 10 matches (max), got %d", output.TotalMatches)
	}
	if !output.Truncated {
		t.Error("expected truncated to be true")
	}
}

func TestHandleGrep_CP1251Encoding(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")
	// CP1251 bytes for "Привет" (Russian "Hello")
	cp1251Bytes := []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2}
	os.WriteFile(testFile, cp1251Bytes, 0644)

	// Search for Cyrillic pattern (will be auto-detected)
	result, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern: "Привет",
		Paths:   []string{testFile},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	if output.TotalMatches != 1 {
		t.Errorf("expected 1 match with encoding detection, got %d", output.TotalMatches)
	}
}

func TestHandleGrep_InvalidRegex(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	result, _, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern: "[invalid",
		Paths:   []string{tempDir},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for invalid regex")
	}
}

func TestHandleGrep_MissingPattern(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	result, _, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Paths: []string{tempDir},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing pattern")
	}
}

func TestHandleGrep_MissingPaths(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	result, _, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error for missing paths")
	}
}

func TestHandleGrep_SkipsBinaryFiles(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// Create binary file with null bytes
	binaryContent := []byte{0x00, 0x01, 0x02, 'f', 'i', 'n', 'd', 'm', 'e'}
	os.WriteFile(filepath.Join(tempDir, "binary.bin"), binaryContent, 0644)

	// Create text file
	os.WriteFile(filepath.Join(tempDir, "text.txt"), []byte("findme"), 0644)

	result, output, err := h.HandleGrep(context.Background(), nil, GrepInput{
		Pattern: "findme",
		Paths:   []string{tempDir},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success, got error")
	}
	// Should only find in text file, not binary
	if output.TotalMatches != 1 {
		t.Errorf("expected 1 match (skipping binary), got %d", output.TotalMatches)
	}
}
