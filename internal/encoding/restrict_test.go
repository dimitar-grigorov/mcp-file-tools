// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import "testing"

// Real Spanish CP1252: every uppercase accent before an ASCII letter is a valid
// GBK pair, so the detector reads the file as Chinese.
const spanishCP1252 = "REVISI\xd3N T\xc9CNICA\r\nM\xd3DULO F\xcdSICAMENTE \xdaNICO\r\n" +
	"GESTI\xd3N DE ART\xcdCULOS\r\nC\xd3DIGO \xdaNICO DE VALIDACI\xd3N\r\nDESCRIPCI\xd3N DEL ART\xcdCULO\r\n"

// pin sets the candidate list for one test and clears it afterwards.
func pin(t *testing.T, names ...string) {
	t.Helper()
	if err := SetDetectionCandidates(names); err != nil {
		t.Fatalf("SetDetectionCandidates(%v) = %v", names, err)
	}
	t.Cleanup(func() { _ = SetDetectionCandidates(nil) })
}

func TestDetect_UnpinnedMisreadsSpanishAsGBK(t *testing.T) {
	// Guards the premise: without a pin the guess really is wrong.
	if got := Detect([]byte(spanishCP1252)); got.Charset != "gbk" {
		t.Skipf("chardet no longer misreads this sample (got %q) — the pin is still useful", got.Charset)
	}
}

func TestDetect_Pinned(t *testing.T) {
	utf16LE := append([]byte{0xFF, 0xFE}, "hi"[0], 0, 'i', 0)
	tests := []struct {
		name string
		pin  []string
		data []byte
		want string
	}{
		{"guess outside the pin is replaced", []string{"utf-8", "windows-1252"}, []byte(spanishCP1252), "windows-1252"},
		{"guess inside the pin is kept", []string{"utf-8", "gbk", "windows-1252"}, []byte(spanishCP1252), "gbk"},
		{"first candidate that decodes wins", []string{"utf-8", "windows-1251"}, []byte(spanishCP1252), "windows-1251"},
		{"nothing fits, no answer", []string{"utf-8"}, []byte(spanishCP1252), ""},
		{"a BOM outranks the pin", []string{"windows-1252"}, utf16LE, "utf-16-le"},
		{"aliases resolve", []string{"cp1252"}, []byte(spanishCP1252), "windows-1252"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pin(t, tc.pin...)
			if got := Detect(tc.data); got.Charset != tc.want {
				t.Fatalf("charset = %q, want %q", got.Charset, tc.want)
			}
		})
	}
}

// UTF-16 decodes almost any bytes into something, so it must never win by validation alone.
func TestDetect_PinnedUTF16NeedsStructure(t *testing.T) {
	pin(t, "utf-16-le")
	if got := Detect([]byte(spanishCP1252)); got.Charset != "" {
		t.Fatalf("charset = %q, want no answer", got.Charset)
	}

	utf16 := []byte("h\x00e\x00l\x00l\x00o\x00 \x00w\x00o\x00r\x00l\x00d\x00")
	if got := Detect(utf16); got.Charset != "utf-16-le" {
		t.Fatalf("charset = %q, want utf-16-le from the structural classifier", got.Charset)
	}
}

// A pinned UTF-16 out of the list falls through to the restricted legacy path.
func TestDetect_PinnedRejectsUTF16(t *testing.T) {
	pin(t, "windows-1251")
	utf16 := []byte("h\x00e\x00l\x00l\x00o\x00")
	if got := Detect(utf16); got.Charset != "windows-1251" {
		t.Fatalf("charset = %q, want windows-1251", got.Charset)
	}
}

func TestDetect_UnpinnedIsUnrestricted(t *testing.T) {
	if got := DetectionCandidates(); got != nil {
		t.Fatalf("candidates = %v, want nil by default", got)
	}
	if got := Detect([]byte(spanishCP1252)); got.Charset == "" {
		t.Fatal("unrestricted detection should still answer")
	}
}

func TestSetDetectionCandidates_UnknownName(t *testing.T) {
	if err := SetDetectionCandidates([]string{"utf-8", "klingon-1"}); err == nil {
		t.Fatal("expected an error for an unknown encoding")
	}
	if got := DetectionCandidates(); got != nil {
		t.Fatalf("candidates = %v, want the restriction left off", got)
	}
}

// The ranked alternatives detect_encoding reports honour the same pin.
func TestCandidates_Pinned(t *testing.T) {
	pin(t, "windows-1251")
	for _, c := range CandidatesFromSample([]byte(spanishCP1252)) {
		if c.Charset != "windows-1251" {
			t.Fatalf("ranked candidate %q is outside the pin", c.Charset)
		}
	}
}
