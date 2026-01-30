package encoding

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/wlynxg/chardet"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

// EncodingInfo holds metadata about a supported encoding.
type EncodingInfo struct {
	Encoding    encoding.Encoding // nil means UTF-8 passthrough
	DisplayName string
	Aliases     []string
	Description string
}

// encodings defines all supported encodings with their metadata.
// The key is the canonical (primary) name.
var encodings = map[string]EncodingInfo{
	// UTF-8 (passthrough - no conversion needed)
	"utf-8": {
		Encoding:    nil,
		DisplayName: "UTF-8",
		Aliases:     []string{"utf8", "ascii"},
		Description: "Unicode, no conversion",
	},

	// Cyrillic encodings
	"windows-1251": {
		Encoding:    charmap.Windows1251,
		DisplayName: "Windows-1251",
		Aliases:     []string{"cp1251"},
		Description: "Windows Cyrillic",
	},
	"koi8-r": {
		Encoding:    charmap.KOI8R,
		DisplayName: "KOI8-R",
		Aliases:     []string{"koi8r"},
		Description: "Russian Cyrillic (Unix/Linux)",
	},
	"koi8-u": {
		Encoding:    charmap.KOI8U,
		DisplayName: "KOI8-U",
		Aliases:     []string{"koi8u"},
		Description: "Ukrainian Cyrillic (Unix/Linux)",
	},
	"ibm866": {
		Encoding:    charmap.CodePage866,
		DisplayName: "CP866",
		Aliases:     []string{"cp866", "dos-866"},
		Description: "DOS Cyrillic",
	},
	"iso-8859-5": {
		Encoding:    charmap.ISO8859_5,
		DisplayName: "ISO-8859-5",
		Aliases:     []string{"iso88595", "cyrillic"},
		Description: "ISO Cyrillic",
	},

	// Western European encodings
	"windows-1252": {
		Encoding:    charmap.Windows1252,
		DisplayName: "Windows-1252",
		Aliases:     []string{"cp1252"},
		Description: "Windows Western European",
	},
	"iso-8859-1": {
		Encoding:    charmap.ISO8859_1,
		DisplayName: "ISO-8859-1",
		Aliases:     []string{"iso88591", "latin1"},
		Description: "Latin-1 Western European",
	},
	"iso-8859-15": {
		Encoding:    charmap.ISO8859_15,
		DisplayName: "ISO-8859-15",
		Aliases:     []string{"iso885915", "latin9"},
		Description: "Latin-9 Western European (Euro)",
	},

	// Central European encodings
	"windows-1250": {
		Encoding:    charmap.Windows1250,
		DisplayName: "Windows-1250",
		Aliases:     []string{"cp1250"},
		Description: "Windows Central European",
	},
	"iso-8859-2": {
		Encoding:    charmap.ISO8859_2,
		DisplayName: "ISO-8859-2",
		Aliases:     []string{"iso88592", "latin2"},
		Description: "Latin-2 Central European",
	},

	// Greek
	"windows-1253": {
		Encoding:    charmap.Windows1253,
		DisplayName: "Windows-1253",
		Aliases:     []string{"cp1253"},
		Description: "Windows Greek",
	},
	"iso-8859-7": {
		Encoding:    charmap.ISO8859_7,
		DisplayName: "ISO-8859-7",
		Aliases:     []string{"iso88597", "greek"},
		Description: "ISO Greek",
	},

	// Turkish
	"windows-1254": {
		Encoding:    charmap.Windows1254,
		DisplayName: "Windows-1254",
		Aliases:     []string{"cp1254"},
		Description: "Windows Turkish",
	},
	"iso-8859-9": {
		Encoding:    charmap.ISO8859_9,
		DisplayName: "ISO-8859-9",
		Aliases:     []string{"iso88599", "latin5"},
		Description: "Latin-5 Turkish",
	},

	// Hebrew
	"windows-1255": {
		Encoding:    charmap.Windows1255,
		DisplayName: "Windows-1255",
		Aliases:     []string{"cp1255"},
		Description: "Windows Hebrew",
	},

	// Arabic
	"windows-1256": {
		Encoding:    charmap.Windows1256,
		DisplayName: "Windows-1256",
		Aliases:     []string{"cp1256"},
		Description: "Windows Arabic",
	},

	// Baltic
	"windows-1257": {
		Encoding:    charmap.Windows1257,
		DisplayName: "Windows-1257",
		Aliases:     []string{"cp1257"},
		Description: "Windows Baltic",
	},

	// Vietnamese
	"windows-1258": {
		Encoding:    charmap.Windows1258,
		DisplayName: "Windows-1258",
		Aliases:     []string{"cp1258"},
		Description: "Windows Vietnamese",
	},

	// Thai
	"windows-874": {
		Encoding:    charmap.Windows874,
		DisplayName: "Windows-874",
		Aliases:     []string{"cp874", "tis-620"},
		Description: "Windows Thai",
	},
}

// registry is the fast lookup map built from encodings.
// Maps all names (canonical + aliases) to their EncodingInfo.
var registry map[string]*EncodingInfo

// canonicalNames maps all names to their canonical name.
var canonicalNames map[string]string

func init() {
	registry = make(map[string]*EncodingInfo)
	canonicalNames = make(map[string]string)

	for canonical, info := range encodings {
		infoCopy := info // Create a copy to get a stable pointer
		registry[canonical] = &infoCopy
		canonicalNames[canonical] = canonical

		for _, alias := range info.Aliases {
			registry[alias] = &infoCopy
			canonicalNames[alias] = canonical
		}
	}
}

// Get returns the encoding for the given name.
func Get(name string) (encoding.Encoding, bool) {
	info, ok := registry[strings.ToLower(name)]
	if !ok {
		return nil, false
	}
	return info.Encoding, true
}

// GetInfo returns the full encoding info for the given name.
func GetInfo(name string) (*EncodingInfo, bool) {
	info, ok := registry[strings.ToLower(name)]
	return info, ok
}

// GetCanonicalName returns the canonical name for an encoding.
func GetCanonicalName(name string) (string, bool) {
	canonical, ok := canonicalNames[strings.ToLower(name)]
	return canonical, ok
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

// ListEncodings returns a structured list of all supported encodings.
func ListEncodings() []EncodingListItem {
	var items []EncodingListItem

	for canonical, info := range encodings {
		items = append(items, EncodingListItem{
			Name:        canonical,
			DisplayName: info.DisplayName,
			Aliases:     info.Aliases,
			Description: info.Description,
		})
	}

	// Sort by display name for consistent output
	sort.Slice(items, func(i, j int) bool {
		return items[i].DisplayName < items[j].DisplayName
	})

	return items
}

// EncodingListItem represents an encoding in the list output.
type EncodingListItem struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Aliases     []string `json:"aliases"`
	Description string   `json:"description"`
}

// List returns a human-readable string of supported encodings (legacy function).
func List() string {
	items := ListEncodings()
	var sb strings.Builder
	sb.WriteString("Supported encodings:\n")

	for _, item := range items {
		aliases := ""
		if len(item.Aliases) > 0 {
			aliases = fmt.Sprintf(" (%s)", strings.Join(item.Aliases, ", "))
		}
		sb.WriteString(fmt.Sprintf("- %s%s - %s\n", item.Name, aliases, item.Description))
	}

	return sb.String()
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
