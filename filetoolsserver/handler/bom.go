// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"fmt"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/v4/internal/encoding"
)

// bomPolicy decides whether a file we write carries a Unicode BOM.
type bomPolicy string

const (
	// bomAuto adds a BOM for UTF-16 targets and keeps one of the same encoding.
	bomAuto     bomPolicy = "auto"
	bomAlways   bomPolicy = "always"
	bomNever    bomPolicy = "never"
	bomPreserve bomPolicy = "preserve"
)

// contentBOM is the BOM as a UTF-8 text rune, which is how it reaches us in content.
const contentBOM = "\uFEFF"

// bomInfo describes a BOM found at the start of a file.
type bomInfo struct {
	HasBOM bool
	Type   string // canonical charset, e.g. "utf-16-le"
}

// parseBOMPolicy resolves the bom parameter, defaulting to auto when empty.
func parseBOMPolicy(value string) (bomPolicy, error) {
	policy := bomPolicy(strings.ToLower(strings.TrimSpace(value)))
	switch policy {
	case "":
		return bomAuto, nil
	case bomAuto, bomAlways, bomNever, bomPreserve:
		return policy, nil
	default:
		return "", fmt.Errorf("%w: %q — valid: auto, always, never, preserve", ErrBOMPolicyInvalid, value)
	}
}

// splitBOM separates a leading BOM from the payload.
func splitBOM(data []byte) ([]byte, bomInfo) {
	result, found := encoding.DetectBOM(data)
	if !found {
		return data, bomInfo{}
	}
	return data[encoding.BOMSize(result.Charset):], bomInfo{HasBOM: true, Type: result.Charset}
}

// checkBOMConflict rejects a BOM that contradicts an explicitly requested encoding,
// e.g. a UTF-8 BOM on a file the caller asked to read as CP1251.
func checkBOMConflict(bom bomInfo, encodingName string) error {
	if !bom.HasBOM {
		return nil
	}
	if canonicalCharset(encodingName) == bom.Type {
		return nil
	}
	return fmt.Errorf("%w: file starts with a %s BOM but encoding %s was requested", ErrBOMEncodingConflict, bom.Type, encodingName)
}

// bomBytesForPolicy returns the BOM to prepend when writing charset under policy.
// existing is the BOM of the file being replaced (its source, for a conversion).
func bomBytesForPolicy(policy bomPolicy, charset string, existing bomInfo) ([]byte, error) {
	canonical := canonicalCharset(charset)
	utf16Target := canonical == "utf-16-le" || canonical == "utf-16-be"
	want := false
	switch policy {
	case bomNever:
		return nil, nil
	case bomAlways:
		want = true
	case bomPreserve:
		want = existing.HasBOM
	case bomAuto:
		// UTF-16 needs a BOM to stay detectable; otherwise only keep a BOM of the
		// same flavour, since a UTF-16 BOM is transport rather than intent.
		want = utf16Target || (existing.HasBOM && existing.Type == canonical)
	default:
		return nil, fmt.Errorf("%w: %q", ErrBOMPolicyInvalid, policy)
	}

	if !want {
		return nil, nil
	}
	bom := encoding.BOMBytesFor(canonical)
	if len(bom) == 0 {
		// Only "always" can demand a BOM the encoding does not define; the rest fall back to none.
		if policy == bomAlways {
			return nil, fmt.Errorf("%w: bom=%q but encoding %s has no BOM", ErrBOMPolicyInvalid, policy, canonical)
		}
		return nil, nil
	}
	return bom, nil
}

// trimContentBOM drops a leading U+FEFF from UTF-8 content. read_text_file hands
// the BOM back as text, so without this a read/write round-trip doubles it.
func trimContentBOM(content string) (string, bool) {
	if strings.HasPrefix(content, contentBOM) {
		return strings.TrimPrefix(content, contentBOM), true
	}
	return content, false
}

// canonicalCharset resolves aliases so BOM lookups and comparisons agree.
func canonicalCharset(name string) string {
	if canonical, ok := encoding.Canonical(name); ok {
		return canonical
	}
	return strings.ToLower(name)
}

// prependBOM returns bom followed by payload, without touching payload.
func prependBOM(bom, payload []byte) []byte {
	if len(bom) == 0 {
		return payload
	}
	out := make([]byte, 0, len(bom)+len(payload))
	out = append(out, bom...)
	return append(out, payload...)
}
