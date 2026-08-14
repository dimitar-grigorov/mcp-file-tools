// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/wlynxg/chardet"
)

// Past the third guess the confidences are noise.
const maxCandidates = 3

// Candidate is one guess at what the bytes are, with the detector's confidence.
type Candidate struct {
	Charset    string
	Confidence int
	Supported  bool // in the registry, so usable as an encoding parameter
}

// candidates ranks what the bytes could be, best first. Nil once a BOM or the
// UTF-16 classifier decided: nothing to choose between.
func candidates(data []byte) []Candidate {
	if _, ok := DetectBOM(data); ok {
		return nil
	}
	if mayContainUTF16(data) {
		if _, handled := detectUTF16(data); handled {
			return nil
		}
	}

	ranked := make([]Candidate, 0, maxCandidates)
	seen := make(map[string]bool)
	for _, result := range chardet.DetectAll(data) {
		charset, confidence := correctCharset(strings.ToLower(result.Encoding), int(result.Confidence*100), data)
		if charset == "" {
			continue
		}
		// The registry's own name, so the agent can pass it straight back.
		supported := false
		if canonical, ok := Canonical(charset); ok {
			charset, supported = canonical, true
		}
		if seen[charset] {
			continue
		}
		seen[charset] = true

		ranked = append(ranked, Candidate{Charset: charset, Confidence: confidence, Supported: supported})
		if len(ranked) == maxCandidates {
			break
		}
	}
	return ranked
}

// CandidatesFromSample ranks over the samples DetectSample reads: one bounded pass.
func CandidatesFromSample(data []byte) []Candidate {
	if len(data) <= SmallFileThreshold {
		return candidates(data)
	}
	return candidates(joinDetectionSamples(detectionSamplesFromData(data)))
}

// CandidatesFromFile ranks over the bytes the detector saw: all in "full" mode, samples otherwise.
func CandidatesFromFile(path string, mode string) ([]Candidate, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	size := stat.Size()
	if mode == "full" || size <= SmallFileThreshold {
		data := make([]byte, size)
		if _, err := file.ReadAt(data, 0); err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
		return candidates(data), nil
	}

	samples, err := readDetectionSamples(file, size)
	if err != nil {
		return nil, err
	}
	return candidates(joinDetectionSamples(samples)), nil
}

// SupportedAlternatives keeps what a caller could retry with: in the registry, and not `exclude`.
func SupportedAlternatives(ranked []Candidate, exclude string) []Candidate {
	alternatives := make([]Candidate, 0, len(ranked))
	for _, candidate := range ranked {
		if !candidate.Supported || SameCharset(candidate.Charset, exclude) {
			continue
		}
		alternatives = append(alternatives, candidate)
	}
	return alternatives
}

// FormatCandidates renders a ranked list as "windows-1251 (62%), koi8-r (55%)".
func FormatCandidates(ranked []Candidate) string {
	parts := make([]string, 0, len(ranked))
	for _, candidate := range ranked {
		parts = append(parts, fmt.Sprintf("%s (%d%%)", candidate.Charset, candidate.Confidence))
	}
	return strings.Join(parts, ", ")
}

// SameCharset compares through the registry, so "ascii" and "utf-8" are one answer.
func SameCharset(a, b string) bool {
	if strings.EqualFold(a, b) {
		return true
	}
	canonicalA, okA := Canonical(a)
	canonicalB, okB := Canonical(b)
	return okA && okB && canonicalA == canonicalB
}
