// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// 64 bits of sha256: a staleness guard, not a signature.
const contentHashBytes = 8

// minExpectedHashLen is the shortest prefix a caller may send as expectedHash.
const minExpectedHashLen = 8

// contentHash returns the short sha256 of raw file bytes, before any decoding.
func contentHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:contentHashBytes])
}

// fileContentHash hashes a file by streaming it, so a large file costs no memory.
func fileContentHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)[:contentHashBytes]), nil
}

// normalizeHash lowercases and drops an optional "sha256:" prefix.
func normalizeHash(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	return strings.TrimPrefix(h, "sha256:")
}

// hashesMatch accepts a prefix either way, so a full sha256 works as well as the short form.
func hashesMatch(expected, actual string) bool {
	expected, actual = normalizeHash(expected), normalizeHash(actual)
	if len(expected) < len(actual) {
		return strings.HasPrefix(actual, expected)
	}
	return strings.HasPrefix(expected, actual)
}

// checkExpectedHash compares expectedHash against the bytes in hand. An empty
// expectedHash means the caller opted out of the guard.
func checkExpectedHash(expected string, data []byte, path string) error {
	if expected == "" {
		return nil
	}
	if len(normalizeHash(expected)) < minExpectedHashLen {
		return fmt.Errorf("expectedHash must be at least %d hex characters; use the contentHash from read_text_file", minExpectedHashLen)
	}
	actual := contentHash(data)
	if !hashesMatch(expected, actual) {
		return staleFileError(path, expected, actual)
	}
	return nil
}

// checkExpectedHashOfFile is checkExpectedHash for a file not already read into memory.
func checkExpectedHashOfFile(expected, path string) error {
	if expected == "" {
		return nil
	}
	if len(normalizeHash(expected)) < minExpectedHashLen {
		return fmt.Errorf("expectedHash must be at least %d hex characters; use the contentHash from read_text_file", minExpectedHashLen)
	}
	actual, err := fileContentHash(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("expectedHash was given but %s does not exist; omit expectedHash to create it", path)
		}
		return fmt.Errorf("failed to hash file: %w", err)
	}
	if !hashesMatch(expected, actual) {
		return staleFileError(path, expected, actual)
	}
	return nil
}

func staleFileError(path, expected, actual string) error {
	return fmt.Errorf("file changed since you read it: expectedHash %s, file is now %s. NOTHING was changed. Re-read %s and redo the edit against its current content",
		normalizeHash(expected), actual, path)
}
