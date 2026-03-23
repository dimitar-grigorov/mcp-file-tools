package handler

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleManageBom_DetectUTF8BOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "utf8bom.txt")

	// UTF-8 BOM + content
	data := append([]byte{0xEF, 0xBB, 0xBF}, []byte("Hello")...)
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	input := ManageBomInput{Path: testFile, Action: "detect"}
	result, output, err := h.HandleManageBom(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if !output.HasBOM {
		t.Error("expected HasBOM=true")
	}
	if output.BOMType != "utf-8" {
		t.Errorf("expected bomType=utf-8, got %s", output.BOMType)
	}
	if output.BOMBytes != 3 {
		t.Errorf("expected bomBytes=3, got %d", output.BOMBytes)
	}
	if output.Changed {
		t.Error("detect should not change the file")
	}
}

func TestHandleManageBom_DetectUTF16LEBOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "utf16le.txt")

	// UTF-16 LE BOM + some non-null content (to avoid UTF-32 LE match)
	data := append([]byte{0xFF, 0xFE}, []byte{0x48, 0x00, 0x69, 0x00}...)
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	input := ManageBomInput{Path: testFile, Action: "detect"}
	_, output, _ := h.HandleManageBom(context.Background(), nil, input)

	if !output.HasBOM {
		t.Error("expected HasBOM=true")
	}
	if output.BOMType != "utf-16-le" {
		t.Errorf("expected bomType=utf-16-le, got %s", output.BOMType)
	}
	if output.BOMBytes != 2 {
		t.Errorf("expected bomBytes=2, got %d", output.BOMBytes)
	}
}

func TestHandleManageBom_DetectUTF16BEBOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "utf16be.txt")

	data := append([]byte{0xFE, 0xFF}, []byte{0x00, 0x48, 0x00, 0x69}...)
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	input := ManageBomInput{Path: testFile, Action: "detect"}
	_, output, _ := h.HandleManageBom(context.Background(), nil, input)

	if !output.HasBOM {
		t.Error("expected HasBOM=true")
	}
	if output.BOMType != "utf-16-be" {
		t.Errorf("expected bomType=utf-16-be, got %s", output.BOMType)
	}
	if output.BOMBytes != 2 {
		t.Errorf("expected bomBytes=2, got %d", output.BOMBytes)
	}
}

func TestHandleManageBom_DetectUTF32LEBOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "utf32le.txt")

	// UTF-32 LE BOM: FF FE 00 00
	data := append([]byte{0xFF, 0xFE, 0x00, 0x00}, []byte{0x48, 0x00, 0x00, 0x00}...)
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	input := ManageBomInput{Path: testFile, Action: "detect"}
	_, output, _ := h.HandleManageBom(context.Background(), nil, input)

	if !output.HasBOM {
		t.Error("expected HasBOM=true")
	}
	if output.BOMType != "utf-32-le" {
		t.Errorf("expected bomType=utf-32-le, got %s", output.BOMType)
	}
	if output.BOMBytes != 4 {
		t.Errorf("expected bomBytes=4, got %d", output.BOMBytes)
	}
}

func TestHandleManageBom_DetectNoBOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "nobom.txt")

	if err := os.WriteFile(testFile, []byte("plain ASCII content"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ManageBomInput{Path: testFile, Action: "detect"}
	_, output, _ := h.HandleManageBom(context.Background(), nil, input)

	if output.HasBOM {
		t.Error("expected HasBOM=false for file without BOM")
	}
	if output.BOMType != "" {
		t.Errorf("expected empty bomType, got %s", output.BOMType)
	}
}

func TestHandleManageBom_DetectEmptyFile(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "empty.txt")

	if err := os.WriteFile(testFile, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	input := ManageBomInput{Path: testFile, Action: "detect"}
	_, output, _ := h.HandleManageBom(context.Background(), nil, input)

	if output.HasBOM {
		t.Error("expected HasBOM=false for empty file")
	}
}

func TestHandleManageBom_StripUTF8BOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "strip.txt")

	content := []byte("Hello, world!")
	data := append([]byte{0xEF, 0xBB, 0xBF}, content...)
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	input := ManageBomInput{Path: testFile, Action: "strip"}
	result, output, err := h.HandleManageBom(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if !output.Changed {
		t.Error("expected Changed=true")
	}
	if output.BOMType != "utf-8" {
		t.Errorf("expected bomType=utf-8, got %s", output.BOMType)
	}
	if output.BOMBytes != 3 {
		t.Errorf("expected bomBytes=3, got %d", output.BOMBytes)
	}

	// Verify file content — BOM removed, content preserved
	result_data, _ := os.ReadFile(testFile)
	if !bytes.Equal(result_data, content) {
		t.Errorf("expected content %q, got %q", content, result_data)
	}
}

