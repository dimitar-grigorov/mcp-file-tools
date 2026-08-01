// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleConvertEncoding converts one file (path) or a batch (paths) to a target
// encoding. With dryRun it reports what would happen without touching anything.
func (h *Handler) HandleConvertEncoding(ctx context.Context, req *mcp.CallToolRequest, input ConvertEncodingInput) (*mcp.CallToolResult, ConvertEncodingOutput, error) {
	if input.To == "" {
		return errorResult("target encoding (to) is required"), ConvertEncodingOutput{}, nil
	}
	if input.Path != "" && len(input.Paths) > 0 {
		return errorResult("pass either path (one file) or paths (a batch), not both"), ConvertEncodingOutput{}, nil
	}

	batch := len(input.Paths) > 0
	paths := input.Paths
	if !batch {
		if input.Path == "" {
			return errorResult("path is required (or paths for a batch)"), ConvertEncodingOutput{}, nil
		}
		paths = []string{input.Path}
	}

	// Resolve the BOM policy and target encoding before anything mutates a file
	policy, err := parseBOMPolicy(input.BOM)
	if err != nil {
		return errorResult(err.Error()), ConvertEncodingOutput{}, nil
	}

	targetEncodingName := strings.ToLower(input.To)
	if _, ok := encoding.Get(targetEncodingName); !ok {
		return errorResult(fmt.Sprintf("unsupported target encoding: %s. Use list_encodings to see available encodings.", input.To)), ConvertEncodingOutput{}, nil
	}

	results := make([]ConvertFileResult, 0, len(paths))
	for _, p := range paths {
		if r := cancelled(ctx); r != nil {
			return r, ConvertEncodingOutput{}, nil
		}
		results = append(results, h.convertOne(p, input, policy, targetEncodingName))
	}

	output := ConvertEncodingOutput{
		TargetEncoding: targetEncodingName,
		DryRun:         input.DryRun,
	}

	// One file addressed by path keeps the flat, pre-batch output shape
	if !batch {
		r := results[0]
		if r.Error != "" {
			return errorResult(r.Error), ConvertEncodingOutput{}, nil
		}
		output.Message = r.Message
		output.SourceEncoding = r.SourceEncoding
		output.BackupPath = r.BackupPath
		output.HasBOM = r.HasBOM
		output.BOMType = r.BOMType
		output.Changed = r.Changed
		return &mcp.CallToolResult{}, output, nil
	}

	var changed, unchanged int
	for _, r := range results {
		switch {
		case r.Error != "":
			output.ErrorCount++
			output.Errors = append(output.Errors, fmt.Sprintf("%s: %s", r.Path, r.Error))
		case r.Changed:
			changed++
		default:
			unchanged++
		}
	}
	output.Results = results
	output.SuccessCount = changed + unchanged

	verb := "converted"
	if input.DryRun {
		verb = "would convert"
	}
	output.Message = fmt.Sprintf("%d of %d files %s to %s", changed, len(results), verb, targetEncodingName)
	if unchanged > 0 {
		output.Message += fmt.Sprintf("; %d already %s", unchanged, targetEncodingName)
	}
	if output.ErrorCount > 0 {
		output.Message += fmt.Sprintf("; %d failed (see results)", output.ErrorCount)
	}
	return &mcp.CallToolResult{}, output, nil
}

