package handler

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
)

// Multi-script sample, so detection sees realistic UTF-16.
const (
	utf16SampleCRLF  = "#property strict\r\n// Привет, мир\r\n// città\r\n// 中文\r\n"
	utf16SampleLF    = "#property strict\n// Привет, мир\n// città\n// 中文\n"
	utf16SampleMixed = "#property strict\r\n// Привет, мир\n// città\r\n// 中文\r\n"
)

// utf16Bytes encodes s as UTF-16, optionally with a BOM.
func utf16Bytes(s string, littleEndian, withBOM bool) []byte {
	var out []byte
	if withBOM {
		if littleEndian {
			out = append(out, 0xFF, 0xFE)
		} else {
			out = append(out, 0xFE, 0xFF)
		}
	}
	for _, unit := range utf16.Encode([]rune(s)) {
		if littleEndian {
			out = append(out, byte(unit), byte(unit>>8))
		} else {
			out = append(out, byte(unit>>8), byte(unit))
		}
	}
	return out
}

type utf16Variant struct {
	name         string
	littleEndian bool
	withBOM      bool
	encoding     string // empty relies on BOM auto-detection
}

func utf16Variants() []utf16Variant {
	return []utf16Variant{
		{"le-bom", true, true, ""},
		{"le-nobom", true, false, "utf-16-le"},
		{"be-bom", false, true, ""},
		{"be-nobom", false, false, "utf-16-be"},
	}
}

func writeUTF16File(t *testing.T, dir, name, text string, v utf16Variant) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, utf16Bytes(text, v.littleEndian, v.withBOM), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestHandleDetectLineEndings_UTF16(t *testing.T) {
	cases := []struct {
		style          string
		text           string
		wantStyle      string
		wantTotalLines int
		wantInconsist  []int
	}{
		{"crlf", utf16SampleCRLF, LineEndingCRLF, 5, nil},
		{"lf", utf16SampleLF, LineEndingLF, 5, nil},
		{"mixed", utf16SampleMixed, LineEndingMixed, 5, []int{2}},
	}

	for _, v := range utf16Variants() {
		for _, c := range cases {
			t.Run(v.name+"/"+c.style, func(t *testing.T) {
				dir := t.TempDir()
				h := NewHandler([]string{dir})
				path := writeUTF16File(t, dir, "sample.txt", c.text, v)

				result, output, err := h.HandleDetectLineEndings(context.Background(), nil, DetectLineEndingsInput{Path: path, Encoding: v.encoding})
				if err != nil {
					t.Fatal(err)
				}
				if result.IsError {
					t.Fatal("expected success, got error")
				}
				if output.Style != c.wantStyle {
					t.Errorf("Style = %q, want %q", output.Style, c.wantStyle)
				}
				if output.TotalLines != c.wantTotalLines {
					t.Errorf("TotalLines = %d, want %d", output.TotalLines, c.wantTotalLines)
				}
				if len(output.InconsistentLines) != len(c.wantInconsist) {
					t.Fatalf("InconsistentLines = %v, want %v", output.InconsistentLines, c.wantInconsist)
				}
				for i, line := range output.InconsistentLines {
					if line != c.wantInconsist[i] {
						t.Errorf("InconsistentLines[%d] = %d, want %d", i, line, c.wantInconsist[i])
					}
				}
			})
		}
	}
}

