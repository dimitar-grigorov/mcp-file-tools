package handler

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleConvertEncoding converts a file from one encoding to another.
func (h *Handler) HandleConvertEncoding(ctx context.Context, req *mcp.CallToolRequest, input ConvertEncodingInput) (*mcp.CallToolResult, ConvertEncodingOutput, error) {
	// Validate required target encoding
	if input.To == "" {
		return errorResult("target encoding (to) is required"), ConvertEncodingOutput{}, nil
	}

	// Validate path
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, ConvertEncodingOutput{}, nil
	}

	// Resolve the BOM policy before anything mutates the file
	policy, err := parseBOMPolicy(input.BOM)
	if err != nil {
		return errorResult(err.Error()), ConvertEncodingOutput{}, nil
	}

	// Validate target encoding
	targetEnc, ok := encoding.Get(strings.ToLower(input.To))
	if !ok {
		return errorResult(fmt.Sprintf("unsupported target encoding: %s. Use list_encodings to see available encodings.", input.To)), ConvertEncodingOutput{}, nil
	}

	// Check file size - warn if large file will be loaded to memory
	if loadToMemory, size := h.shouldLoadEntireFile(v.Path); !loadToMemory {
		slog.Warn("loading large file into memory", "path", input.Path, "size", size, "threshold", h.config.MemoryThreshold)
	}

	// Preserve original file permissions
	originalMode := getFileMode(v.Path)

	// Read file
	data, err := os.ReadFile(v.Path)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read file: %v", err)), ConvertEncodingOutput{}, nil
	}

	// Resolve source encoding
	var sourceEncodingName string
	if input.From != "" {
		sourceEncodingName = strings.ToLower(input.From)
		_, ok := encoding.Get(sourceEncodingName)
		if !ok {
			return errorResult(fmt.Sprintf("unsupported source encoding: %s. Use list_encodings to see available encodings.", input.From)), ConvertEncodingOutput{}, nil
		}
	} else {
		// Auto-detect source encoding
		detection, _ := encoding.DetectSample(data)
		if detection.Charset == "" {
			return errorResult("could not detect source encoding. Please specify 'from' parameter."), ConvertEncodingOutput{}, nil
		}
		sourceEncodingName = detection.Charset

		// Validate detected encoding is supported
		_, ok := encoding.Get(sourceEncodingName)
		if !ok {
			return errorResult(fmt.Sprintf("detected encoding %s is not supported. Please specify 'from' parameter.", sourceEncodingName)), ConvertEncodingOutput{}, nil
		}
	}

	// Strip any BOM before decoding — it is transport, not content
	payload, sourceBOM := splitBOM(data)
	if input.From != "" {
		if err := checkBOMConflict(sourceBOM, sourceEncodingName); err != nil {
			return errorResult(err.Error()), ConvertEncodingOutput{}, nil
		}
	}

	// Decode to UTF-8
	var utf8Content string
	if encoding.IsUTF8(sourceEncodingName) {
		utf8Content = string(payload)
	} else {
		sourceEnc, _ := encoding.Get(sourceEncodingName)
		decoder := sourceEnc.NewDecoder()
		decoded, err := decoder.Bytes(payload)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to decode from %s: %v", sourceEncodingName, err)), ConvertEncodingOutput{}, nil
		}
		utf8Content = string(decoded)
	}

	// Encode to target
	var targetData []byte
	targetEncodingName := strings.ToLower(input.To)
	if encoding.IsUTF8(targetEncodingName) {
		targetData = []byte(utf8Content)
	} else {
		encoder := targetEnc.NewEncoder()
		encoded, err := encoder.Bytes([]byte(utf8Content))
		if err != nil {
			return errorResult(fmt.Sprintf("failed to encode to %s: %v", targetEncodingName, err)), ConvertEncodingOutput{}, nil
		}
		targetData = encoded
	}

	targetBOM, err := bomBytesForPolicy(policy, targetEncodingName, sourceBOM)
	if err != nil {
		return errorResult(err.Error()), ConvertEncodingOutput{}, nil
	}
	targetData = prependBOM(targetBOM, targetData)

	output := ConvertEncodingOutput{
		SourceEncoding: sourceEncodingName,
		TargetEncoding: targetEncodingName,
		HasBOM:         len(targetBOM) > 0,
	}
	if output.HasBOM {
		output.BOMType = canonicalCharset(targetEncodingName)
	}

	// Nothing to do if the target bytes match what is already on disk
	if bytes.Equal(targetData, data) {
		output.Message = fmt.Sprintf("%s is already %s, left unchanged", input.Path, targetEncodingName)
		return &mcp.CallToolResult{}, output, nil
	}

	var backupPath string
	if input.Backup {
		backupPath = v.Path + ".bak"
	}

	if r := cancelled(ctx); r != nil {
		return r, ConvertEncodingOutput{}, nil
	}

	if err := atomicWriteWithBackup(v.Path, targetData, originalMode, backupPath); err != nil {
		return errorResult(fmt.Sprintf("failed to write converted file: %v", err)), ConvertEncodingOutput{}, nil
	}

	message := fmt.Sprintf("Successfully converted %s from %s to %s", input.Path, sourceEncodingName, targetEncodingName)
	if backupPath != "" {
		message += fmt.Sprintf(" (backup: %s)", backupPath)
	}

	output.Message = message
	output.BackupPath = backupPath
	output.Changed = true
	return &mcp.CallToolResult{}, output, nil
}
