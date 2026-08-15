// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"unicode/utf8"
)

// Detection is statistical and no tuning removes its blind spots: Spanish CP1252 like
// "MÓDULO FÍSICAMENTE ÚNICO" is plausible GBK, every uppercase accent before an ASCII
// letter being a valid hanzi pair. Pinning the answer set makes the guess stop mattering.

// pinnedConfidence is what a pin decides with: validation cannot rank, so a flat value above the trust threshold is the honest answer.
const pinnedConfidence = 75

// allowedCharsets is the pinned set in priority order, nil when unrestricted. Set at startup, read on every detect.
var allowedCharsets atomic.Pointer[[]string]

// SetDetectionCandidates restricts detection to these encodings; an empty list clears the restriction.
func SetDetectionCandidates(names []string) error {
	if len(names) == 0 {
		allowedCharsets.Store(nil)
		return nil
	}
	canonical := make([]string, 0, len(names))
	for _, name := range names {
		c, ok := Canonical(name)
		if !ok {
			return fmt.Errorf("unknown encoding %q", name)
		}
		canonical = append(canonical, c)
	}
	allowedCharsets.Store(&canonical)
	return nil
}

// DetectionCandidates returns the pinned set, nil when detection is unrestricted.
func DetectionCandidates() []string {
	if pinned := allowedCharsets.Load(); pinned != nil {
		return *pinned
	}
	return nil
}

// charsetAllowed reports whether detection may answer charset. BOM verdicts never come here: a BOM is a declaration, not a guess.
func charsetAllowed(charset string) bool {
	pinned := DetectionCandidates()
	if len(pinned) == 0 {
		return true
	}
	canonical, ok := Canonical(charset)
	return ok && slices.Contains(pinned, canonical)
}

// pinnedVerdict replaces a ruled-out guess with the first candidate that decodes data. Nothing fitting means no answer, not a wrong one.
func pinnedVerdict(data []byte) DetectionResult {
	for _, charset := range DetectionCandidates() {
		if decodesCleanly(charset, data) {
			return DetectionResult{Charset: charset, Confidence: pinnedConfidence}
		}
	}
	return DetectionResult{}
}

// decodesCleanly reports whether data survives charset without a replacement character.
func decodesCleanly(charset string, data []byte) bool {
	switch {
	case IsUTF8(charset):
		return utf8.Valid(data)
	case strings.HasPrefix(charset, "utf-16"), strings.HasPrefix(charset, "utf-32"):
		// These swallow almost any bytes, so only a BOM or the structural classifier may name them.
		return false
	}
	// A single-byte table needs no decode: an undefined byte maps to U+FFFD.
	if cm := charmapFor(charset); cm != nil {
		for _, b := range data {
			if cm.DecodeByte(b) == utf8.RuneError {
				return false
			}
		}
		return true
	}
	enc, ok := Get(charset)
	if !ok {
		return false
	}
	decoded, err := enc.NewDecoder().Bytes(data)
	return err == nil && !bytes.ContainsRune(decoded, utf8.RuneError)
}
