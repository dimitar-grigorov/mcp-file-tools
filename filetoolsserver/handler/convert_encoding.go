package handler

import (
	"context"
	"fmt"
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

	// Validate target encoding
	targetEnc, ok := encoding.Get(strings.ToLower(input.To))
	if !ok {
		return errorResult(fmt.Sprintf("unsupported target encoding: %s. Use list_encodings to see available encodings.", input.To)), ConvertEncodingOutput{}, nil
	}

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

	// Decode to UTF-8
	var utf8Content string
	if encoding.IsUTF8(sourceEncodingName) {
		utf8Content = string(data)
	} else {
		sourceEnc, _ := encoding.Get(sourceEncodingName)
		decoder := sourceEnc.NewDecoder()
		decoded, err := decoder.Bytes(data)
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

	// Create backup if requested
	var backupPath string
	if input.Backup {
		backupPath = v.Path + ".bak"
		if err := os.WriteFile(backupPath, data, 0644); err != nil {
			return errorResult(fmt.Sprintf("failed to create backup: %v", err)), ConvertEncodingOutput{}, nil
		}
	}

	// Write converted file
	if err := os.WriteFile(v.Path, targetData, 0644); err != nil {
		return errorResult(fmt.Sprintf("failed to write converted file: %v", err)), ConvertEncodingOutput{}, nil
	}

	message := fmt.Sprintf("Successfully converted %s from %s to %s", input.Path, sourceEncodingName, targetEncodingName)
	if backupPath != "" {
		message += fmt.Sprintf(" (backup: %s)", backupPath)
	}

	return &mcp.CallToolResult{}, ConvertEncodingOutput{
		Message:        message,
		SourceEncoding: sourceEncodingName,
		TargetEncoding: targetEncodingName,
		BackupPath:     backupPath,
	}, nil
}
