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