func TestHandleChangeLineEndings_UTF16(t *testing.T) {
	cases := []struct {
		name         string
		text         string
		target       string
		wantOriginal string
		wantText     string
		wantChanged  int
	}{
		{"crlf-to-lf", utf16SampleCRLF, LineEndingLF, LineEndingCRLF, utf16SampleLF, 4},
		{"lf-to-crlf", utf16SampleLF, LineEndingCRLF, LineEndingLF, utf16SampleCRLF, 4},
		{"mixed-to-lf", utf16SampleMixed, LineEndingLF, LineEndingMixed, utf16SampleLF, 3},
		{"mixed-to-crlf", utf16SampleMixed, LineEndingCRLF, LineEndingMixed, utf16SampleCRLF, 1},
	}

	for _, v := range utf16Variants() {
		for _, c := range cases {
			t.Run(v.name+"/"+c.name, func(t *testing.T) {
				dir := t.TempDir()
				h := NewHandler([]string{dir})
				path := writeUTF16File(t, dir, "sample.txt", c.text, v)

				result, output, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{Path: path, Style: c.target, Encoding: v.encoding})
				if err != nil {
					t.Fatal(err)
				}
				if result.IsError {
					t.Fatalf("expected success, got error: %+v", result)
				}
				if output.OriginalStyle != c.wantOriginal {
					t.Errorf("OriginalStyle = %q, want %q", output.OriginalStyle, c.wantOriginal)
				}
				if output.LinesChanged != c.wantChanged {
					t.Errorf("LinesChanged = %d, want %d", output.LinesChanged, c.wantChanged)
				}

				got, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				want := utf16Bytes(c.wantText, v.littleEndian, v.withBOM)
				if !bytes.Equal(got, want) {
					t.Fatalf("converted bytes = % x\nwant                = % x", got, want)
				}
			})
		}
	}
}

func TestHandleChangeLineEndings_UTF16RoundTrip(t *testing.T) {
	for _, v := range utf16Variants() {
		t.Run(v.name, func(t *testing.T) {
			dir := t.TempDir()
			h := NewHandler([]string{dir})
			path := writeUTF16File(t, dir, "sample.txt", utf16SampleCRLF, v)
			original := utf16Bytes(utf16SampleCRLF, v.littleEndian, v.withBOM)

			for _, style := range []string{LineEndingLF, LineEndingCRLF} {
				result, _, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{Path: path, Style: style, Encoding: v.encoding})
				if err != nil {
					t.Fatal(err)
				}
				if result.IsError {
					t.Fatalf("conversion to %s failed", style)
				}
			}

			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, original) {
				t.Fatalf("round trip is not byte-identical:\ngot  % x\nwant % x", got, original)
			}
		})
	}
}

func TestHandleChangeLineEndings_UTF16Truncated(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := filepath.Join(dir, "truncated.txt")

	data := utf16Bytes(utf16SampleLF, true, true)
	if err := os.WriteFile(path, data[:len(data)-1], 0644); err != nil {
		t.Fatal(err)
	}

	result, _, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{Path: path, Style: LineEndingCRLF})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected an error for truncated UTF-16 data")
	}
}

func TestConvertUTF16LineEndings(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		target       string
		littleEndian bool
		want         string
	}{
		{"le crlf to lf", "a\r\nb\r\n", LineEndingLF, true, "a\nb\n"},
		{"le lf to crlf", "a\nb\n", LineEndingCRLF, true, "a\r\nb\r\n"},
		{"be crlf to lf", "a\r\nb\r\n", LineEndingLF, false, "a\nb\n"},
		{"be lf to crlf", "a\nb\n", LineEndingCRLF, false, "a\r\nb\r\n"},
		{"mixed to crlf", "a\r\nb\nc", LineEndingCRLF, true, "a\r\nb\r\nc"},
		{"trailing cr kept", "a\rb", LineEndingCRLF, true, "a\rb"},
		{"surrogate pair kept", "🌍\n🌍\n", LineEndingCRLF, true, "🌍\r\n🌍\r\n"},
		{"no endings", "abc", LineEndingCRLF, true, "abc"},
		{"empty", "", LineEndingCRLF, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertUTF16LineEndings(utf16Bytes(tt.input, tt.littleEndian, false), tt.target, tt.littleEndian)
			if err != nil {
				t.Fatal(err)
			}
			want := utf16Bytes(tt.want, tt.littleEndian, false)
			if !bytes.Equal(got, want) {
				t.Errorf("got % x, want % x", got, want)
			}
		})
	}

	if _, err := convertUTF16LineEndings([]byte{0x41}, LineEndingLF, true); err == nil {
		t.Error("expected an error for an odd byte length")
	}
}
