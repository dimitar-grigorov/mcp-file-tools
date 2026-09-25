// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"fmt"
	"slices"
	"sync/atomic"
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

// pinnedVerdict picks the pinned candidate that reads most like text, else the first that decodes: the pin says the files are in one of these.
func pinnedVerdict(data []byte) DetectionResult {
	best, fallback := "", ""
	bestShare := 0.0
	for _, charset := range DetectionCandidates() {
		p := scoreCharset(charset, data)
		if !p.readable {
			continue
		}
		if fallback == "" {
			fallback = charset
		}
		if p.plausible() && p.share() > bestShare {
			best, bestShare = charset, p.share()
		}
	}
	if best == "" {
		best = fallback
	}
	if best == "" {
		return DetectionResult{}
	}
	return DetectionResult{Charset: best, Confidence: pinnedConfidence}
}
