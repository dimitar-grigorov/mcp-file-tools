// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The backup must be a copy: if the original is renamed away instead, the target
// is briefly absent and a crash in that window loses the file.
func TestAtomicWriteWithBackup_KeepsTargetInPlace(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	backupPath := target + ".bak"
	original := []byte("original content")

	if err := os.WriteFile(target, original, 0644); err != nil {
		t.Fatal(err)
	}

	// A hard link keeps a handle on the original file object, so os.SameFile can
	// tell a copied backup from the original renamed away.
	probe := filepath.Join(dir, "probe.link")
	if err := os.Link(target, probe); err != nil {
		t.Skipf("hard links unsupported here: %v", err)
	}

	if err := atomicWriteWithBackup(target, []byte("updated content"), 0644, backupPath); err != nil {
		t.Fatal(err)
	}

	probeInfo, err := os.Stat(probe)
	if err != nil {
		t.Fatal(err)
	}
	backupInfo, err := os.Stat(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(probeInfo, backupInfo) {
		t.Error("backup is the original file moved away; the target was briefly missing")
	}

	got, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Errorf("backup = %q, want %q", got, original)
	}
	got, err = os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "updated content" {
		t.Errorf("target = %q, want %q", got, "updated content")
	}
}

// A stale backup must survive until the new one is committed, then be replaced.
func TestAtomicWriteWithBackup_ReplacesStaleBackup(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	backupPath := target + ".bak"
	original := []byte("original content")

	if err := os.WriteFile(target, original, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupPath, []byte("stale backup"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := atomicWriteWithBackup(target, []byte("updated"), 0644, backupPath); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Errorf("backup = %q, want %q", got, original)
	}
}

func TestAtomicWriteFile_LeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")

	if err := atomicWriteFile(target, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "file.txt" {
		t.Errorf("directory contains %v, want only file.txt", entries)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "content" {
		t.Errorf("target = %q, want %q", got, "content")
	}
}

func TestHandleConvertEncoding_BackupIsOriginal(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := filepath.Join(dir, "convert.txt")
	original := []byte("Привет")

	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}

	result, output, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path:   path,
		From:   "utf-8",
		To:     "cp1251",
		Backup: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("expected success, got %v", result.Content)
	}
	if output.BackupPath == "" {
		t.Fatal("no backup path reported")
	}

	backup, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(backup, original) {
		t.Errorf("backup = %q, want the original bytes", backup)
	}
}

// A repeat run with an explicit from decodes the converted file again; replacing the .bak then would lose the original.
func TestHandleConvertEncoding_ExistingBackupIsNeverReplaced(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := filepath.Join(dir, "convert.txt")
	original := []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2} // "Привет" in cp1251
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}

	for i, dryRun := range []bool{false, true, false} {
		result, _, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
			Path: path, From: "cp1251", To: "utf-8", Backup: true, DryRun: dryRun,
		})
		if err != nil {
			t.Fatal(err)
		}
		message := extractTextFromResult(result.Content)
		if i == 0 && result.IsError {
			t.Fatalf("first conversion failed: %s", message)
		}
		if i > 0 && (!result.IsError || !strings.Contains(message, "already exists")) {
			t.Errorf("call %d (dryRun=%t) should refuse over the existing backup, got %q", i+1, dryRun, message)
		}
	}

	backup, _ := os.ReadFile(path + ".bak")
	if !bytes.Equal(backup, original) {
		t.Errorf("backup = % x, want the original cp1251 bytes", backup)
	}
	if got, _ := os.ReadFile(path); string(got) != "Привет" {
		t.Errorf("file = %q, want one conversion, not two", got)
	}
}