func TestHandleManageBom_StripUTF16LEBOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "strip16.txt")

	content := []byte{0x48, 0x00, 0x69, 0x00} // "Hi" in UTF-16 LE
	data := append([]byte{0xFF, 0xFE}, content...)
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	input := ManageBomInput{Path: testFile, Action: "strip"}
	_, output, _ := h.HandleManageBom(context.Background(), nil, input)

	if !output.Changed {
		t.Error("expected Changed=true")
	}
	if output.BOMBytes != 2 {
		t.Errorf("expected bomBytes=2, got %d", output.BOMBytes)
	}

	result_data, _ := os.ReadFile(testFile)
	if !bytes.Equal(result_data, content) {
		t.Errorf("expected content without BOM, got %x", result_data)
	}
}

func TestHandleManageBom_StripNoBOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "nobom.txt")

	content := []byte("no BOM here")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	input := ManageBomInput{Path: testFile, Action: "strip"}
	_, output, _ := h.HandleManageBom(context.Background(), nil, input)

	if output.Changed {
		t.Error("expected Changed=false when no BOM to strip")
	}
	if output.HasBOM {
		t.Error("expected HasBOM=false")
	}

	// File should be unchanged
	result_data, _ := os.ReadFile(testFile)
	if !bytes.Equal(result_data, content) {
		t.Errorf("file content should be unchanged")
	}
}

func TestHandleManageBom_AddUTF8BOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "add.txt")

	content := []byte("Hello, world!")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	input := ManageBomInput{Path: testFile, Action: "add", Encoding: "utf-8"}
	result, output, err := h.HandleManageBom(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatal("expected success")
	}
	if !output.Changed {
		t.Error("expected Changed=true")
	}
	if !output.HasBOM {
		t.Error("expected HasBOM=true")
	}
	if output.BOMType != "utf-8" {
		t.Errorf("expected bomType=utf-8, got %s", output.BOMType)
	}
	if output.BOMBytes != 3 {
		t.Errorf("expected bomBytes=3, got %d", output.BOMBytes)
	}

	// Verify file starts with UTF-8 BOM followed by original content
	result_data, _ := os.ReadFile(testFile)
	expected := append([]byte{0xEF, 0xBB, 0xBF}, content...)
	if !bytes.Equal(result_data, expected) {
		t.Errorf("expected %x, got %x", expected, result_data)
	}
}

func TestHandleManageBom_AddUTF16BEBOM(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "add16be.txt")

	content := []byte{0x00, 0x48, 0x00, 0x69} // "Hi" in UTF-16 BE
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	input := ManageBomInput{Path: testFile, Action: "add", Encoding: "utf-16-be"}
	_, output, _ := h.HandleManageBom(context.Background(), nil, input)

	if !output.Changed {
		t.Error("expected Changed=true")
	}
	if output.BOMBytes != 2 {
		t.Errorf("expected bomBytes=2, got %d", output.BOMBytes)
	}

	result_data, _ := os.ReadFile(testFile)
	expected := append([]byte{0xFE, 0xFF}, content...)
	if !bytes.Equal(result_data, expected) {
		t.Errorf("expected %x, got %x", expected, result_data)
	}
}

