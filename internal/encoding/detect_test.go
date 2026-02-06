package encoding

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// isASCIICompatible checks if charset is UTF-8 compatible (utf-8 or ascii)
func isASCIICompatible(charset string) bool {
	return charset == "utf-8" || charset == "ascii"
}

func TestDetect_BOMs(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		wantCharset string
	}{
		{"UTF-8 BOM", []byte{0xEF, 0xBB, 0xBF, 'H', 'i'}, "utf-8"},
		{"UTF-16 LE BOM", []byte{0xFF, 0xFE, 'H', 0x00}, "utf-16-le"},
		{"UTF-16 BE BOM", []byte{0xFE, 0xFF, 0x00, 'H'}, "utf-16-be"},
		{"UTF-32 LE BOM", []byte{0xFF, 0xFE, 0x00, 0x00, 'H', 0x00, 0x00, 0x00}, "utf-32-le"},
		{"UTF-32 BE BOM", []byte{0x00, 0x00, 0xFE, 0xFF, 0x00, 0x00, 0x00, 'H'}, "utf-32-be"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Detect(tt.data)
			if result.Charset != tt.wantCharset {
				t.Errorf("Charset = %q, want %q", result.Charset, tt.wantCharset)
			}
			if result.Confidence != 100 {
				t.Errorf("Confidence = %d, want 100", result.Confidence)
			}
			if !result.HasBOM {
				t.Error("HasBOM = false, want true")
			}
		})
	}
}

func TestDetect_PlainASCII(t *testing.T) {
	data := []byte("Hello, World!")
	result := Detect(data)

	// chardet returns "ascii" for pure ASCII content
	if !isASCIICompatible(result.Charset) {
		t.Errorf("Charset = %q, want utf-8 or ascii", result.Charset)
	}
	if result.Confidence < 50 {
		t.Errorf("Confidence = %d, want >= 50", result.Confidence)
	}
}

func TestDetect_EmptyData(t *testing.T) {
	result := Detect([]byte{})
	// Empty data is valid UTF-8
	if result.Charset != "utf-8" {
		t.Errorf("Charset = %q, want utf-8", result.Charset)
	}
}

func TestDetectSample_SmallFile(t *testing.T) {
	data := []byte("Hello, World!")
	result, trusted := DetectSample(data)

	if !isASCIICompatible(result.Charset) {
		t.Errorf("Charset = %q, want utf-8 or ascii", result.Charset)
	}
	if !trusted && result.Confidence >= MinConfidenceThreshold {
		t.Errorf("trusted = %v, expected true for confidence %d", trusted, result.Confidence)
	}
}

func TestDetectSample_LargeFile(t *testing.T) {
	// Create a file larger than SmallFileThreshold
	data := bytes.Repeat([]byte("Hello, World! "), SmallFileThreshold/14+1)
	result, _ := DetectSample(data)

	if !isASCIICompatible(result.Charset) {
		t.Errorf("Charset = %q, want utf-8 or ascii", result.Charset)
	}
}

// --- DetectFromFile tests ---

func TestDetectFromFile_SmallFile_SampleMode(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "small.txt")
	content := []byte("Hello, World! This is a small UTF-8 file.")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DetectFromFile(path, "sample")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isASCIICompatible(result.Charset) {
		t.Errorf("Charset = %q, want utf-8 or ascii", result.Charset)
	}
}

func TestDetectFromFile_WithBOM(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "bom.txt")
	// UTF-8 BOM + content
	content := append([]byte{0xEF, 0xBB, 0xBF}, []byte("Hello with BOM")...)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DetectFromFile(path, "sample")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Charset != "utf-8" {
		t.Errorf("Charset = %q, want utf-8", result.Charset)
	}
	if !result.HasBOM {
		t.Error("HasBOM = false, want true")
	}
	if result.Confidence != 100 {
		t.Errorf("Confidence = %d, want 100", result.Confidence)
	}
}

func TestDetectFromFile_CP1251(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "cyrillic.txt")
	// CP1251 bytes for "Привет мир" (Hello world in Russian)
	cp1251Content := []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2, 0x20, 0xEC, 0xE8, 0xF0}
	if err := os.WriteFile(path, cp1251Content, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DetectFromFile(path, "sample")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should detect some Cyrillic encoding
	if result.Charset == "" {
		t.Error("expected non-empty charset for Cyrillic content")
	}
}

func TestDetectFromFile_NonExistent(t *testing.T) {
	_, err := DetectFromFile("/nonexistent/file.txt", "sample")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestDetectFromFile_InvalidMode(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := DetectFromFile(path, "invalid")
	if err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestDetectFromFile_EmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "empty.txt")
	if err := os.WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DetectFromFile(path, "sample")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty file should be detected as UTF-8
	if result.Charset != "utf-8" {
		t.Errorf("Charset = %q, want utf-8 for empty file", result.Charset)
	}
}

// --- Mode-specific tests ---

func TestDetectFromFile_ChunkedMode_SmallFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "small.txt")
	content := []byte("Small file content for chunked mode test.")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DetectFromFile(path, "chunked")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isASCIICompatible(result.Charset) {
		t.Errorf("Charset = %q, want utf-8 or ascii", result.Charset)
	}
}

