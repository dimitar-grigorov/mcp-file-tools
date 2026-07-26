package handler

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

var (
	utf8BOM    = []byte{0xEF, 0xBB, 0xBF}
	utf16LEBOM = []byte{0xFF, 0xFE}
	utf16BEBOM = []byte{0xFE, 0xFF}
)

func writeTestFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// --- parseBOMPolicy ---

func TestParseBOMPolicy(t *testing.T) {
	tests := []struct {
		in      string
		want    bomPolicy
		wantErr bool
	}{
		{"", bomAuto, false},
		{"auto", bomAuto, false},
		{" ALWAYS ", bomAlways, false},
		{"never", bomNever, false},
		{"preserve", bomPreserve, false},
		{"sometimes", "", true},
	}
	for _, tc := range tests {
		got, err := parseBOMPolicy(tc.in)
		if (err != nil) != tc.wantErr {
			t.Errorf("parseBOMPolicy(%q) err = %v, wantErr %v", tc.in, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && got != tc.want {
			t.Errorf("parseBOMPolicy(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBOMBytesForPolicy(t *testing.T) {
	utf8Had := bomInfo{HasBOM: true, Type: "utf-8"}
	utf16Had := bomInfo{HasBOM: true, Type: "utf-16-le"}

	tests := []struct {
		policy   bomPolicy
		charset  string
		existing bomInfo
		want     []byte
		wantErr  bool
	}{
		{bomAuto, "utf-8", bomInfo{}, nil, false},
		{bomAuto, "utf-8", utf8Had, utf8BOM, false}, // same flavour, keep it
		{bomAuto, "utf-8", utf16Had, nil, false},    // UTF-16 BOM was transport, not intent
		{bomAuto, "utf-16-le", bomInfo{}, utf16LEBOM, false},
		{bomAuto, "utf-16-be", bomInfo{}, utf16BEBOM, false},
		{bomAuto, "cp1251", utf8Had, nil, false}, // cp1251 has no BOM
		{bomAlways, "utf-8", bomInfo{}, utf8BOM, false},
		{bomAlways, "utf16le", bomInfo{}, utf16LEBOM, false}, // alias resolves
		{bomAlways, "cp1251", bomInfo{}, nil, true},
		{bomNever, "utf-16-le", utf16Had, nil, false},
		{bomPreserve, "utf-8", utf8Had, utf8BOM, false},
		{bomPreserve, "utf-8", bomInfo{}, nil, false},
		{bomPreserve, "utf-16-be", utf8Had, utf16BEBOM, false}, // explicit ask: keep having a BOM
		{bomPreserve, "utf-16-le", bomInfo{}, nil, false},
	}
	for _, tc := range tests {
		got, err := bomBytesForPolicy(tc.policy, tc.charset, tc.existing)
		if (err != nil) != tc.wantErr {
			t.Errorf("bomBytesForPolicy(%q, %q, %+v) err = %v, wantErr %v", tc.policy, tc.charset, tc.existing, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && !bytes.Equal(got, tc.want) {
			t.Errorf("bomBytesForPolicy(%q, %q, %+v) = % x, want % x", tc.policy, tc.charset, tc.existing, got, tc.want)
		}
	}
}

func TestCheckBOMConflict(t *testing.T) {
	tests := []struct {
		bom      bomInfo
		encoding string
		wantErr  bool
	}{
		{bomInfo{}, "cp1251", false}, // no BOM, nothing to contradict
		{bomInfo{HasBOM: true, Type: "utf-8"}, "utf-8", false},
		{bomInfo{HasBOM: true, Type: "utf-8"}, "utf8", false},        // alias
		{bomInfo{HasBOM: true, Type: "utf-16-le"}, "utf16le", false}, // alias
		{bomInfo{HasBOM: true, Type: "utf-8"}, "cp1251", true},
		{bomInfo{HasBOM: true, Type: "utf-16-le"}, "utf-16-be", true},
	}
	for _, tc := range tests {
		err := checkBOMConflict(tc.bom, tc.encoding)
		if (err != nil) != tc.wantErr {
			t.Errorf("checkBOMConflict(%+v, %q) = %v, wantErr %v", tc.bom, tc.encoding, err, tc.wantErr)
		}
	}
}

// --- write_file ---

func TestHandleWriteFile_BOMPolicies(t *testing.T) {
	tests := []struct {
		name     string
		policy   string
		encoding string
		existing []byte // nil = new file
		wantBOM  []byte
	}{
		{"auto adds BOM for utf-16-le", "auto", "utf-16-le", nil, utf16LEBOM},
		{"auto adds BOM for utf-16-be", "auto", "utf-16-be", nil, utf16BEBOM},
		{"auto leaves new utf-8 bare", "auto", "utf-8", nil, nil},
		{"auto keeps an existing utf-8 BOM", "auto", "utf-8", append(append([]byte{}, utf8BOM...), []byte("old")...), utf8BOM},
		{"never strips a utf-16 BOM", "never", "utf-16-le", nil, nil},
		{"never overrides an existing BOM", "never", "utf-8", append(append([]byte{}, utf8BOM...), []byte("old")...), nil},
		{"always adds a utf-8 BOM", "always", "utf-8", nil, utf8BOM},
		{"preserve keeps none on a bare file", "preserve", "utf-16-le", []byte("old"), nil},
		{"preserve keeps an existing BOM", "preserve", "utf-8", append(append([]byte{}, utf8BOM...), []byte("old")...), utf8BOM},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			h := NewHandler([]string{tempDir})

			path := filepath.Join(tempDir, "out.txt")
			if tc.existing != nil {
				writeTestFile(t, tempDir, "out.txt", tc.existing)
			}

			result, output, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{
				Path: path, Content: "hi", Encoding: tc.encoding, BOM: tc.policy,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatalf("expected success, got error result")
			}

			got := readTestFile(t, path)
			if len(tc.wantBOM) == 0 {
				if _, bom := splitBOM(got); bom.HasBOM {
					t.Errorf("expected no BOM, got %s (% x)", bom.Type, got)
				}
				if output.HasBOM {
					t.Errorf("output.HasBOM = true, want false")
				}
				return
			}
			if !bytes.HasPrefix(got, tc.wantBOM) {
				t.Errorf("bytes = % x, want prefix % x", got, tc.wantBOM)
			}
			if !output.HasBOM || output.BOMType != canonicalCharset(tc.encoding) {
				t.Errorf("output HasBOM=%v BOMType=%q, want true/%q", output.HasBOM, output.BOMType, canonicalCharset(tc.encoding))
			}
			// The payload after the BOM must still decode as written (no double BOM)
			if _, bom := splitBOM(got[len(tc.wantBOM):]); bom.HasBOM {
				t.Errorf("payload starts with a second BOM: % x", got)
			}
		})
	}
}

// read_text_file returns the BOM as content, so the round-trip must not double it.
func TestHandleWriteFile_ReadWriteRoundTripKeepsOneBOM(t *testing.T) {
	for _, enc := range []string{"utf-8", "utf-16-le"} {
		t.Run(enc, func(t *testing.T) {
			tempDir := t.TempDir()
			h := NewHandler([]string{tempDir})

			var original []byte
			switch enc {
			case "utf-8":
				original = append(append([]byte{}, utf8BOM...), []byte("Привет")...)
			case "utf-16-le":
				original = append(append([]byte{}, utf16LEBOM...), 'h', 0, 'i', 0)
			}
			path := writeTestFile(t, tempDir, "rt.txt", original)

			_, readOut, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{Path: path})
			if err != nil {
				t.Fatal(err)
			}

			result, writeOut, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{
				Path: path, Content: readOut.Content, Encoding: enc,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatalf("expected success, got error result")
			}
			if !writeOut.HasBOM {
				t.Error("round-trip lost the BOM")
			}
			if !bytes.Equal(readTestFile(t, path), original) {
				t.Errorf("round-trip changed bytes: % x, want % x", readTestFile(t, path), original)
			}
		})
	}
}

func TestHandleWriteFile_AlwaysOnEncodingWithoutBOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	path := filepath.Join(tempDir, "out.txt")
	result, _, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{
		Path: path, Content: "hi", Encoding: "cp1251", BOM: "always",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error for bom=always on cp1251")
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Error("file must not be created when the BOM policy cannot be satisfied")
	}
}

func TestHandleWriteFile_InvalidPolicyDoesNotWrite(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	path := writeTestFile(t, tempDir, "out.txt", []byte("keep me"))
	result, _, err := h.HandleWriteFile(context.Background(), nil, WriteFileInput{
		Path: path, Content: "overwritten", Encoding: "utf-8", BOM: "maybe",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error for an invalid bom policy")
	}
	if string(readTestFile(t, path)) != "keep me" {
		t.Error("file was modified despite an invalid bom policy")
	}
}

// --- convert_encoding ---

// Regression: a BOM used to be decoded as content, which made converting a
// BOM'd UTF-8 file to CP1251 fail outright.
func TestHandleConvertEncoding_StripsSourceUTF8BOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	path := writeTestFile(t, tempDir, "bom.txt", append(append([]byte{}, utf8BOM...), []byte("Привет")...))

	result, output, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: path, From: "utf-8", To: "cp1251",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error result")
	}
	if !output.Changed {
		t.Error("expected Changed = true")
	}
	want := []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2} // "Привет" in CP1251
	if got := readTestFile(t, path); !bytes.Equal(got, want) {
		t.Errorf("bytes = % x, want % x", got, want)
	}
}

func TestHandleConvertEncoding_StripsSourceUTF16BOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	// "hi" in UTF-16 LE with a BOM
	path := writeTestFile(t, tempDir, "u16.txt", append(append([]byte{}, utf16LEBOM...), 'h', 0, 'i', 0))

	result, _, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: path, From: "utf-16-le", To: "utf-8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error result")
	}
	if got := readTestFile(t, path); string(got) != "hi" {
		t.Errorf("content = %q (% x), want %q", got, got, "hi")
	}
}

