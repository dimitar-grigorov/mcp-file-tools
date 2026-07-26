package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Unmodified upstream files; manifest.json records their provenance.
const realFixtureDir = "testdata/line_endings_real"

type realFixture struct {
	Encoding           string `json:"encoding"`
	File               string `json:"file"`
	LicenseFile        string `json:"license_file"`
	SHA256             string `json:"sha256"`
	ByteLength         int    `json:"byte_length"`
	BOM                string `json:"bom"`
	ExpectedStyle      string `json:"expected_style"`
	CRLFCount          int    `json:"crlf_count"`
	LFCount            int    `json:"lf_count"`
	ExpectedTotalLines int    `json:"expected_total_lines"`
}

func loadRealFixtures(t *testing.T) []realFixture {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(realFixtureDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Fixtures []realFixture `json:"fixtures"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Fixtures) == 0 {
		t.Fatal("fixture manifest is empty")
	}
	return manifest.Fixtures
}

// readRealFixture reads a fixture and checks it against the manifest.
func readRealFixture(t *testing.T, f realFixture) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(realFixtureDir, f.File))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != f.ByteLength {
		t.Fatalf("%s byte length = %d, want %d", f.File, len(data), f.ByteLength)
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != f.SHA256 {
		t.Fatalf("%s SHA-256 = %s, want %s", f.File, got, f.SHA256)
	}
	if _, err := os.Stat(filepath.Join(realFixtureDir, f.LicenseFile)); err != nil {
		t.Fatalf("%s license file: %v", f.File, err)
	}
	return data
}

func TestRealUTF16FixturesDetectLineEndings(t *testing.T) {
	fixtureDir, err := filepath.Abs(realFixtureDir)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler([]string{fixtureDir})

	for _, f := range loadRealFixtures(t) {
		t.Run(f.Encoding, func(t *testing.T) {
			readRealFixture(t, f)

			// BOM-less needs an explicit encoding.
			requested := f.Encoding
			if f.BOM != "none" {
				requested = ""
			}

			result, output, err := h.HandleDetectLineEndings(context.Background(), nil, DetectLineEndingsInput{
				Path:     filepath.Join(fixtureDir, f.File),
				Encoding: requested,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatalf("detect_line_endings failed for the real %s fixture", f.Encoding)
			}
			if output.Style != f.ExpectedStyle {
				t.Errorf("Style = %q, want %q", output.Style, f.ExpectedStyle)
			}
			if output.TotalLines != f.ExpectedTotalLines {
				t.Errorf("TotalLines = %d, want %d", output.TotalLines, f.ExpectedTotalLines)
			}
			if len(output.InconsistentLines) != 0 {
				t.Errorf("InconsistentLines = %v, want []", output.InconsistentLines)
			}
		})
	}
}

func TestRealUTF16FixturesChangeLineEndingsRoundTrip(t *testing.T) {
	for _, f := range loadRealFixtures(t) {
		t.Run(f.Encoding, func(t *testing.T) {
			original := readRealFixture(t, f)
			if f.ExpectedStyle != LineEndingLF {
				t.Fatalf("fixture style %q is not what this test assumes", f.ExpectedStyle)
			}

			dir := t.TempDir()
			h := NewHandler([]string{dir})
			path := filepath.Join(dir, f.File)
			if err := os.WriteFile(path, original, 0644); err != nil {
				t.Fatal(err)
			}

			requested := f.Encoding
			if f.BOM != "none" {
				requested = ""
			}

			result, output, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{
				Path:     path,
				Style:    LineEndingCRLF,
				Encoding: requested,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatalf("change_line_endings failed for the real %s fixture", f.Encoding)
			}
			if output.OriginalStyle != LineEndingLF || output.LinesChanged != f.LFCount {
				t.Errorf("conversion = %s -> %s, %d lines; want lf -> crlf, %d lines",
					output.OriginalStyle, output.NewStyle, output.LinesChanged, f.LFCount)
			}

			_, detected, err := h.HandleDetectLineEndings(context.Background(), nil, DetectLineEndingsInput{Path: path, Encoding: requested})
			if err != nil {
				t.Fatal(err)
			}
			if detected.Style != LineEndingCRLF {
				t.Errorf("converted style = %q, want crlf", detected.Style)
			}
			if detected.TotalLines != f.ExpectedTotalLines {
				t.Errorf("converted TotalLines = %d, want %d", detected.TotalLines, f.ExpectedTotalLines)
			}

			if _, _, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{
				Path:     path,
				Style:    LineEndingLF,
				Encoding: requested,
			}); err != nil {
				t.Fatal(err)
			}

			roundTripped, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(roundTripped, original) {
				t.Errorf("real %s fixture is not byte-identical after a line-ending round trip", f.Encoding)
			}
		})
	}
}

// MetaTrader sources: the real-world case behind the UTF-16 fix.
func mqlFixtures() []struct {
	file       string
	style      string
	totalLines int
} {
	return []struct {
		file       string
		style      string
		totalLines int
	}{
		{"localized_utf16le_crlf.mq5", LineEndingCRLF, 6},
		{"localized_utf16le_crlf.mqh", LineEndingCRLF, 8},
		{"localized_utf8_lf.mq5", LineEndingLF, 6},
	}
}

func TestMQLFixturesLineEndingsRoundTrip(t *testing.T) {
	for _, f := range mqlFixtures() {
		t.Run(f.file, func(t *testing.T) {
			original, err := os.ReadFile(filepath.Join("testdata/mql", f.file))
			if err != nil {
				t.Fatal(err)
			}

			dir := t.TempDir()
			h := NewHandler([]string{dir})
			path := filepath.Join(dir, f.file)
			if err := os.WriteFile(path, original, 0644); err != nil {
				t.Fatal(err)
			}

			// Auto-detection only.
			result, output, err := h.HandleDetectLineEndings(context.Background(), nil, DetectLineEndingsInput{Path: path})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatalf("detect_line_endings failed for %s", f.file)
			}
			if output.Style != f.style || output.TotalLines != f.totalLines {
				t.Errorf("detected %s / %d lines, want %s / %d", output.Style, output.TotalLines, f.style, f.totalLines)
			}

			_, before, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{Path: path})
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"città", "Привет", "中文", "🌍"} {
				if !strings.Contains(before.Content, want) {
					t.Fatalf("fixture is missing %q", want)
				}
			}

			other := LineEndingCRLF
			if f.style == LineEndingCRLF {
				other = LineEndingLF
			}
			for _, style := range []string{other, f.style} {
				result, _, err := h.HandleChangeLineEndings(context.Background(), nil, ChangeLineEndingsInput{Path: path, Style: style})
				if err != nil {
					t.Fatal(err)
				}
				if result.IsError {
					t.Fatalf("conversion to %s failed", style)
				}
			}

			roundTripped, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(roundTripped, original) {
				t.Errorf("%s is not byte-identical after a line-ending round trip", f.file)
			}
		})
	}
}