// os.Stat follows links, so a dangling .bak link would look absent and be replaced.
func TestHandleConvertEncoding_DanglingBackupLinkIsNeverReplaced(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := filepath.Join(dir, "convert.txt")
	if err := os.WriteFile(path, []byte{0xCF, 0xF0, 0xE8, 0xE2, 0xE5, 0xF2}, 0644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "gone")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	createDirLink(t, target, path+".bak")
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}

	result, _, err := h.HandleConvertEncoding(context.Background(), nil, ConvertEncodingInput{
		Path: path, From: "cp1251", To: "utf-8", Backup: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if message := extractTextFromResult(result.Content); !result.IsError || !strings.Contains(message, "already exists") {
		t.Errorf("expected the dangling backup link to be refused as existing, got %q", message)
	}
	if info, err := os.Lstat(path + ".bak"); err != nil {
		t.Errorf("backup link should be untouched: %v", err)
	} else if info.Mode().IsRegular() {
		t.Error("backup link was replaced by a regular file")
	}
}

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestHandleWriteFile_CancelledLeavesFileUnchanged(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(path, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	result, _, err := h.HandleWriteFile(cancelledContext(), nil, WriteFileInput{
		Path:     path,
		Content:  "replacement",
		Encoding: "utf-8",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected cancellation error")
	}
	if data, _ := os.ReadFile(path); string(data) != "original" {
		t.Errorf("file = %q, want %q", data, "original")
	}
}

func TestHandleCopyFile_CancelledLeavesDestinationMissing(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	source := filepath.Join(dir, "source.txt")
	destination := filepath.Join(dir, "destination.txt")
	if err := os.WriteFile(source, []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}

	result, _, err := h.HandleCopyFile(cancelledContext(), nil, CopyFileInput{Source: source, Destination: destination})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected cancellation error")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Errorf("destination should not exist, stat error = %v", err)
	}
}

func TestHandleMoveFile_CancelledLeavesSourceUnchanged(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	source := filepath.Join(dir, "source.txt")
	destination := filepath.Join(dir, "destination.txt")
	if err := os.WriteFile(source, []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}

	result, _, err := h.HandleMoveFile(cancelledContext(), nil, MoveFileInput{Source: source, Destination: destination})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected cancellation error")
	}
	if data, err := os.ReadFile(source); err != nil || string(data) != "source" {
		t.Errorf("source = %q, err = %v; want %q", data, err, "source")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Errorf("destination should not exist, stat error = %v", err)
	}
}

// os.Stat follows symlinks, so a dangling link at the destination used to look absent.
func TestHandleMoveFile_RefusesDanglingSymlinkDestination(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	source := filepath.Join(dir, "source.txt")
	destination := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(source, []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "missing.txt"), destination); err != nil {
		t.Skipf("symlinks unsupported here: %v", err)
	}

	result, _, err := h.HandleMoveFile(context.Background(), nil, MoveFileInput{Source: source, Destination: destination})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected the existing destination to be refused")
	}
	if data, err := os.ReadFile(source); err != nil || string(data) != "source" {
		t.Errorf("source = %q, err = %v; want %q", data, err, "source")
	}
	if _, err := os.Lstat(destination); err != nil {
		t.Errorf("destination symlink should be untouched: %v", err)
	}
}

func TestHandleDeleteFile_CancelledLeavesFileUnchanged(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(path, []byte("target"), 0644); err != nil {
		t.Fatal(err)
	}

	result, _, err := h.HandleDeleteFile(cancelledContext(), nil, DeleteFileInput{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected cancellation error")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "target" {
		t.Errorf("file = %q, err = %v; want %q", data, err, "target")
	}
}

func TestHandleManageBom_CancelledLeavesFileUnchanged(t *testing.T) {
	dir := t.TempDir()
	h := NewHandler([]string{dir})
	path := filepath.Join(dir, "target.txt")
	original := append([]byte{0xEF, 0xBB, 0xBF}, []byte("target")...)
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}

	result, _, err := h.HandleManageBom(cancelledContext(), nil, ManageBomInput{Path: path, Action: "strip"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatal("expected cancellation error")
	}
	actual, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, original) {
		t.Error("cancelled BOM operation changed the file")
	}
}
