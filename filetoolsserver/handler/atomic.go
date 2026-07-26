// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const tempFileSuffixBytes = 16

// atomicWriteFile writes data atomically using temp file + rename.
func atomicWriteFile(path string, data []byte, mode os.FileMode) (err error) {
	tempPath, err := generateTempPath(path)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	if err = writeTempFile(tempPath, data, mode); err != nil {
		return err
	}

	if err = os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	syncParentDir(path)
	return nil
}

// atomicWriteWithBackup writes data atomically, staging the backup as a copy so
// the target is never absent and the previous backup lives until the new one lands.
func atomicWriteWithBackup(path string, data []byte, mode os.FileMode, backupPath string) (err error) {
	tempPath, err := generateTempPath(path)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	if err = writeTempFile(tempPath, data, mode); err != nil {
		return err
	}

	if backupPath != "" {
		if err = backupCopy(path, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// A failed rename leaves the original in place, so the backup still matches it.
	if err = os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	syncParentDir(path)
	return nil
}

// backupCopy copies path to backupPath, preserving permissions and modification time.
func backupCopy(path, backupPath string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	return copyFile(path, backupPath, info.Mode().Perm(), info.ModTime())
}

// writeTempFile writes data and flushes it to disk before returning.
func writeTempFile(path string, data []byte, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	return nil
}

// syncParentDir persists the directory entry a rename just created. Best effort:
// the rename already succeeded, so a sync failure must not fail the operation.
func syncParentDir(path string) {
	if err := syncDir(filepath.Dir(path)); err != nil {
		slog.Debug("failed to sync directory after rename", "path", path, "error", err)
	}
}

// generateTempPath creates a random temp file path based on the target filepath.
func generateTempPath(filepath string) (string, error) {
	randBytes := make([]byte, tempFileSuffixBytes)
	if _, err := rand.Read(randBytes); err != nil {
		return "", fmt.Errorf("failed to generate temp filename: %w", err)
	}
	return fmt.Sprintf("%s.%s.tmp", filepath, hex.EncodeToString(randBytes)), nil
}
