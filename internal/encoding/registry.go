// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"sort"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
)

type encodingInfo struct {
	Encoding    encoding.Encoding // nil = UTF-8 passthrough
	DisplayName string
	Aliases     []string
	Description string

	// DetectorLabels are the detector's spellings for this encoding; empty means detection may never answer it.
	DetectorLabels []string
}

// Two questions, one table: what this server can decode, and what detection may name. Labels living on the codec entry keep the second a subset of the first.
var encodings = map[string]encodingInfo{
	"utf-8": {
		Encoding:       nil, // UTF-8 passthrough
		DisplayName:    "UTF-8",
		Aliases:        []string{"utf8", "ascii"},
		Description:    "Unicode, no conversion",
		DetectorLabels: []string{"utf-8", "utf-8-sig"},
	},
	"utf-16-le": {
		Encoding:    unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),
		DisplayName: "UTF-16 LE",
		Aliases:     []string{"utf16le", "utf-16le"},
		Description: "Unicode UTF-16 Little Endian",
	},
	"utf-16-be": {
		Encoding:    unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM),
		DisplayName: "UTF-16 BE",
		Aliases:     []string{"utf16be", "utf-16be"},
		Description: "Unicode UTF-16 Big Endian",
	},

	// Cyrillic
	"windows-1251": {
		Encoding:       charmap.Windows1251,
		DisplayName:    "Windows-1251",
		Aliases:        []string{"cp1251"},
		Description:    "Windows Cyrillic",
		DetectorLabels: []string{"windows-1251"},
	},
	"koi8-r": {
		Encoding:       charmap.KOI8R,
		DisplayName:    "KOI8-R",
		Aliases:        []string{"koi8r"},
		Description:    "Russian Cyrillic (Unix/Linux)",
		DetectorLabels: []string{"koi8-r"},
	},
	"koi8-u": {
		Encoding:    charmap.KOI8U,
		DisplayName: "KOI8-U",
		Aliases:     []string{"koi8u"},
		Description: "Ukrainian Cyrillic (Unix/Linux)",
	},
	"ibm866": {
		Encoding:       charmap.CodePage866,
		DisplayName:    "CP866",
		Aliases:        []string{"cp866", "dos-866"},
		Description:    "DOS Cyrillic",
		DetectorLabels: []string{"ibm866"},
	},
	"iso-8859-5": {
		Encoding:       charmap.ISO8859_5,
		DisplayName:    "ISO-8859-5",
		Aliases:        []string{"iso88595", "cyrillic"},
		Description:    "ISO Cyrillic",
		DetectorLabels: []string{"iso-8859-5"},
	},
	"x-mac-cyrillic": {
		Encoding:       charmap.MacintoshCyrillic,
		DisplayName:    "MacCyrillic",
		Aliases:        []string{"maccyrillic", "mac-cyrillic"},
		Description:    "Macintosh Cyrillic",
		DetectorLabels: []string{"maccyrillic"},
	},

	// Western European
	"windows-1252": {
		Encoding:       charmap.Windows1252,
		DisplayName:    "Windows-1252",
		Aliases:        []string{"cp1252"},
		Description:    "Windows Western European",
		DetectorLabels: []string{"windows-1252"},
	},
	"iso-8859-1": {
		Encoding:       charmap.ISO8859_1,
		DisplayName:    "ISO-8859-1",
		Aliases:        []string{"iso88591", "latin1"},
		Description:    "Latin-1 Western European",
		DetectorLabels: []string{"iso-8859-1"},
	},
	"iso-8859-15": {
		Encoding:    charmap.ISO8859_15,
		DisplayName: "ISO-8859-15",
		Aliases:     []string{"iso885915", "latin9"},
		Description: "Latin-9 Western European (Euro)",
	},

	// Central European
	"windows-1250": {
		Encoding:       charmap.Windows1250,
		DisplayName:    "Windows-1250",
		Aliases:        []string{"cp1250"},
		Description:    "Windows Central European",
		DetectorLabels: []string{"windows-1250"},
	},
	"iso-8859-2": {
		Encoding:       charmap.ISO8859_2,
		DisplayName:    "ISO-8859-2",
		Aliases:        []string{"iso88592", "latin2"},
		Description:    "Latin-2 Central European",
		DetectorLabels: []string{"iso-8859-2"},
	},

	// Greek
	"windows-1253": {
		Encoding:       charmap.Windows1253,
		DisplayName:    "Windows-1253",
		Aliases:        []string{"cp1253"},
		Description:    "Windows Greek",
		DetectorLabels: []string{"windows-1253"},
	},
	"iso-8859-7": {
		Encoding:       charmap.ISO8859_7,
		DisplayName:    "ISO-8859-7",
		Aliases:        []string{"iso88597", "greek"},
		Description:    "ISO Greek",
		DetectorLabels: []string{"iso-8859-7"},
	},

	// Turkish
	"windows-1254": {
		Encoding:       charmap.Windows1254,
		DisplayName:    "Windows-1254",
		Aliases:        []string{"cp1254"},
		Description:    "Windows Turkish",
		DetectorLabels: []string{"windows-1254"},
	},
	"iso-8859-9": {
		Encoding:       charmap.ISO8859_9,
		DisplayName:    "ISO-8859-9",
		Aliases:        []string{"iso88599", "latin5"},
		Description:    "Latin-5 Turkish",
		DetectorLabels: []string{"iso-8859-9"},
	},

	// Other
	"windows-1255": {
		Encoding:       charmap.Windows1255,
		DisplayName:    "Windows-1255",
		Aliases:        []string{"cp1255"},
		Description:    "Windows Hebrew",
		DetectorLabels: []string{"windows-1255"},
	},
	"windows-1256": {
		Encoding:       charmap.Windows1256,
		DisplayName:    "Windows-1256",
		Aliases:        []string{"cp1256"},
		Description:    "Windows Arabic",
		DetectorLabels: []string{"windows-1256"},
	},
	"windows-1257": {
		Encoding:       charmap.Windows1257,
		DisplayName:    "Windows-1257",
		Aliases:        []string{"cp1257"},
		Description:    "Windows Baltic",
		DetectorLabels: []string{"windows-1257"},
	},
	"windows-1258": {
		Encoding:    charmap.Windows1258,
		DisplayName: "Windows-1258",
		Aliases:     []string{"cp1258"},
		Description: "Windows Vietnamese",
	},
	"windows-874": {
		Encoding:       charmap.Windows874,
		DisplayName:    "Windows-874",
		Aliases:        []string{"cp874", "tis-620"},
		Description:    "Windows Thai",
		DetectorLabels: []string{"tis-620"},
	},

	// Chinese (Simplified)
	"gbk": {
		Encoding:       simplifiedchinese.GBK,
		DisplayName:    "GBK",
		Aliases:        []string{"cp936", "gb2312", "gb-2312"},
		Description:    "Chinese Simplified (GBK)",
		DetectorLabels: []string{"gb2312", "hz-gb-2312"},
	},
	"gb18030": {
		Encoding:    simplifiedchinese.GB18030,
		DisplayName: "GB18030",
		Aliases:     []string{"gb-18030"},
		Description: "Chinese Simplified (GB18030, full Unicode)",
	},
}

