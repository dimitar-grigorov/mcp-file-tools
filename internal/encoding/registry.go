package encoding

import (
	"sort"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

type EncodingInfo struct {
	Encoding    encoding.Encoding // nil = UTF-8 passthrough
	DisplayName string
	Aliases     []string
	Description string
}

var encodings = map[string]EncodingInfo{
	"utf-8":    {nil, "UTF-8", []string{"utf8", "ascii"}, "Unicode, no conversion"},
	"utf-16-le": {unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM), "UTF-16 LE", []string{"utf16le", "utf-16le"}, "Unicode UTF-16 Little Endian"},
	"utf-16-be": {unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM), "UTF-16 BE", []string{"utf16be", "utf-16be"}, "Unicode UTF-16 Big Endian"},

	// Cyrillic
	"windows-1251": {charmap.Windows1251, "Windows-1251", []string{"cp1251"}, "Windows Cyrillic"},
	"koi8-r":       {charmap.KOI8R, "KOI8-R", []string{"koi8r"}, "Russian Cyrillic (Unix/Linux)"},
	"koi8-u":       {charmap.KOI8U, "KOI8-U", []string{"koi8u"}, "Ukrainian Cyrillic (Unix/Linux)"},
	"ibm866":       {charmap.CodePage866, "CP866", []string{"cp866", "dos-866"}, "DOS Cyrillic"},
	"iso-8859-5":   {charmap.ISO8859_5, "ISO-8859-5", []string{"iso88595", "cyrillic"}, "ISO Cyrillic"},

	// Western European
	"windows-1252": {charmap.Windows1252, "Windows-1252", []string{"cp1252"}, "Windows Western European"},
	"iso-8859-1":   {charmap.ISO8859_1, "ISO-8859-1", []string{"iso88591", "latin1"}, "Latin-1 Western European"},
	"iso-8859-15":  {charmap.ISO8859_15, "ISO-8859-15", []string{"iso885915", "latin9"}, "Latin-9 Western European (Euro)"},

	// Central European
	"windows-1250": {charmap.Windows1250, "Windows-1250", []string{"cp1250"}, "Windows Central European"},
	"iso-8859-2":   {charmap.ISO8859_2, "ISO-8859-2", []string{"iso88592", "latin2"}, "Latin-2 Central European"},

	// Greek
	"windows-1253": {charmap.Windows1253, "Windows-1253", []string{"cp1253"}, "Windows Greek"},
	"iso-8859-7":   {charmap.ISO8859_7, "ISO-8859-7", []string{"iso88597", "greek"}, "ISO Greek"},

	// Turkish
	"windows-1254": {charmap.Windows1254, "Windows-1254", []string{"cp1254"}, "Windows Turkish"},
	"iso-8859-9":   {charmap.ISO8859_9, "ISO-8859-9", []string{"iso88599", "latin5"}, "Latin-5 Turkish"},

	// Other
	"windows-1255": {charmap.Windows1255, "Windows-1255", []string{"cp1255"}, "Windows Hebrew"},
	"windows-1256": {charmap.Windows1256, "Windows-1256", []string{"cp1256"}, "Windows Arabic"},
	"windows-1257": {charmap.Windows1257, "Windows-1257", []string{"cp1257"}, "Windows Baltic"},
	"windows-1258": {charmap.Windows1258, "Windows-1258", []string{"cp1258"}, "Windows Vietnamese"},
	"windows-874":  {charmap.Windows874, "Windows-874", []string{"cp874", "tis-620"}, "Windows Thai"},
}

// registry maps all names (canonical + aliases) to EncodingInfo for fast lookup.
var registry map[string]*EncodingInfo

func init() {
	registry = make(map[string]*EncodingInfo)
	for canonical, info := range encodings {
		infoCopy := info
		registry[canonical] = &infoCopy
		for _, alias := range info.Aliases {
			registry[alias] = &infoCopy
		}
	}
}

func Get(name string) (encoding.Encoding, bool) {
	info, ok := registry[strings.ToLower(name)]
	if !ok {
		return nil, false
	}
	return info.Encoding, true
}

func IsUTF8(name string) bool {
	lower := strings.ToLower(name)
	return lower == "utf-8" || lower == "utf8" || lower == "ascii"
}

type EncodingListItem struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"displayName"`
	Aliases     []string `json:"aliases"`
	Description string   `json:"description"`
}

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
	sort.Slice(items, func(i, j int) bool {
		return items[i].DisplayName < items[j].DisplayName
	})
	return items
}