func TestHandleConvertEncoding_BOMPolicies(t *testing.T) {
	tests := []struct {
		name    string
		policy  string
		source  []byte
		from    string
		to      string
		wantBOM []byte
	}{
		{"auto adds BOM for a utf-16 target", "auto", []byte("hi"), "utf-8", "utf-16-le", utf16LEBOM},
		{"auto keeps a utf-8 BOM", "auto", append(append([]byte{}, utf8BOM...), []byte("hi")...), "utf-8", "utf-8", utf8BOM},
		{"auto does not turn a utf-16 BOM into a utf-8 one", "auto", append(append([]byte{}, utf16LEBOM...), 'h', 0), "utf-16-le", "utf-8", nil},
		{"auto drops a BOM the target cannot hold", "auto", append(append([]byte{}, utf8BOM...), []byte("hi")...), "utf-8", "cp1251", nil},
		{"never strips the source BOM", "never", append(append([]byte{}, utf8BOM...), []byte("hi")...), "utf-8", "utf-8", nil},
		{"never keeps a utf-16 target bare", "never", []byte("hi"), "utf-8", "utf-16-be", nil},
		{"always adds a utf-8 BOM", "always", []byte("hi"), "utf-8", "utf-8", utf8BOM},
		{"preserve skips a bare source", "preserve", []byte("hi"), "utf-8", "utf-16-le", nil},
		{"preserve carries the BOM over", "preserve", append(append([]byte{}, utf8BOM...), []byte("hi")...), "utf-8", "utf-16-be", utf16BEBOM},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			h := NewHandler([]string{tempDir})

			path := writeTestFile(t, tempDir, "in.txt", tc.source)
			result, output, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
				Path: path, From: tc.from, To: tc.to, BOM: tc.policy,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatalf("expected success, got error result")
			}

			got := readTestFile(t, path)
			_, bom := splitBOM(got)
			if len(tc.wantBOM) == 0 {
				if bom.HasBOM {
					t.Errorf("expected no BOM, got %s (% x)", bom.Type, got)
				}
				if output.HasBOM {
					t.Error("output.HasBOM = true, want false")
				}
				return
			}
			if !bytes.HasPrefix(got, tc.wantBOM) {
				t.Errorf("bytes = % x, want prefix % x", got, tc.wantBOM)
			}
			if !output.HasBOM || output.BOMType != canonicalCharset(tc.to) {
				t.Errorf("output HasBOM=%v BOMType=%q, want true/%q", output.HasBOM, output.BOMType, canonicalCharset(tc.to))
			}
		})
	}
}