func TestDetectFromFile_ChunkedMode_WithBOM(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "bom.txt")
	// Create file larger than ChunkSize with BOM
	content := append([]byte{0xEF, 0xBB, 0xBF}, bytes.Repeat([]byte("A"), ChunkSize+1000)...)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DetectFromFile(path, "chunked")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Charset != "utf-8" {
		t.Errorf("Charset = %q, want utf-8", result.Charset)
	}
	if !result.HasBOM {
		t.Error("HasBOM = false, want true")
	}
}

func TestDetectFromFile_ChunkedMode_LargeFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "large.txt")
	// Create file spanning multiple chunks
	content := bytes.Repeat([]byte("Hello, World! "), ChunkSize/14*3)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DetectFromFile(path, "chunked")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isASCIICompatible(result.Charset) {
		t.Errorf("Charset = %q, want utf-8 or ascii", result.Charset)
	}
}

func TestDetectFromFile_FullMode(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "full.txt")
	content := []byte("Content for full mode detection test.")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DetectFromFile(path, "full")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isASCIICompatible(result.Charset) {
		t.Errorf("Charset = %q, want utf-8 or ascii", result.Charset)
	}
}

func TestDetectFromFile_SampleMode_LargeFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "large_sample.txt")
	// Create file larger than SmallFileThreshold to trigger sampling
	content := bytes.Repeat([]byte("Sample content. "), SmallFileThreshold/16+1000)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DetectFromFile(path, "sample")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isASCIICompatible(result.Charset) {
		t.Errorf("Charset = %q, want utf-8 or ascii", result.Charset)
	}
}

func TestDetectFromFile_SampleMode_LargeFileWithBOM(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "large_bom.txt")
	// Create large file with BOM
	content := append([]byte{0xEF, 0xBB, 0xBF}, bytes.Repeat([]byte("X"), SmallFileThreshold+1000)...)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DetectFromFile(path, "sample")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Charset != "utf-8" {
		t.Errorf("Charset = %q, want utf-8", result.Charset)
	}
	if !result.HasBOM {
		t.Error("HasBOM = false, want true")
	}
}

func TestDetectFromFile_UTF16LE_WithBOM(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "utf16le.txt")
	// UTF-16 LE BOM + "Hi" encoded as UTF-16 LE
	content := []byte{0xFF, 0xFE, 'H', 0x00, 'i', 0x00}
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	for _, mode := range []string{"sample", "chunked", "full"} {
		t.Run(mode, func(t *testing.T) {
			result, err := DetectFromFile(path, mode)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Charset != "utf-16-le" {
				t.Errorf("Charset = %q, want utf-16-le", result.Charset)
			}
			if !result.HasBOM {
				t.Error("HasBOM = false, want true")
			}
		})
	}
}

func TestDetectFromFile_UTF16BE_WithBOM(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "utf16be.txt")
	// UTF-16 BE BOM + "Hi" encoded as UTF-16 BE
	content := []byte{0xFE, 0xFF, 0x00, 'H', 0x00, 'i'}
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DetectFromFile(path, "sample")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Charset != "utf-16-be" {
		t.Errorf("Charset = %q, want utf-16-be", result.Charset)
	}
	if !result.HasBOM {
		t.Error("HasBOM = false, want true")
	}
}

func TestDetect_NoEncoding(t *testing.T) {
	// Random binary data that might not have a clear encoding
	data := []byte{0x00, 0x01, 0x02, 0x03, 0x04}
	result := Detect(data)
	// Should either detect something or return empty
	// Just verify it doesn't panic
	_ = result
}

func TestDetectSample_VeryLargeWithMiddleEnd(t *testing.T) {
	// Create data larger than 2*ChunkSize to trigger middle and end sampling
	// Use content that might produce lower confidence to force full sampling
	size := ChunkSize*3 + 1000

	// Mix some CP1251 Cyrillic bytes throughout to get lower initial confidence
	data := make([]byte, size)
	cp1251Pattern := []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2, 0x20} // "Привет " in CP1251
	asciiPattern := []byte("Hello World ")

	// Interleave patterns
	pos := 0
	for pos < size {
		if pos%(len(cp1251Pattern)+len(asciiPattern)) < len(cp1251Pattern) {
			copy(data[pos:], cp1251Pattern)
			pos += len(cp1251Pattern)
		} else {
			copy(data[pos:], asciiPattern)
			pos += len(asciiPattern)
		}
	}

	result, _ := DetectSample(data)
	// Just verify detection completes without error
	if result.Charset == "" {
		// Might be empty for very ambiguous content, that's okay
		t.Log("No encoding detected for ambiguous content")
	}
}

func TestDetectFromFile_SampleMode_LowConfidenceForceFullSampling(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "low_conf.txt")

	// Create large file with CP1251 content that may have lower initial confidence
	size := ChunkSize*3 + 1000
	cp1251Content := bytes.Repeat([]byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2, 0x20, 0xEC, 0xE8, 0xF0, 0x21, 0x20}, size/12+1)
	if err := os.WriteFile(path, cp1251Content[:size], 0644); err != nil {
		t.Fatal(err)
	}

	result, err := DetectFromFile(path, "sample")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should detect some encoding (likely Cyrillic-related)
	if result.Charset == "" {
		t.Log("No encoding detected, this is acceptable for edge cases")
	}
}

