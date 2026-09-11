// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"
)

var detectionModes = []string{"sample", "chunked", "full"}

// writeTempFile drops data in a file the caller can point DetectFromFile at.
func writeTempFile(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.pas")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// asciiFiller is plain Pascal, the same under every encoding here, in the quantity a real unit has.
func asciiFiller(bytes int) string {
	const line = "procedure DoSomething(const AValue: Integer; var AResult: string);\r\n"
	return strings.Repeat(line, bytes/len(line)+1)
}

// A unit that is ASCII for its first chunks and Cyrillic at the end is a Cyrillic file. Reading only the head says ascii, which is not an answer, and the cheap modes used to stop there.
func TestDetectFromFile_EvidencePastTheHead(t *testing.T) {
	data := append([]byte(asciiFiller(3*ChunkSize)), charmapEncode(t, charmap.Windows1251, strings.Repeat(cyrillicFixture+"\r\n", 40))...)
	path := writeTempFile(t, data)

	for _, mode := range detectionModes {
		t.Run(mode, func(t *testing.T) {
			result, err := DetectFromFile(path, mode)
			if err != nil {
				t.Fatal(err)
			}
			if result.Charset != "windows-1251" {
				t.Errorf("mode %s: got %s (%d%%), want windows-1251", mode, result.Charset, result.Confidence)
			}
		})
	}
}

// The tail alone carrying the evidence is the harder half: the modes that sample or vote weigh far more ASCII than Cyrillic.
func TestDetectFromFile_ASCIIDoesNotOutweighEvidence(t *testing.T) {
	utf8Tail := strings.Repeat(cyrillicFixture+"\r\n", 40)
	tests := []struct {
		name string
		tail []byte
		want string
	}{
		{"cyrillic tail", charmapEncode(t, charmap.Windows1251, utf8Tail), "windows-1251"},
		{"utf-8 tail", []byte(utf8Tail), "utf-8"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTempFile(t, append([]byte(asciiFiller(5*ChunkSize)), tc.tail...))
			for _, mode := range detectionModes {
				result, err := DetectFromFile(path, mode)
				if err != nil {
					t.Fatal(err)
				}
				if result.Charset != tc.want {
					t.Errorf("mode %s: got %s (%d%%), want %s", mode, result.Charset, result.Confidence, tc.want)
				}
			}
		})
	}
}

// A file with nothing but ASCII in it must still say so rather than reach for a table at random.
func TestDetectFromFile_AllASCIIStaysASCII(t *testing.T) {
	path := writeTempFile(t, []byte(asciiFiller(5*ChunkSize)))
	for _, mode := range detectionModes {
		result, err := DetectFromFile(path, mode)
		if err != nil {
			t.Fatal(err)
		}
		if result.Charset != ASCII || result.Conclusive() {
			t.Errorf("mode %s: got %s (%d%%), want ascii and inconclusive", mode, result.Charset, result.Confidence)
		}
	}
}

// A BOM is a declaration, so it settles the answer before any mode gets to read the rest.
func TestDetectFromFile_BOMWinsInEveryMode(t *testing.T) {
	data := append(BOMBytesFor("utf-8"), asciiFiller(5*ChunkSize)...)
	data = append(data, cyrillicFixture...)
	path := writeTempFile(t, data)

	for _, mode := range detectionModes {
		result, err := DetectFromFile(path, mode)
		if err != nil {
			t.Fatal(err)
		}
		if result.Charset != "utf-8" || !result.HasBOM {
			t.Errorf("mode %s: got %s hasBOM=%t, want utf-8 with a BOM", mode, result.Charset, result.HasBOM)
		}
	}
}
