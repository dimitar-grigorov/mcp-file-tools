// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"fmt"
	"strings"
)

const (
	lineEndingsPreserve = "preserve" // match the existing file, else the configured default
	lineEndingsAsIs     = "asis"     // write the content byte for byte
)

// How much of an existing file we read to pick its style.
const lineEndingSampleSize = 64 * 1024

// parseLineEndingPolicy resolves the lineEndings parameter, defaulting to preserve.
func parseLineEndingPolicy(value string) (string, error) {
	policy := strings.ToLower(strings.TrimSpace(value))
	switch policy {
	case "":
		return lineEndingsPreserve, nil
	case lineEndingsPreserve, lineEndingsAsIs, LineEndingCRLF, LineEndingLF:
		return policy, nil
	default:
		return "", fmt.Errorf("%w: %q — valid: preserve, crlf, lf, asis", ErrLineEndingPolicyInvalid, value)
	}
}

// dominantLineEnding returns crlf or lf, else "". Mixed resolves to the commoner
// style, so writing over a mixed file repairs it.
func dominantLineEnding(info LineEndingInfo) string {
	switch info.Style {
	case LineEndingCRLF:
		return LineEndingCRLF
	case LineEndingLF:
		return LineEndingLF
	case LineEndingMixed:
		if info.CRLFCount >= info.LFCount {
			return LineEndingCRLF
		}
		return LineEndingLF
	}
	return ""
}

// resolveLineEndingStyle picks the style to write with, or "" to leave content alone.
// Order: explicit policy > existing file's style > configured default.
func resolveLineEndingStyle(policy, path, cfgDefault string) string {
	switch policy {
	case lineEndingsAsIs:
		return ""
	case LineEndingCRLF, LineEndingLF:
		return policy
	}

	if head, err := readFileHead(path, lineEndingSampleSize); err == nil {
		if style := dominantLineEnding(DetectLineEndings(head)); style != "" {
			return style
		}
	}
	return cfgDefault
}
