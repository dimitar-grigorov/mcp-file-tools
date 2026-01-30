package encoding

import (
	"strings"
	"unicode/utf8"

	"github.com/wlynxg/chardet"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

// Registry maps encoding names to their implementations.
// nil means UTF-8 passthrough.
var Registry = map[string]encoding.Encoding{
	"utf-8":        nil,
	"utf8":         nil,
	"ascii":        nil,
	"cp1251":       charmap.Windows1251,
	"windows-1251": charmap.Windows1251,
}

// Get returns the encoding for the given name.
func Get(name string) (encoding.Encoding, bool) {
	enc, ok := Registry[strings.ToLower(name)]
	return enc, ok
}

// IsUTF8 checks if the encoding name refers to UTF-8.
func IsUTF8(name string) bool {
	lower := strings.ToLower(name)
	return lower == "utf-8" || lower == "utf8" || lower == "ascii"
}

// DetectionResult holds encoding detection result.
type DetectionResult struct {
	Charset    string
	Confidence int
	HasBOM     bool
}

// Detect detects the encoding of the given data.
func Detect(data []byte) DetectionResult {
	// Check UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return DetectionResult{Charset: "utf-8", Confidence: 100, HasBOM: true}
	}

	detected := chardet.Detect(data)
	if detected.Encoding == "" {
		if utf8.Valid(data) {
			return DetectionResult{Charset: "utf-8", Confidence: 80}
		}
		return DetectionResult{}
	}

	return DetectionResult{
		Charset:    strings.ToLower(detected.Encoding),
		Confidence: int(detected.Confidence * 100),
	}
}

// IsValidUTF8 checks if the given bytes are valid UTF-8.
func IsValidUTF8(data []byte) bool {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return true
	}
	return utf8.Valid(data)
}

// List returns supported encodings.
func List() string {
	return "Supported encodings:\n" +
		"- utf-8 (utf8) - Unicode, no conversion\n" +
		"- cp1251 (windows-1251) - Cyrillic"
}

const (
	// ChunkSize is the size of chunks to read for encoding detection
	ChunkSize = 128 * 1024 // 128KB
	// SmallFileThreshold is the max size to read entirely for detection
	SmallFileThreshold = 128 * 1024 // 128KB
	// HighConfidenceThreshold is the confidence level to stop sampling
	HighConfidenceThreshold = 80
	// MinConfidenceThreshold is the minimum confidence to use detected encoding
	MinConfidenceThreshold = 50
)

// DetectFromChunks detects encoding from file data, using chunked sampling for large files.
// For small files (< 64KB), it uses all data.
// For larger files, it samples beginning, middle, and end.
// Returns the detection result and whether the detected encoding should be trusted.
func DetectFromChunks(data []byte) (DetectionResult, bool) {
	fileSize := len(data)

	// For small files, detect on entire content
	if fileSize <= SmallFileThreshold {
		result := Detect(data)
		trusted := result.Confidence >= MinConfidenceThreshold
		return result, trusted
	}

	// For larger files, sample chunks from beginning, middle, and end
	var samples []byte

	// Beginning chunk
	endOfFirst := ChunkSize
	if endOfFirst > fileSize {
		endOfFirst = fileSize
	}
	samples = append(samples, data[:endOfFirst]...)

	// Check beginning chunk first - if high confidence, use it
	result := Detect(samples)
	if result.Confidence >= HighConfidenceThreshold {
		return result, true
	}

	// Middle chunk (if file is large enough)
	if fileSize > ChunkSize*2 {
		midStart := (fileSize - ChunkSize) / 2
		midEnd := midStart + ChunkSize
		if midEnd > fileSize {
			midEnd = fileSize
		}
		samples = append(samples, data[midStart:midEnd]...)
	}

	// End chunk (if file is large enough)
	if fileSize > ChunkSize {
		endStart := fileSize - ChunkSize
		if endStart < 0 {
			endStart = 0
		}
		samples = append(samples, data[endStart:]...)
	}

	// Detect on combined samples
	result = Detect(samples)
	trusted := result.Confidence >= MinConfidenceThreshold
	return result, trusted
}
