package handler

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// skipOnWindows skips the test on Windows where Unix permissions don't apply
func skipOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission test on Windows")
	}
}

func TestWriteFile_PreservesPermissions(t *testing.T) {
	skipOnWindows(t)

	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")

	// Create file with specific permissions (0600 = owner read/write only)
	if err := os.WriteFile(testFile, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}

	// Verify original permissions
	info, _ := os.Stat(testFile)
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600, got %o", info.Mode().Perm())
	}

	// Write new content using handler
	input := WriteFileInput{
		Path:     testFile,
		Content:  "modified",
		Encoding: "utf-8",
	}

	result, _, err := h.HandleWriteFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success")
	}

	// Verify permissions are preserved
	info, _ = os.Stat(testFile)
	if info.Mode().Perm() != 0600 {
		t.Errorf("permissions should be preserved, expected 0600, got %o", info.Mode().Perm())
	}
}

func TestWriteFile_NewFileUsesDefaultPermissions(t *testing.T) {
	skipOnWindows(t)

	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "newfile.txt")

	// Write to new file
	input := WriteFileInput{
		Path:     testFile,
		Content:  "new content",
		Encoding: "utf-8",
	}

	result, _, err := h.HandleWriteFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success")
	}

	// Verify default permissions (0644)
	info, _ := os.Stat(testFile)
	if info.Mode().Perm() != DefaultFileMode {
		t.Errorf("new file should have default permissions, expected %o, got %o", DefaultFileMode, info.Mode().Perm())
	}
}

func TestEditFile_PreservesPermissions(t *testing.T) {
	skipOnWindows(t)

	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")

	// Create file with specific permissions (0755 = rwxr-xr-x)
	if err := os.WriteFile(testFile, []byte("hello world"), 0755); err != nil {
		t.Fatal(err)
	}

	// Edit file
	input := EditFileInput{
		Path:  testFile,
		Edits: []EditOperation{{OldText: "world", NewText: "go"}},
	}

	result, _, err := h.HandleEditFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success")
	}

	// Verify permissions are preserved
	info, _ := os.Stat(testFile)
	if info.Mode().Perm() != 0755 {
		t.Errorf("permissions should be preserved, expected 0755, got %o", info.Mode().Perm())
	}
}

func TestConvertEncoding_PreservesPermissions(t *testing.T) {
	skipOnWindows(t)

	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")

	// Create file with specific permissions
	if err := os.WriteFile(testFile, []byte("hello"), 0640); err != nil {
		t.Fatal(err)
	}

	// Convert encoding
	input := ConvertEncodingInput{
		Path: testFile,
		To:   "utf-8",
	}

	result, _, err := h.HandleConvertEncoding(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success")
	}

	// Verify permissions are preserved
	info, _ := os.Stat(testFile)
	if info.Mode().Perm() != 0640 {
		t.Errorf("permissions should be preserved, expected 0640, got %o", info.Mode().Perm())
	}
}

func TestConvertEncoding_BackupPreservesPermissions(t *testing.T) {
	skipOnWindows(t)

	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	testFile := filepath.Join(tempDir, "test.txt")

	// Create file with specific permissions
	if err := os.WriteFile(testFile, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}

	// Convert encoding with backup
	input := ConvertEncodingInput{
		Path:   testFile,
		To:     "utf-8",
		Backup: true,
	}

	result, output, err := h.HandleConvertEncoding(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success")
	}

	// Verify backup file has original permissions
	backupInfo, err := os.Stat(output.BackupPath)
	if err != nil {
		t.Fatal(err)
	}
	if backupInfo.Mode().Perm() != 0600 {
		t.Errorf("backup permissions should match original, expected 0600, got %o", backupInfo.Mode().Perm())
	}
}

func TestCopyFile_PreservesPermissions(t *testing.T) {
	skipOnWindows(t)

	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	srcFile := filepath.Join(tempDir, "source.txt")
	dstFile := filepath.Join(tempDir, "dest.txt")

	// Create source with specific permissions
	if err := os.WriteFile(srcFile, []byte("content"), 0751); err != nil {
		t.Fatal(err)
	}

	// Copy file
	input := CopyFileInput{
		Source:      srcFile,
		Destination: dstFile,
	}

	result, _, err := h.HandleCopyFile(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("expected success")
	}

	// Verify destination has source permissions
	dstInfo, _ := os.Stat(dstFile)
	if dstInfo.Mode().Perm() != 0751 {
		t.Errorf("copy should preserve permissions, expected 0751, got %o", dstInfo.Mode().Perm())
	}
}

func TestGetFileMode_ExistingFile(t *testing.T) {
	skipOnWindows(t)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create file with specific permissions
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}

	mode := getFileMode(testFile)
	if mode != 0600 {
		t.Errorf("expected 0600, got %o", mode)
	}
}

func TestGetFileMode_NonExistentFile(t *testing.T) {
	mode := getFileMode("/nonexistent/path/file.txt")
	if mode != DefaultFileMode {
		t.Errorf("expected default mode %o, got %o", DefaultFileMode, mode)
	}
}
