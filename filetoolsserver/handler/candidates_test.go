// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Samples chosen for a stable chardet verdict, noted per line.
var (
	latin1Accents = []byte("caf\xE9 na\xEFve r\xF4le\n")                 // iso-8859-1, 73%: trusted, not certain
	macAccents    = []byte("caf\x8E na\x9Fve r\x99le \xA5\n")            // macroman, 46%: unsupported and untrusted
	big5Sample    = []byte("\xA4\xE9\xA5\xBB\xB8\xEA\xAE\xC6\xAA\xF8\n") // big5, 99%: outside the registry, so never a verdict
	cp1251Sample  = []byte("\xC4\xEE\xE1\xF0\xE5 \xF3\xF2\xF0\xEE \xF1\xE2\xFF\xF2\n")
)

func writeSample(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDetectEncodingCandidates(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})

	cases := []struct {
		name     string
		data     []byte
		wantSome bool
	}{
		{"ascii", []byte("procedure Main;\nbegin\nend;\n"), false},
		{"utf8-bom", append([]byte{0xEF, 0xBB, 0xBF}, "Привет"...), false},
		{"cp1251", cp1251Sample, false},
		{"latin1-73pct", latin1Accents, true},
	}

	for _, c := range cases {
		path := writeSample(t, dir, c.name+".txt", c.data)
		_, output, err := h.HandleDetectEncoding(context.Background(), nil, DetectEncodingInput{Path: path})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := len(output.Candidates) > 0; got != c.wantSome {
			t.Errorf("%s: detected %s at %d%%, candidates=%v", c.name, output.Encoding, output.Confidence, output.Candidates)
		}
	}
}

// Big5 has no codec here, so detection declines rather than naming it, and the refusal says what to try.
func TestDetectEncodingDeclinesAnUnreadableCharset(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := writeSample(t, dir, "big5.txt", big5Sample)

	result, output, err := h.HandleDetectEncoding(context.Background(), nil, DetectEncodingInput{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if output.Encoding == "big5" {
		t.Fatalf("detection answered big5, which the registry cannot read")
	}
	if !result.IsError {
		return // it found a readable answer instead, which is the other acceptable outcome
	}
	if message := extractTextFromResult(result.Content); !strings.Contains(message, "Other candidates:") {
		t.Errorf("refusal %q leaves the caller nowhere to go", message)
	}
}

// Without alternatives in the hint, an unreadable file is a dead end.
func TestReadHintNamesAlternatives(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})

	cases := []struct {
		name string
		data []byte
		want []string
	}{
		{"big5", big5Sample, []string{"inconclusive", "retry read_text_file with encoding set to one of:", "iso-8859-1"}},
		{"macroman", macAccents, []string{"inconclusive", "retry read_text_file with encoding set to one of:", "windows-1254"}},
	}

	for _, c := range cases {
		path := writeSample(t, dir, c.name+".txt", c.data)
		encResult, err := h.resolveEncoding("", path)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		for _, want := range c.want {
			if !strings.Contains(encResult.fallbackHint, want) {
				t.Errorf("%s: hint %q does not mention %q", c.name, encResult.fallbackHint, want)
			}
		}
	}
}

func TestReadHintQuietWhenSettled(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := writeSample(t, dir, "cp1251.pas", cp1251Sample)

	encResult, err := h.resolveEncoding("", path)
	if err != nil {
		t.Fatal(err)
	}
	if encResult.fallbackHint != "" {
		t.Errorf("unexpected hint on a confident detection: %q", encResult.fallbackHint)
	}
}

func TestConvertEncodingErrorsNameAlternatives(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})

	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"untrusted", macAccents, "windows-1254"},
		{"undetected", big5Sample, "iso-8859-1"},
	}

	for _, c := range cases {
		path := writeSample(t, dir, c.name+".txt", c.data)
		result, _, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{Path: path, To: "utf-8"})
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if !result.IsError {
			t.Fatalf("%s: expected the conversion to be refused", c.name)
		}
		message := extractTextFromResult(result.Content)
		if !strings.Contains(message, "Other candidates:") || !strings.Contains(message, c.want) {
			t.Errorf("%s: error %q does not offer %s", c.name, message, c.want)
		}
	}
}
