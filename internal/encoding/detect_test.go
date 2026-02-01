package encoding

import (
	"bytes"
	"testing"
)

// isASCIICompatible checks if charset is UTF-8 compatible (utf-8 or ascii)
func isASCIICompatible(charset string) bool {
	return charset == "utf-8" || charset == "ascii"
}

func TestDetect_UTF8BOM(t *testing.T) {
	// UTF-8 BOM: EF BB BF
	data := []byte{0xEF, 0xBB, 0xBF, 'H', 'e', 'l', 'l', 'o'}
	result := Detect(data)

	if result.Charset != "utf-8" {
		t.Errorf("Charset = %q, want utf-8", result.Charset)
	}
	if result.Confidence != 100 {
		t.Errorf("Confidence = %d, want 100", result.Confidence)
	}
	if !result.HasBOM {
		t.Error("HasBOM = false, want true")
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

func TestIsValidUTF8(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"valid ascii", []byte("Hello"), true},
		{"valid utf8", []byte("Привет"), true},
		{"utf8 bom", []byte{0xEF, 0xBB, 0xBF, 'a'}, true},
		{"invalid sequence", []byte{0xFF, 0xFE}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidUTF8(tt.data); got != tt.want {
				t.Errorf("IsValidUTF8() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectFromChunks_SmallFile(t *testing.T) {
	data := []byte("Hello, World!")
	result, trusted := DetectFromChunks(data)

	if !isASCIICompatible(result.Charset) {
		t.Errorf("Charset = %q, want utf-8 or ascii", result.Charset)
	}
	if !trusted && result.Confidence >= MinConfidenceThreshold {
		t.Errorf("trusted = %v, expected true for confidence %d", trusted, result.Confidence)
	}
}

func TestDetectFromChunks_LargeFile(t *testing.T) {
	// Create a file larger than SmallFileThreshold
	data := bytes.Repeat([]byte("Hello, World! "), SmallFileThreshold/14+1)
	result, _ := DetectFromChunks(data)

	if !isASCIICompatible(result.Charset) {
		t.Errorf("Charset = %q, want utf-8 or ascii", result.Charset)
	}
}

func TestConstants(t *testing.T) {
	if ChunkSize != 128*1024 {
		t.Errorf("ChunkSize = %d, want %d", ChunkSize, 128*1024)
	}
	if SmallFileThreshold != 128*1024 {
		t.Errorf("SmallFileThreshold = %d, want %d", SmallFileThreshold, 128*1024)
	}
	if HighConfidenceThreshold != 80 {
		t.Errorf("HighConfidenceThreshold = %d, want 80", HighConfidenceThreshold)
	}
	if MinConfidenceThreshold != 50 {
		t.Errorf("MinConfidenceThreshold = %d, want 50", MinConfidenceThreshold)
	}
}