// registry maps all names (canonical + aliases) to encodingInfo for fast lookup.
var registry map[string]*encodingInfo

// canonicalNames maps all names (canonical + aliases) to the canonical name.
var canonicalNames map[string]string

// detectorCharsets maps a detector label to the name detection may answer with.
var detectorCharsets map[string]string

func init() {
	registry = make(map[string]*encodingInfo)
	canonicalNames = make(map[string]string)
	detectorCharsets = make(map[string]string)
	for canonical, info := range encodings {
		infoCopy := info
		registry[canonical] = &infoCopy
		canonicalNames[canonical] = canonical
		for _, alias := range info.Aliases {
			registry[alias] = &infoCopy
			canonicalNames[alias] = canonical
		}
		for _, label := range info.DetectorLabels {
			detectorCharsets[label] = canonical
		}
	}
	// "ascii" is a narrower claim than utf-8, not a synonym, and Conclusive leans on the difference.
	detectorCharsets[ASCII] = ASCII
}

// ASCII is the detection answer for bytes that are plain ASCII: readable under every encoding here, so it settles nothing.
const ASCII = "ascii"

// detectorCharset resolves a lowercased detector label; not ok means no codec here, so the label is a hint at best.
func detectorCharset(label string) (string, bool) {
	charset, ok := detectorCharsets[label]
	return charset, ok
}

// DetectableCharsets lists what detection is allowed to answer, canonical names sorted.
func DetectableCharsets() []string {
	names := make([]string, 0, len(detectorCharsets))
	seen := make(map[string]bool)
	for _, charset := range detectorCharsets {
		if !seen[charset] {
			seen[charset] = true
			names = append(names, charset)
		}
	}
	sort.Strings(names)
	return names
}

// Count returns the number of supported encodings.
func Count() int {
	return len(encodings)
}

// Canonical resolves a name or alias to the canonical encoding name.
func Canonical(name string) (string, bool) {
	canonical, ok := canonicalNames[strings.ToLower(name)]
	return canonical, ok
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
