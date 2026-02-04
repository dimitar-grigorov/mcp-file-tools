package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
)

// atomicWriteFile writes data atomically using temp file + rename.
func atomicWriteFile(filepath string, data []byte, mode os.FileMode) (err error) {
	tempPath, err := generateTempPath(filepath)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	if err = os.WriteFile(tempPath, data, mode); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err = os.Rename(tempPath, filepath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// atomicWriteWithBackup writes data atomically with optional backup (write-ahead pattern).
func atomicWriteWithBackup(filepath string, data []byte, mode os.FileMode, backupPath string) (err error) {
	tempPath, err := generateTempPath(filepath)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			os.Remove(tempPath)
		}
	}()

	if err = os.WriteFile(tempPath, data, mode); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if backupPath != "" {
		os.Remove(backupPath)
		if err = os.Rename(filepath, backupPath); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	if err = os.Rename(tempPath, filepath); err != nil {
		if backupPath != "" {
			if restoreErr := os.Rename(backupPath, filepath); restoreErr != nil {
				slog.Error("failed to restore backup after rename failure", "backup", backupPath, "error", restoreErr)
			}
		}
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// generateTempPath creates a random temp file path based on the target filepath.
func generateTempPath(filepath string) (string, error) {
	randBytes := make([]byte, 16)
	if _, err := rand.Read(randBytes); err != nil {
		return "", fmt.Errorf("failed to generate temp filename: %w", err)
	}
	return fmt.Sprintf("%s.%s.tmp", filepath, hex.EncodeToString(randBytes)), nil
}
