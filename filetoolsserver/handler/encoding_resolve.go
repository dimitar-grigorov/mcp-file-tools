// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/config"
	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/encoding"
	textEncoding "golang.org/x/text/encoding"
)

// Shared by every tool that touches encoding: read, write, edit, grep, convert.

// encodingResult is the outcome of resolving which encoding to read a file with.
type encodingResult struct {
	encoder            textEncoding.Encoding
	name               string
	detectedEncoding   string
	encodingConfidence int
	autoDetected       bool
	fallbackHint       string // set when a detected encoding was discarded
}

// validateEncodingName lowercases a requested encoding and rejects unknown ones.
func validateEncodingName(name string) (string, error) {
	lower := strings.ToLower(name)
	if _, ok := encoding.Get(lower); !ok {
		return "", fmt.Errorf("%w: %s. Use list_encodings to see available encodings", ErrEncodingUnsupported, lower)
	}
	return lower, nil
}

// fallbackEncoding is what every resolver uses once detection decides nothing.
func (h *Handler) fallbackEncoding() string {
	if h.config.DefaultEncoding == "" {
		return config.DefaultEncoding
	}
	return h.config.DefaultEncoding
}

// encodingSource says which branch of resolveWriteEncoding produced the name.
type encodingSource int

const (
	encodingFromRequest  encodingSource = iota // explicit encoding parameter
	encodingFromExisting                       // detected on the file being overwritten
	encodingFromDefault                        // configured default, file did not exist
	encodingFromFallback                       // configured default, existing file was inconclusive
)

// resolveWriteEncoding returns encoding for writes: explicit > existing file > config default.
func (h *Handler) resolveWriteEncoding(inputEncoding string, filePath string) (string, encodingSource, error) {
	if inputEncoding != "" {
		name, err := validateEncodingName(inputEncoding)
		if err != nil {
			return "", encodingFromRequest, err
		}
		return name, encodingFromRequest, nil
	}

	fileExists := false
	if _, err := os.Stat(filePath); err == nil {
		fileExists = true
		detected, err := encoding.DetectFromFile(filePath, "sample")
		if err == nil && detected.Conclusive() {
			slog.Debug("preserving existing file encoding", "path", filePath, "encoding", detected.Charset, "confidence", detected.Confidence)
			return detected.Charset, encodingFromExisting, nil
		}
		slog.Debug("encoding detection inconclusive, using default", "path", filePath, "detected", detected.Charset, "confidence", detected.Confidence)
	}

	if fileExists {
		return h.fallbackEncoding(), encodingFromFallback, nil
	}
	return h.fallbackEncoding(), encodingFromDefault, nil
}

// resolveEncodingFromData returns encoding from loaded data: explicit > auto-detect.
func (h *Handler) resolveEncodingFromData(inputEncoding string, data []byte, filePath string) (string, error) {
	if inputEncoding != "" {
		return validateEncodingName(inputEncoding)
	}

	detected := encoding.Detect(data)
	if detected.Conclusive() {
		slog.Debug("auto-detected encoding from data", "path", filePath, "encoding", detected.Charset, "confidence", detected.Confidence)
		return detected.Charset, nil
	}

	slog.Debug("encoding detection inconclusive, using configured default", "path", filePath, "detected", detected.Charset, "confidence", detected.Confidence, "default", h.fallbackEncoding())
	return h.fallbackEncoding(), nil
}

// resolveEncoding returns explicit encoding or auto-detects based on file size.
func (h *Handler) resolveEncoding(inputEncoding string, filePath string) (encodingResult, error) {
	result := encodingResult{}

	if inputEncoding != "" {
		name, err := validateEncodingName(inputEncoding)
		if err != nil {
			return result, err
		}
		result.name = name
		result.encoder, _ = encoding.Get(name)
		return result, nil
	}

	// Sample a file too large to hold in memory; read the rest in full.
	detectionMode := "full"
	if loadToMemory, _ := h.shouldLoadEntireFile(filePath); !loadToMemory {
		detectionMode = "sample"
	}

	// Inconclusive detection uses the configured default, as writes already do.
	fallback := h.fallbackEncoding()

	result.autoDetected = true
	detection, err := encoding.DetectFromFile(filePath, detectionMode)
	if err != nil {
		result.setFallback(fallback, "detection failed, using "+fallback)
		return result, nil
	}
	result.detectedEncoding = detection.Charset
	result.encodingConfidence = detection.Confidence

	trusted := detection.Confidence >= encoding.MinConfidenceThreshold
	if trusted && detection.Charset != "" {
		result.name = detection.Charset
	} else {
		note := ""
		if detection.Charset != "" {
			note = detection.Charset + " (low confidence, using " + fallback + ")"
		}
		result.setFallback(fallback, note)
		if alternatives := alternativeEncodings(filePath, detectionMode, fallback); alternatives != "" {
			result.fallbackHint = fmt.Sprintf(
				"Encoding detection was inconclusive, so the file was read as %s and non-ASCII text may be garbled — tell the user. "+
					"If it looks wrong, retry read_text_file with encoding set to one of: %s.",
				fallback, alternatives)
		}
	}

	enc, ok := encoding.Get(result.name)
	if !ok {
		slog.Warn("detected encoding not supported", "path", filePath, "detected", detection.Charset,
			"confidence", detection.Confidence, "fallback", fallback)
		result.setFallback(fallback, result.detectedEncoding+" (unsupported, using "+fallback+")")
		// Phrased as an instruction because models relay instructions and ignore trivia.
		result.fallbackHint = fmt.Sprintf(
			"Detected encoding %s is not supported, so the file was read as %s and non-ASCII text may be garbled — tell the user. "+
				"If it looks wrong, retry read_text_file with an explicit encoding.",
			detection.Charset, fallback)
		if alternatives := alternativeEncodings(filePath, detectionMode, detection.Charset); alternatives != "" {
			result.fallbackHint += " Ranked alternatives: " + alternatives + "."
		}
	} else {
		result.encoder = enc
	}

	return result, nil
}

// alternativeEncodings ranks what to retry with; empty when nothing to suggest.
func alternativeEncodings(filePath string, mode string, exclude string) string {
	ranked, err := encoding.CandidatesFromFile(filePath, mode)
	if err != nil {
		return ""
	}
	return encoding.FormatCandidates(encoding.SupportedAlternatives(ranked, exclude))
}

// alternativesSuffix appends the same list to an error, so a refusal says what to try.
func alternativesSuffix(data []byte, exclude string) string {
	alternatives := encoding.FormatCandidates(encoding.SupportedAlternatives(encoding.CandidatesFromSample(data), exclude))
	if alternatives == "" {
		return ""
	}
	return " Other candidates: " + alternatives + "."
}

// setFallback switches to name and resolves its encoder; note replaces the report.
func (r *encodingResult) setFallback(name, note string) {
	r.name = name
	r.encoder, _ = encoding.Get(name) // nil for utf-8, which decodeContent expects
	if note != "" {
		r.detectedEncoding = note
	}
}

// decodeContent decodes file data to UTF-8 using the resolved encoding.
func decodeContent(data []byte, encResult encodingResult) (string, error) {
	return encoding.Decode(data, encResult.name)
}
