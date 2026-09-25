// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"slices"
	"testing"

	"golang.org/x/text/transform"
)

// Every codec in the registry has to survive a round trip through text it was made for, or it is a name with no encoder behind it.
func TestEveryEncodingRoundTrips(t *testing.T) {
	samples := map[string]string{
		"macintosh":   "Le café était très agréable",
		"ibm437":      "Ungültiger Dateiname",
		"ibm850":      "Ungültiger Dateiname",
		"ibm852":      "Nepřístupný soubor",
		"ibm855":      "Неуспешна връзка",
		"iso-8859-3":  "Ġensna Malti",
		"iso-8859-4":  "Šiaurės Eiropa",
		"iso-8859-6":  "مرحبا بالعالم",
		"iso-8859-8":  "שלום עולם",
		"iso-8859-10": "Norðurlönd",
		"iso-8859-13": "Šiaurės Europa",
		"iso-8859-14": "Gaeilge agus Cymraeg",
		"iso-8859-16": "Română și maghiară",
		"big5":        "下午 12:40:08 台灣",
		"shift_jis":   "日本語のテキスト",
		"euc-jp":      "日本語のテキスト",
		"iso-2022-jp": "日本語のテキスト",
		"euc-kr":      "한국어 텍스트",
		"utf-32-le":   "Unicode текст 日本語",
		"utf-32-be":   "Unicode текст 日本語",
	}
	for name, text := range samples {
		t.Run(name, func(t *testing.T) {
			enc, ok := Get(name)
			if !ok || enc == nil {
				t.Fatalf("%s is not in the registry", name)
			}
			encoded, _, err := transform.Bytes(enc.NewEncoder(), []byte(text))
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			decoded, _, err := transform.Bytes(enc.NewDecoder(), encoded)
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			if string(decoded) != text {
				t.Errorf("round trip gave %q, want %q", decoded, text)
			}
		})
	}
}

// A table added for decoding must stay out of detection: it reads almost any bytes, and guessing it costs a correct verdict elsewhere.
func TestDecodeOnlyTablesAreNeverGuessed(t *testing.T) {
	detectable := DetectableCharsets()
	for _, name := range []string{"macintosh", "ibm437", "ibm850", "ibm852", "ibm855",
		"iso-8859-3", "iso-8859-4", "iso-8859-6", "iso-8859-8", "iso-8859-10",
		"iso-8859-13", "iso-8859-14", "iso-8859-16", "utf-32-le", "utf-32-be"} {
		if _, ok := Get(name); !ok {
			t.Errorf("%s should be decodable", name)
		}
		if slices.Contains(detectable, name) {
			t.Errorf("%s is a detection answer, it should only decode", name)
		}
	}
}