// convertOne converts a single file. Failures are returned in the result's Error
// field rather than aborting, so a batch can report every file.
func (h *Handler) convertOne(path string, input ConvertEncodingInput, policy bomPolicy, targetEncodingName string) ConvertFileResult {
	res := ConvertFileResult{Path: path}

	v := h.ValidatePath(path)
	if !v.Ok() {
		res.Error = v.Err.Error()
		return res
	}

	// Caller has already validated the target encoding
	targetEnc, _ := encoding.Get(targetEncodingName)

	if loadToMemory, size := h.shouldLoadEntireFile(v.Path); !loadToMemory {
		slog.Warn("loading large file into memory", "path", path, "size", size, "threshold", h.config.MemoryThreshold)
	}

	originalMode := getFileMode(v.Path)

	data, err := os.ReadFile(v.Path)
	if err != nil {
		res.Error = fmt.Sprintf("failed to read file: %v", err)
		return res
	}

	// Resolve source encoding
	var sourceEncodingName string
	if input.From != "" {
		sourceEncodingName = strings.ToLower(input.From)
		if _, ok := encoding.Get(sourceEncodingName); !ok {
			res.Error = fmt.Sprintf("unsupported source encoding: %s. Use list_encodings to see available encodings.", input.From)
			return res
		}
	} else {
		detection, _ := encoding.DetectSample(data)
		if detection.Charset == "" {
			res.Error = "could not detect source encoding. Please specify 'from' parameter."
			return res
		}
		sourceEncodingName = detection.Charset
		if _, ok := encoding.Get(sourceEncodingName); !ok {
			res.Error = fmt.Sprintf("detected encoding %s is not supported. Please specify 'from' parameter.", sourceEncodingName)
			return res
		}
	}
	res.SourceEncoding = sourceEncodingName

	// Strip any BOM before decoding — it is transport, not content
	payload, sourceBOM := splitBOM(data)
	if input.From != "" {
		if err := checkBOMConflict(sourceBOM, sourceEncodingName); err != nil {
			res.Error = err.Error()
			return res
		}
	}

	// Decode to UTF-8
	var utf8Content string
	if encoding.IsUTF8(sourceEncodingName) {
		utf8Content = string(payload)
	} else {
		sourceEnc, _ := encoding.Get(sourceEncodingName)
		decoded, err := sourceEnc.NewDecoder().Bytes(payload)
		if err != nil {
			res.Error = fmt.Sprintf("failed to decode from %s: %v", sourceEncodingName, err)
			return res
		}
		utf8Content = string(decoded)
	}

	// Encode to target
	var targetData []byte
	if encoding.IsUTF8(targetEncodingName) {
		targetData = []byte(utf8Content)
	} else {
		encoded, err := encoding.Encode(utf8Content, targetEnc, targetEncodingName)
		if err != nil {
			// Surface the offending characters as data too, so a dry run over a
			// whole tree is machine-readable and not just a wall of prose.
			var ue *encoding.UnsupportedError
			if errors.As(err, &ue) {
				res.Unsupported = ue.Runes
				res.UnsupportedCount = ue.Total
			}
			res.Error = fmt.Sprintf("failed to encode to %s: %v", targetEncodingName, err)
			return res
		}
		targetData = encoded
	}

	targetBOM, err := bomBytesForPolicy(policy, targetEncodingName, sourceBOM)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	targetData = prependBOM(targetBOM, targetData)

	res.HasBOM = len(targetBOM) > 0
	if res.HasBOM {
		res.BOMType = canonicalCharset(targetEncodingName)
	}

	// Nothing to do if the target bytes match what is already on disk
	if bytes.Equal(targetData, data) {
		res.Message = fmt.Sprintf("%s is already %s, left unchanged", path, targetEncodingName)
		return res
	}

	res.Changed = true

	if input.Backup {
		res.BackupPath = v.Path + ".bak"
	}

	if input.DryRun {
		res.Message = fmt.Sprintf("Would convert %s from %s to %s", path, sourceEncodingName, targetEncodingName)
		return res
	}

	if err := atomicWriteWithBackup(v.Path, targetData, originalMode, res.BackupPath); err != nil {
		res.Changed = false
		res.BackupPath = ""
		res.Error = fmt.Sprintf("failed to write converted file: %v", err)
		return res
	}

	res.Message = fmt.Sprintf("Successfully converted %s from %s to %s", path, sourceEncodingName, targetEncodingName)
	if res.BackupPath != "" {
		res.Message += fmt.Sprintf(" (backup: %s)", res.BackupPath)
	}
	return res
}