func TestHandleManageBom_AddFailsIfBOMExists(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "hasbom.txt")

	// File already has UTF-8 BOM
	data := append([]byte{0xEF, 0xBB, 0xBF}, []byte("content")...)
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	input := ManageBomInput{Path: testFile, Action: "add", Encoding: "utf-8"}
	result, _, _ := h.HandleManageBom(context.Background(), nil, input)

	if !result.IsError {
		t.Error("expected error when adding BOM to file that already has one")
	}

	// File should be unchanged
	result_data, _ := os.ReadFile(testFile)
	if !bytes.Equal(result_data, data) {
		t.Error("file should not be modified when add fails")
	}
}

func TestHandleManageBom_AddMissingEncoding(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ManageBomInput{Path: testFile, Action: "add"}
	result, _, _ := h.HandleManageBom(context.Background(), nil, input)

	if !result.IsError {
		t.Error("expected error when encoding is missing for add action")
	}
}

func TestHandleManageBom_AddInvalidEncoding(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ManageBomInput{Path: testFile, Action: "add", Encoding: "cp1251"}
	result, _, _ := h.HandleManageBom(context.Background(), nil, input)

	if !result.IsError {
		t.Error("expected error for non-Unicode encoding")
	}
}

func TestHandleManageBom_InvalidAction(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	input := ManageBomInput{Path: testFile, Action: "remove"}
	result, _, _ := h.HandleManageBom(context.Background(), nil, input)

	if !result.IsError {
		t.Error("expected error for invalid action")
	}
}

func TestHandleManageBom_StripThenAdd(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "roundtrip.txt")

	// Start with UTF-8 BOM
	content := []byte("Round-trip test")
	original := append([]byte{0xEF, 0xBB, 0xBF}, content...)
	if err := os.WriteFile(testFile, original, 0644); err != nil {
		t.Fatal(err)
	}

	// Strip
	stripInput := ManageBomInput{Path: testFile, Action: "strip"}
	_, stripOut, _ := h.HandleManageBom(context.Background(), nil, stripInput)
	if !stripOut.Changed {
		t.Fatal("strip should change file")
	}

	// Verify stripped
	stripped, _ := os.ReadFile(testFile)
	if !bytes.Equal(stripped, content) {
		t.Fatalf("after strip expected %x, got %x", content, stripped)
	}

	// Add UTF-16 LE BOM (different from original)
	addInput := ManageBomInput{Path: testFile, Action: "add", Encoding: "utf-16-le"}
	_, addOut, _ := h.HandleManageBom(context.Background(), nil, addInput)
	if !addOut.Changed {
		t.Fatal("add should change file")
	}
	if addOut.BOMType != "utf-16-le" {
		t.Errorf("expected bomType=utf-16-le, got %s", addOut.BOMType)
	}

	// Verify new BOM
	final, _ := os.ReadFile(testFile)
	expected := append([]byte{0xFF, 0xFE}, content...)
	if !bytes.Equal(final, expected) {
		t.Errorf("after add expected %x, got %x", expected, final)
	}
}

func TestHandleManageBom_BOMOnlyFile(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})
	testFile := filepath.Join(tempDir, "bomonly.txt")

	// File is just a UTF-8 BOM, no content
	if err := os.WriteFile(testFile, []byte{0xEF, 0xBB, 0xBF}, 0644); err != nil {
		t.Fatal(err)
	}

	// Detect
	_, detectOut, _ := h.HandleManageBom(context.Background(), nil, ManageBomInput{Path: testFile, Action: "detect"})
	if !detectOut.HasBOM {
		t.Error("expected BOM detected on BOM-only file")
	}

	// Strip should produce empty file
	_, stripOut, _ := h.HandleManageBom(context.Background(), nil, ManageBomInput{Path: testFile, Action: "strip"})
	if !stripOut.Changed {
		t.Error("expected Changed=true")
	}

	data, _ := os.ReadFile(testFile)
	if len(data) != 0 {
		t.Errorf("expected empty file after stripping BOM-only file, got %d bytes", len(data))
	}
}
