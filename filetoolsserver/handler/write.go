// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"fmt"

	"github.com/dimitar-grigorov/mcp-file-tools/internal/encoding"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *Handler) HandleWriteFile(ctx context.Context, req *mcp.CallToolRequest, input WriteFileInput) (*mcp.CallToolResult, WriteFileOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, WriteFileOutput{}, nil
	}

	// Resolve the BOM policy before anything mutates the file
	policy, err := parseBOMPolicy(input.BOM)
	if err != nil {
		return errorResult(err.Error()), WriteFileOutput{}, nil
	}

	eolPolicy, err := parseLineEndingPolicy(input.LineEndings)
	if err != nil {
		return errorResult(err.Error()), WriteFileOutput{}, nil
	}

	// Resolve encoding: explicit > preserve existing > configured default
	encodingName, encodingFrom, err := h.resolveWriteEncoding(input.Encoding, v.Path)
	if err != nil {
		return errorResult(err.Error()), WriteFileOutput{}, nil
	}

	enc, _ := encoding.Get(encodingName) // Already validated by resolveWriteEncoding

	// A BOM in the content is transport (read_text_file returns it as text), not text
	// to encode — and asking for it back is what the policy decides below.
	content, contentHadBOM := trimContentBOM(input.Content)
	existing := existingBOM(v.Path)
	if contentHadBOM {
		existing = bomInfo{HasBOM: true, Type: canonicalCharset(encodingName)}
	}

	// Match the file's line endings; agents rewriting a CRLF file usually emit LF.
	eolStyle := resolveLineEndingStyle(eolPolicy, v.Path, h.config.DefaultLineEndings)
	if eolStyle != "" {
		content = ConvertLineEndings(content, eolStyle)
	}

	var contentToWrite []byte
	if encoding.IsUTF8(encodingName) {
		contentToWrite = []byte(content)
	} else {
		encoder := enc.NewEncoder()
		encoded, err := encoder.Bytes([]byte(content))
		if err != nil {
			return errorResult(fmt.Sprintf("failed to encode content: %v", err)), WriteFileOutput{}, nil
		}
		contentToWrite = encoded
	}

	bomBytes, err := bomBytesForPolicy(policy, encodingName, existing)
	if err != nil {
		return errorResult(err.Error()), WriteFileOutput{}, nil
	}
	contentToWrite = prependBOM(bomBytes, contentToWrite)

	if r := cancelled(ctx); r != nil {
		return r, WriteFileOutput{}, nil
	}

	mode := getFileMode(v.Path)
	if err := atomicWriteFile(v.Path, contentToWrite, mode); err != nil {
		return errorResult(fmt.Sprintf("failed to write file: %v", err)), WriteFileOutput{}, nil
	}

	var output WriteFileOutput
	detail := "encoding: " + encodingName
	if len(bomBytes) > 0 {
		output.HasBOM = true
		output.BOMType = canonicalCharset(encodingName)
		detail += ", with " + output.BOMType + " BOM"
	}
	output.Message = fmt.Sprintf("Successfully wrote %d bytes to %s (%s)", len(contentToWrite), input.Path, detail)
	if encodingFrom == encodingFromDefault {
		output.Message += h.utf8TransitionNotice()
	}
	return &mcp.CallToolResult{}, output, nil
}

// existingBOM reports the BOM of the file about to be overwritten; absent file means none.
func existingBOM(path string) bomInfo {
	head, err := readFileHead(path, 4)
	if err != nil {
		return bomInfo{}
	}
	_, bom := splitBOM(head)
	return bom
}
