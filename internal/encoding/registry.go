package encoding

import (
	"sort"
	"strings"

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

func init() {
	registry = make(map[string]*EncodingInfo)

	for canonical, info := range encodings {
		infoCopy := info // Create a copy to get a stable pointer
		registry[canonical] = &infoCopy

		for _, alias := range info.Aliases {
			registry[alias] = &infoCopy
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

// IsUTF8 checks if the encoding name refers to UTF-8.
func IsUTF8(name string) bool {
	lower := strings.ToLower(name)
	return lower == "utf-8" || lower == "utf8" || lower == "ascii"
}

// EncodingListItem represents an encoding in the list output.
type EncodingListItem struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Aliases     []string `json:"aliases"`
	Description string   `json:"description"`
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