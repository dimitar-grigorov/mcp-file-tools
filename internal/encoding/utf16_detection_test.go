// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func encodeUTF16Fixture(t *testing.T, charset, content string) []byte {
	t.Helper()
	enc, ok := Get(charset)
	if !ok {
		t.Fatalf("encoding %q is not registered", charset)
	}
	data, err := enc.NewEncoder().Bytes([]byte(content))
	if err != nil {
		t.Fatalf("encode %s fixture: %v", charset, err)
	}
	return data
}

func appendUTF16Unit(data []byte, order binary.ByteOrder, unit uint16) []byte {
	var encoded [2]byte
	order.PutUint16(encoded[:], unit)
	return append(data, encoded[:]...)
}

func isUTF16Charset(charset string) bool {
	switch charset {
	case "utf-16-le", "utf-16le", "utf16le", "utf-16-be", "utf-16be", "utf16be":
		return true
	default:
		return false
	}
}

// BOM-less UTF-16 must be detected from content alone, identically across the
// sample, chunked, and full pipelines, for any script.
func TestDetectBOMlessUTF16IsContentBased(t *testing.T) {
	content := "title = encoding acceptance\r\nLatin: Città\r\nCyrillic: Привет\r\nGreek: Γειά\r\nHebrew: שלום\r\nArabic: مرحبا\r\nCJK: 中文\r\nEmoji: 🌍\r\n"
	names := []string{"sample.txt", "sample.dat", "extensionless", "sample.random"}

	for _, charset := range []string{"utf-16-le", "utf-16-be"} {
		t.Run(charset, func(t *testing.T) {
			data := encodeUTF16Fixture(t, charset, content)
			direct := Detect(data)
			if direct.Charset != charset || direct.Confidence < HighConfidenceThreshold {
				t.Fatalf("Detect = %+v, want %s with confidence >= %d", direct, charset, HighConfidenceThreshold)
			}

			tempDir := t.TempDir()
			for _, name := range names {
				path := filepath.Join(tempDir, name)
				if err := os.WriteFile(path, data, 0644); err != nil {
					t.Fatal(err)
				}
				for _, mode := range []string{"sample", "chunked", "full"} {
					result, err := DetectFromFile(path, mode)
					if err != nil {
						t.Fatalf("%s/%s: %v", name, mode, err)
					}
					if result != direct {
						t.Fatalf("%s/%s result = %+v, direct = %+v", name, mode, result, direct)
					}
				}
			}
		})
	}
}

// Structurally broken UTF-16 must stay ambiguous, not fall through to a wrong
// single-byte guess.
func TestDetectBOMlessUTF16RejectsMalformedUnicode(t *testing.T) {
	for _, encoding := range []struct {
		name    string
		charset string
		order   binary.ByteOrder
	}{
		{name: "little endian", charset: "utf-16-le", order: binary.LittleEndian},
		{name: "big endian", charset: "utf-16-be", order: binary.BigEndian},
	} {
		t.Run(encoding.name, func(t *testing.T) {
			prefix := encodeUTF16Fixture(t, encoding.charset, "header = valid text\r\n")
			tests := []struct {
				name string
				data []byte
			}{
				{name: "odd byte length", data: append(append([]byte(nil), prefix...), 0x41)},
				{name: "isolated high surrogate", data: appendUTF16Unit(append([]byte(nil), prefix...), encoding.order, 0xD83D)},
				{name: "high surrogate followed by non-low", data: appendUTF16Unit(appendUTF16Unit(append([]byte(nil), prefix...), encoding.order, 0xD83D), encoding.order, 'A')},
				{name: "isolated low surrogate", data: appendUTF16Unit(append([]byte(nil), prefix...), encoding.order, 0xDC00)},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					result := Detect(tt.data)
					if result.Charset != "" || result.Confidence != 0 {
						t.Fatalf("Detect = %+v, malformed UTF-16 must remain ambiguous", result)
					}
				})
			}
		})
	}
}

// Binary blobs with NUL bytes must never be accepted as UTF-16.
func TestDetectBOMlessUTF16RejectsBinaryFalsePositives(t *testing.T) {
	randomData := make([]byte, 512)
	rand.New(rand.NewSource(42)).Read(randomData)

	alternatingControls := make([]byte, 0, 256)
	for i := 0; i < 128; i++ {
		alternatingControls = append(alternatingControls, byte(i%8+1), 0x00)
	}

	sparseNUL := bytes.Repeat([]byte{0x91, 0xA4, 0x00, 0x7F, 0xCC, 0x10, 0xE3, 0x22}, 64)
	tests := []struct {
		name string
		data []byte
	}{
		{name: "png", data: append([]byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), bytes.Repeat([]byte{0x00, 0xFF, 0x12, 0x80}, 64)...)},
		{name: "portable executable", data: append([]byte("MZ"), bytes.Repeat([]byte{0x00, 0x01, 0x02, 0x03}, 128)...)},
		{name: "zip", data: append([]byte("PK\x03\x04"), bytes.Repeat([]byte{0x00, 0xFF, 0x08, 0x00}, 128)...)},
		{name: "alternating NUL controls", data: alternatingControls},
		{name: "sparse NUL", data: sparseNUL},
		{name: "deterministic random", data: randomData},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Detect(tt.data)
			if isUTF16Charset(result.Charset) {
				t.Fatalf("Detect = %+v, binary input must not be accepted as UTF-16", result)
			}
		})
	}
}

