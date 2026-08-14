// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cp1251Bytes holds text the way a legacy Delphi source does.
func cp1251Bytes(t *testing.T, text string) []byte {
	t.Helper()
	enc, ok := Get("windows-1251")
	if !ok {
		t.Fatal("windows-1251 missing from the registry")
	}
	data, err := Encode(text, enc, "windows-1251")
	if err != nil {
		t.Fatalf("encode to cp1251: %v", err)
	}
	return data
}

const bulgarianText = "Неуспешна връзка със сървъра. Проверете настройките и опитайте отново, " +
	"защото поръчката не е изпратена. Списъкът с адреси е празен в момента."

func TestCandidatesRanksCyrillicFirst(t *testing.T) {
	ranked := candidates(cp1251Bytes(t, bulgarianText))
	if len(ranked) == 0 {
		t.Fatal("no candidates for cp1251 text")
	}
	if ranked[0].Charset != "windows-1251" {
		t.Errorf("top candidate = %q, want windows-1251 (full list: %v)", ranked[0].Charset, ranked)
	}
	if !ranked[0].Supported {
		t.Error("windows-1251 reported as unsupported")
	}
	for i := 1; i < len(ranked); i++ {
		if ranked[i].Confidence > ranked[i-1].Confidence {
			t.Errorf("candidates out of order: %v", ranked)
		}
	}
}

func TestCandidatesNilWhenSettled(t *testing.T) {
	utf16 := []byte{0xFF, 0xFE, 'h', 0x00, 'i', 0x00} // BOM decides it
	if ranked := candidates(utf16); ranked != nil {
		t.Errorf("BOM file got candidates: %v", ranked)
	}

	bomless := []byte{'h', 0x00, 'e', 0x00, 'l', 0x00, 'l', 0x00, 'o', 0x00, '\n', 0x00}
	if ranked := candidates(bomless); ranked != nil {
		t.Errorf("BOM-less UTF-16 got candidates: %v", ranked)
	}
}

// An ASCII file has one answer, under the registry's name for it.
func TestCandidatesASCII(t *testing.T) {
	ranked := candidates([]byte("procedure Main;\nbegin\n  Writeln('hi');\nend;\n"))
	if len(ranked) != 1 {
		t.Fatalf("got %d candidates for ASCII, want 1: %v", len(ranked), ranked)
	}
	if !SameCharset(ranked[0].Charset, "ascii") {
		t.Errorf("ASCII candidate = %q", ranked[0].Charset)
	}
}

// Candidates are ranked over the same bytes the detector sampled.
func TestCandidatesFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unit1.pas")
	if err := os.WriteFile(path, cp1251Bytes(t, bulgarianText), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, mode := range []string{"sample", "chunked", "full"} {
		ranked, err := CandidatesFromFile(path, mode)
		if err != nil {
			t.Fatalf("mode %s: %v", mode, err)
		}
		if len(ranked) == 0 || ranked[0].Charset != "windows-1251" {
			t.Errorf("mode %s ranked %v, want windows-1251 first", mode, ranked)
		}
	}

	if _, err := CandidatesFromFile(filepath.Join(dir, "missing.pas"), "sample"); err == nil {
		t.Error("missing file returned no error")
	}
}

// A buffer past the in-memory threshold is sampled, not scanned whole.
func TestCandidatesFromSampleLargeBuffer(t *testing.T) {
	body := cp1251Bytes(t, bulgarianText)
	large := make([]byte, 0, SmallFileThreshold*3)
	for len(large) <= SmallFileThreshold*2 {
		large = append(large, body...)
	}

	ranked := CandidatesFromSample(large)
	if len(ranked) == 0 || ranked[0].Charset != "windows-1251" {
		t.Errorf("ranked %v, want windows-1251 first", ranked)
	}
}

func TestSupportedAlternativesAndFormat(t *testing.T) {
	ranked := []Candidate{
		{Charset: "macroman", Confidence: 90, Supported: false},
		{Charset: "windows-1251", Confidence: 62, Supported: true},
		{Charset: "utf-8", Confidence: 55, Supported: true},
	}

	alternatives := SupportedAlternatives(ranked, "ascii") // ascii == utf-8, so it drops out
	if got := FormatCandidates(alternatives); got != "windows-1251 (62%)" {
		t.Errorf("alternatives = %q", got)
	}
	if got := FormatCandidates(SupportedAlternatives(ranked, "windows-1251")); got != "utf-8 (55%)" {
		t.Errorf("excluding the verdict itself = %q", got)
	}
	if got := FormatCandidates(nil); got != "" {
		t.Errorf("empty list formatted as %q", got)
	}
}

func TestSameCharset(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"ascii", "utf-8", true},
		{"CP1251", "windows-1251", true},
		{"utf-8", "utf-8", true},
		{"windows-1251", "koi8-r", false},
		{"macroman", "macroman", true}, // unregistered, but the same name
		{"macroman", "big5", false},
	}
	for _, c := range cases {
		if got := SameCharset(c.a, c.b); got != c.want {
			t.Errorf("SameCharset(%q, %q) = %v", c.a, c.b, got)
		}
	}
}

// The refactor that split correctCharset out of detectLegacy must not have moved
// any verdict: these are the corrections it carries.
func TestCorrectCharsetCorrections(t *testing.T) {
	utf8Text := []byte("край на файла — със знаци извън ASCII")
	cases := []struct {
		name       string
		charset    string
		confidence int
		data       []byte
		want       string
	}{
		{"BOM-less UTF-16 is dropped", "utf-16-le", 90, nil, ""},
		{"gb2312 folds into gbk", "gb2312", 70, nil, "gbk"},
		{"single-byte guess loses to valid UTF-8", "windows-1251", 60, utf8Text, "utf-8"},
		{"unknown names pass through", "macroman", 80, []byte{0xC0, 0xC1}, "macroman"},
	}
	for _, c := range cases {
		if got, _ := correctCharset(c.charset, c.confidence, c.data); got != c.want {
			t.Errorf("%s: correctCharset(%q) = %q, want %q", c.name, c.charset, got, c.want)
		}
	}

	if got, confidence := correctCharset("windows-1251", 60, cp1251Bytes(t, bulgarianText)); got != "windows-1251" || confidence != 60 {
		t.Errorf("cp1251 verdict changed to %q at %d%%", got, confidence)
	}
}

// The utf-8 override lands several probes on one name; the list must not repeat it.
func TestCandidatesDeduplicates(t *testing.T) {
	ranked := candidates([]byte(strings.Repeat("грешка при отваряне на файла, опитайте пак. ", 40)))
	seen := make(map[string]bool)
	for _, candidate := range ranked {
		if seen[candidate.Charset] {
			t.Errorf("duplicate candidate %q in %v", candidate.Charset, ranked)
		}
		seen[candidate.Charset] = true
	}
}
