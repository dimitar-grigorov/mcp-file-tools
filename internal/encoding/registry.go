// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"sort"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
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
// A single-byte table reads almost any bytes, so one added for decoding gets no label: measured, MacRoman steals a correct cp1251 verdict for the sake of a binary .dfm. A multi-byte label is structural, its bytes have to form valid sequences, so those do answer.
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
	"utf-32-le": {
		Encoding:    utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM),
		DisplayName: "UTF-32 LE",
		Aliases:     []string{"utf32le", "utf-32le"},
		Description: "Unicode UTF-32 Little Endian",
	},
	"utf-32-be": {
		Encoding:    utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM),
		DisplayName: "UTF-32 BE",
		Aliases:     []string{"utf32be", "utf-32be"},
		Description: "Unicode UTF-32 Big Endian",
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
	"ibm855": {
		Encoding:    charmap.CodePage855,
		DisplayName: "CP855",
		Aliases:     []string{"cp855", "dos-855"},
		Description: "DOS Cyrillic (IBM)",
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

	// Baltic
	"iso-8859-13": {
		Encoding:    charmap.ISO8859_13,
		DisplayName: "ISO-8859-13",
		Aliases:     []string{"iso885913", "latin7"},
		Description: "Latin-7 Baltic",
	},
	"iso-8859-4": {
		Encoding:    charmap.ISO8859_4,
		DisplayName: "ISO-8859-4",
		Aliases:     []string{"iso88594", "latin4"},
		Description: "Latin-4 North European",
	},

	// Hebrew and Arabic
	"iso-8859-8": {
		Encoding:    charmap.ISO8859_8,
		DisplayName: "ISO-8859-8",
		Aliases:     []string{"iso88598", "hebrew"},
		Description: "ISO Hebrew (visual order)",
	},
	"iso-8859-6": {
		Encoding:    charmap.ISO8859_6,
		DisplayName: "ISO-8859-6",
		Aliases:     []string{"iso88596", "arabic"},
		Description: "ISO Arabic",
	},

	// Macintosh
	"macintosh": {
		Encoding:    charmap.Macintosh,
		DisplayName: "MacRoman",
		Aliases:     []string{"macroman", "mac-roman", "x-mac-roman"},
		Description: "Macintosh Western European",
	},

	// DOS code pages
	"ibm437": {
		Encoding:    charmap.CodePage437,
		DisplayName: "CP437",
		Aliases:     []string{"cp437", "dos-437"},
		Description: "DOS United States (original IBM PC)",
	},
	"ibm850": {
		Encoding:    charmap.CodePage850,
		DisplayName: "CP850",
		Aliases:     []string{"cp850", "dos-850"},
		Description: "DOS Western European",
	},
	"ibm852": {
		Encoding:    charmap.CodePage852,
		DisplayName: "CP852",
		Aliases:     []string{"cp852", "dos-852"},
		Description: "DOS Central European",
	},

	// Less common ISO tables, no detector probe behind them
	"iso-8859-3": {
		Encoding:    charmap.ISO8859_3,
		DisplayName: "ISO-8859-3",
		Aliases:     []string{"iso88593", "latin3"},
		Description: "Latin-3 South European",
	},
	"iso-8859-10": {
		Encoding:    charmap.ISO8859_10,
		DisplayName: "ISO-8859-10",
		Aliases:     []string{"iso885910", "latin6"},
		Description: "Latin-6 Nordic",
	},
	"iso-8859-14": {
		Encoding:    charmap.ISO8859_14,
		DisplayName: "ISO-8859-14",
		Aliases:     []string{"iso885914", "latin8"},
		Description: "Latin-8 Celtic",
	},
	"iso-8859-16": {
		Encoding:    charmap.ISO8859_16,
		DisplayName: "ISO-8859-16",
		Aliases:     []string{"iso885916", "latin10"},
		Description: "Latin-10 South-Eastern European",
	},

	// Chinese (Traditional)
	"big5": {
		Encoding:       traditionalchinese.Big5,
		DisplayName:    "Big5",
		Aliases:        []string{"big-5", "cp950"},
		Description:    "Chinese Traditional (Big5)",
		DetectorLabels: []string{"big5"},
	},

	// Japanese
	"shift_jis": {
		Encoding:       japanese.ShiftJIS,
		DisplayName:    "Shift_JIS",
		Aliases:        []string{"shift-jis", "sjis", "cp932", "windows-31j"},
		Description:    "Japanese (Shift_JIS)",
		DetectorLabels: []string{"shift_jis", "cp932"},
	},
	"euc-jp": {
		Encoding:       japanese.EUCJP,
		DisplayName:    "EUC-JP",
		Aliases:        []string{"eucjp", "x-euc-jp"},
		Description:    "Japanese (EUC)",
		DetectorLabels: []string{"euc-jp"},
	},
	"iso-2022-jp": {
		Encoding:       japanese.ISO2022JP,
		DisplayName:    "ISO-2022-JP",
		Aliases:        []string{"iso2022jp", "csiso2022jp"},
		Description:    "Japanese (ISO-2022, escape sequences)",
		DetectorLabels: []string{"iso-2022-jp"},
	},

	// Korean
	"euc-kr": {
		Encoding:       korean.EUCKR,
		DisplayName:    "EUC-KR",
		Aliases:        []string{"euckr", "cp949", "uhc", "windows-949"},
		Description:    "Korean (EUC, CP949)",
		DetectorLabels: []string{"euc-kr", "cp949"},
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
