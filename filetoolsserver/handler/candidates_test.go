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
	big5Sample    = []byte("\xA4\xE9\xA5\xBB\xB8\xEA\xAE\xC6\xAA\xF8\n") // big5, 99%: trusted, outside the registry
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
		{"big5-unsupported", big5Sample, true},
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

// The unsupported verdict is listed too, flagged as unusable.
func TestDetectEncodingCandidatesFlagUnsupported(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := writeSample(t, dir, "big5.txt", big5Sample)

	_, output, err := h.HandleDetectEncoding(context.Background(), nil, DetectEncodingInput{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Candidates) < 2 {
		t.Fatalf("want the verdict plus alternatives, got %v", output.Candidates)
	}
	if output.Candidates[0].Encoding != "big5" || output.Candidates[0].Supported {
		t.Errorf("first candidate = %+v, want big5 marked unsupported", output.Candidates[0])
	}
	if !output.Candidates[1].Supported {
		t.Errorf("second candidate %+v is unusable, so it is no alternative", output.Candidates[1])
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
		{"big5", big5Sample, []string{"big5 is not supported", "Ranked alternatives:", "iso-8859-1"}},
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
		{"unsupported", big5Sample, "iso-8859-1"},
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
