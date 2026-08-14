// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleDetectEncoding detects the encoding of a file
func (h *Handler) HandleDetectEncoding(ctx context.Context, req *mcp.CallToolRequest, input DetectEncodingInput) (*mcp.CallToolResult, DetectEncodingOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, DetectEncodingOutput{}, nil
	}

	mode := input.Mode
	if mode == "" {
		mode = "sample"
	}

	result, err := encoding.DetectFromFile(v.Path, mode)
	if err != nil {
		return errorResult(err.Error()), DetectEncodingOutput{}, nil
	}

	if result.Charset == "" {
		return errorResult("could not detect encoding"), DetectEncodingOutput{}, nil
	}

	output := DetectEncodingOutput{
		Encoding:   result.Charset,
		Confidence: result.Confidence,
		HasBOM:     result.HasBOM,
	}
	if inDoubt(result) {
		output.Candidates = rankedCandidates(v.Path, mode, result.Charset)
	}

	return &mcp.CallToolResult{}, output, nil
}

// inDoubt: a low-confidence guess, or a charset the registry cannot read.
// Anything else is settled, and ranking costs a second detection pass.
func inDoubt(result encoding.DetectionResult) bool {
	if result.HasBOM {
		return false
	}
	if result.Confidence < encoding.HighConfidenceThreshold {
		return true
	}
	_, supported := encoding.Get(result.Charset)
	return !supported
}

// rankedCandidates is nil when the detector's one answer is the one reported.
func rankedCandidates(path string, mode string, detected string) []EncodingCandidate {
	ranked, err := encoding.CandidatesFromFile(path, mode)
	if err != nil {
		return nil
	}
	if len(ranked) == 1 && encoding.SameCharset(ranked[0].Charset, detected) {
		return nil
	}

	out := make([]EncodingCandidate, 0, len(ranked))
	for _, candidate := range ranked {
		out = append(out, EncodingCandidate{
			Encoding:   candidate.Charset,
			Confidence: candidate.Confidence,
			Supported:  candidate.Supported,
		})
	}
	return out
}