func TestHandleConvertEncoding_NoOpLeavesFileAlone(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	path := writeTestFile(t, tempDir, "ascii.txt", []byte("plain ascii"))
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	result, output, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: path, From: "utf-8", To: "cp1251", Backup: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error result")
	}
	if output.Changed {
		t.Error("expected Changed = false for identical target bytes")
	}
	if output.BackupPath != "" {
		t.Errorf("expected no backup for a no-op, got %q", output.BackupPath)
	}
	if _, statErr := os.Stat(path + ".bak"); statErr == nil {
		t.Error("no-op must not leave a .bak file behind")
	}

	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Error("no-op must not rewrite the file")
	}
	if string(readTestFile(t, path)) != "plain ascii" {
		t.Error("content changed on a no-op conversion")
	}
}

func TestHandleConvertEncoding_BOMConflictWithExplicitFrom(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	original := append(append([]byte{}, utf8BOM...), []byte("Привет")...)
	path := writeTestFile(t, tempDir, "conflict.txt", original)

	result, _, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: path, From: "cp1251", To: "utf-8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected a BOM/encoding conflict error")
	}
	if !bytes.Equal(readTestFile(t, path), original) {
		t.Error("file must be untouched when the BOM conflicts with 'from'")
	}
}

// An auto-detected source encoding comes from the BOM itself, so it never conflicts.
func TestHandleConvertEncoding_AutoDetectedSourceHasNoConflict(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	path := writeTestFile(t, tempDir, "detect.txt", append(append([]byte{}, utf8BOM...), []byte("Привет")...))

	result, output, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: path, To: "cp1251",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got error result")
	}
	if output.SourceEncoding != "utf-8" {
		t.Errorf("source encoding = %q, want utf-8", output.SourceEncoding)
	}
	want := []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2}
	if got := readTestFile(t, path); !bytes.Equal(got, want) {
		t.Errorf("bytes = % x, want % x", got, want)
	}
}

func TestHandleConvertEncoding_InvalidPolicyDoesNotConvert(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	original := []byte("Привет")
	path := writeTestFile(t, tempDir, "in.txt", original)

	result, _, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: path, From: "utf-8", To: "cp1251", BOM: "maybe",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected an error for an invalid bom policy")
	}
	if !bytes.Equal(readTestFile(t, path), original) {
		t.Error("file was converted despite an invalid bom policy")
	}
}
