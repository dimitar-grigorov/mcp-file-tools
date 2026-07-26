// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"testing"
	"unicode/utf16"
)

// encUTF16Test renders s as UTF-16 bytes without a BOM, for the given endianness.
func encUTF16Test(s string, littleEndian bool) []byte {
	units := utf16.Encode([]rune(s))
	b := make([]byte, 0, len(units)*2)
	for _, u := range units {
		if littleEndian {
			b = append(b, byte(u), byte(u>>8))
		} else {
			b = append(b, byte(u>>8), byte(u))
		}
	}
	return b
}

// BOM-less UTF-16 of non-Latin scripts is the real gap: chardet reports "ascii"
// or a wrong single-byte codec and the text decodes to garbage. Detect must
// recognize the structural signal and return the right charset and endianness.
func TestDetect_BOMlessUTF16(t *testing.T) {
	cases := []struct {
		name string
		text string
		le   bool
		want string
	}{
		{"cyrillic-le", "Привет, мир! Това е конфигурационен файл.\nключ=стойност\n", true, "utf-16-le"},
		{"cyrillic-be", "Привет, мир! Това е конфигурационен файл.\nключ=стойност\n", false, "utf-16-be"},
		{"mixed-ascii-cyrillic-le", "port=8080\nхост=localhost\nname=Приложение\n", true, "utf-16-le"},
		{"ascii-le", "Hello, world!\nThis is a plain config.\nkey=value\nport=8080\n", true, "utf-16-le"},
		{"ascii-be", "Hello, world!\nThis is a plain config.\nkey=value\nport=8080\n", false, "utf-16-be"},
		{"greek-le", "Καλημέρα κόσμε\nαρχείο ρυθμίσεων\n", true, "utf-16-le"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Detect(encUTF16Test(tc.text, tc.le))
			if got.Charset != tc.want {
				t.Errorf("Detect = %q (conf %d), want %q", got.Charset, got.Confidence, tc.want)
			}
		})
	}
}

// The detector must not fire on real single-byte or UTF-8 text, otherwise it
// would corrupt files that decode fine today.
func TestDetect_BOMlessUTF16_NoFalsePositives(t *testing.T) {
	// Real Windows-1251 Cyrillic bytes (single-byte, high bit set, no nulls).
	cp1251 := []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2, 0x20, 0xEC, 0xE8, 0xF0, 0x0A} // "Привет мир\n"
	if got := Detect(cp1251); got.Charset == "utf-16-le" || got.Charset == "utf-16-be" {
		t.Errorf("cp1251 text misdetected as %q", got.Charset)
	}

	utf8Text := []byte("Привет мир\nこんにちは\nПлайн UTF-8 текст с достатъчно съдържание.\n")
	if got := Detect(utf8Text); got.Charset == "utf-16-le" || got.Charset == "utf-16-be" {
		t.Errorf("utf-8 text misdetected as %q", got.Charset)
	}

	ascii := []byte("plain ascii config\nkey=value\nport=8080\nenabled=true\n")
	if got := Detect(ascii); got.Charset == "utf-16-le" || got.Charset == "utf-16-be" {
		t.Errorf("ascii text misdetected as %q", got.Charset)
	}
}
