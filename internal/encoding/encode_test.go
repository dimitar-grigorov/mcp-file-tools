// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"errors"
	"strings"
	"testing"
)

func TestEncodeSucceedsWhenCharsetCovers(t *testing.T) {
	enc, ok := Get("cp1251")
	if !ok {
		t.Fatal("cp1251 not registered")
	}
	out, err := Encode("Привет мир", enc, "cp1251")
	if err != nil {
		t.Fatalf("Encode() error = %v, want nil", err)
	}
	if len(out) == 0 {
		t.Fatal("Encode() returned no bytes")
	}
}

func TestEncodeNamesUnsupportedRunes(t *testing.T) {
	enc, _ := Get("cp1251")

	// Line 1 has no offenders; the umlauts are on line 2, the sharp s on line 3.
	content := "ok ascii\nBäcker Grüße\nStraße\n"

	_, err := Encode(content, enc, "cp1251")
	if err == nil {
		t.Fatal("Encode() error = nil, want UnsupportedError")
	}

	var ue *UnsupportedError
	if !errors.As(err, &ue) {
		t.Fatalf("Encode() error type = %T, want *UnsupportedError", err)
	}

	// ä, ü, ß, ß = 4 offenders
	if ue.Total != 4 {
		t.Errorf("Total = %d, want 4", ue.Total)
	}
	if ue.Charset != "cp1251" {
		t.Errorf("Charset = %q, want cp1251", ue.Charset)
	}

	first := ue.Runes[0]
	if first.Char != "ä" {
		t.Errorf("first char = %q, want ä", first.Char)
	}
	if first.Code != "U+00E4" {
		t.Errorf("first code = %q, want U+00E4", first.Code)
	}
	if first.Line != 2 {
		t.Errorf("first line = %d, want 2", first.Line)
	}
	if first.Column != 2 {
		t.Errorf("first column = %d, want 2", first.Column)
	}

	// The message has to carry the actionable part, not just the count.
	msg := ue.Error()
	for _, want := range []string{"cp1251", "ä", "U+00E4", "line 2", "column 2", "utf-8"} {
		if !strings.Contains(msg, want) {
			t.Errorf("Error() = %q, missing %q", msg, want)
		}
	}
}

func TestEncodeCapsReportedRunes(t *testing.T) {
	enc, _ := Get("iso-8859-1")

	// 25 Cyrillic runes, none representable in latin-1.
	content := strings.Repeat("ф", 25)

	_, err := Encode(content, enc, "iso-8859-1")
	var ue *UnsupportedError
	if !errors.As(err, &ue) {
		t.Fatalf("error type = %T, want *UnsupportedError", err)
	}
	if ue.Total != 25 {
		t.Errorf("Total = %d, want 25", ue.Total)
	}
	if len(ue.Runes) != maxReportedRunes {
		t.Errorf("len(Runes) = %d, want %d", len(ue.Runes), maxReportedRunes)
	}
	if msg := ue.Error(); !strings.Contains(msg, "25 characters") {
		t.Errorf("Error() = %q, want the full count", msg)
	}
}

func TestFindUnsupportedNilWhenCovered(t *testing.T) {
	enc, _ := Get("cp1251")
	if got := findUnsupported("Привет ASCII 123", enc, "cp1251"); got != nil {
		t.Errorf("findUnsupported() = %v, want nil", got)
	}
}

func TestFindUnsupportedTracksColumnsAfterCRLF(t *testing.T) {
	enc, _ := Get("cp1251")
	// CRLF must not shift the reported column on the following line.
	ue := findUnsupported("a\r\nxä", enc, "cp1251")
	if ue == nil {
		t.Fatal("findUnsupported() = nil, want an offender")
	}
	if ue.Runes[0].Line != 2 {
		t.Errorf("line = %d, want 2", ue.Runes[0].Line)
	}
	if ue.Runes[0].Column != 2 {
		t.Errorf("column = %d, want 2", ue.Runes[0].Column)
	}
}