// Real single-byte and UTF-8 text must keep their legacy detection.
func TestDetectBOMlessUTF16DoesNotOverrideUTF8OrLegacyEncodings(t *testing.T) {
	cp1251, ok := Get("cp1251")
	if !ok {
		t.Fatal("cp1251 encoding is not registered")
	}
	cp1251Data, err := cp1251.NewEncoder().Bytes([]byte("Привет, это обычный текст в кодовой странице."))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		data []byte
	}{
		{name: "UTF-8 multilingual", data: []byte("Città Привет 中文 🌍")},
		{name: "ASCII", data: []byte("plain ASCII text with punctuation, spaces, and newlines\n")},
		{name: "CP1251", data: cp1251Data},
		{name: "GBK", data: gbkEncode(t, "你好，世界！这是一个用于测试编码检测的中文字符串。")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Detect(tt.data)
			if isUTF16Charset(result.Charset) {
				t.Fatalf("Detect = %+v, non-UTF-16 input must retain legacy detection", result)
			}
		})
	}
}

func TestDetectBOMOnlyRemainsAuthoritative(t *testing.T) {
	for _, charset := range []string{"utf-8", "utf-16-le", "utf-16-be", "utf-32-le", "utf-32-be"} {
		result := Detect(BOMBytesFor(charset))
		if result.Charset != charset || result.Confidence != 100 || !result.HasBOM {
			t.Fatalf("%s BOM result = %+v", charset, result)
		}
	}
}

// Too-short or endian-ambiguous input must not be forced to a UTF-16 guess.
func TestDetectBOMlessUTF16LeavesShortAndEndianAmbiguousInputUnforced(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "short LE", data: encodeUTF16Fixture(t, "utf-16-le", "Hi")},
		{name: "short BE", data: encodeUTF16Fixture(t, "utf-16-be", "Hi")},
		{name: "endian ambiguous printable units", data: bytes.Repeat([]byte{0x2D, 0x4E}, 16)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Detect(tt.data)
			if isUTF16Charset(result.Charset) {
				t.Fatalf("Detect = %+v, ambiguous input must not be forced to UTF-16", result)
			}
		})
	}
}

// A surrogate pair split across the 128KB chunk boundary must still be detected.
func TestDetectFromFileBOMlessUTF16AcrossChunkBoundary(t *testing.T) {
	for _, tt := range []struct {
		charset string
		pair    []byte
		unit    []byte
	}{
		{charset: "utf-16-le", pair: []byte{0x3D, 0xD8, 0x0D, 0xDF}, unit: []byte{'A', 0x00}},
		{charset: "utf-16-be", pair: []byte{0xD8, 0x3D, 0xDF, 0x0D}, unit: []byte{0x00, 'A'}},
	} {
		t.Run(tt.charset, func(t *testing.T) {
			prefix := bytes.Repeat(tt.unit, ChunkSize/2-1)
			data := append(prefix, tt.pair...)
			data = append(data, bytes.Repeat(tt.unit, ChunkSize/2+32)...)

			path := filepath.Join(t.TempDir(), "boundary.random")
			if err := os.WriteFile(path, data, 0644); err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"sample", "chunked", "full"} {
				result, err := DetectFromFile(path, mode)
				if err != nil {
					t.Fatalf("%s: %v", mode, err)
				}
				if result.Charset != tt.charset || result.Confidence < HighConfidenceThreshold {
					t.Fatalf("%s result = %+v, want %s", mode, result, tt.charset)
				}
			}
		})
	}
}

func FuzzDetectUTF16Validation(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{'A', 0x00, 'B', 0x00})
	f.Add([]byte{0x3D, 0xD8, 0x0D, 0xDF})
	f.Add([]byte{0x00, 0xD8, 0x41, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		result := Detect(data)
		if isUTF16Charset(result.Charset) && result.Confidence < MinConfidenceThreshold {
			t.Fatalf("trusted UTF-16 result has low confidence: %+v", result)
		}
	})
}
