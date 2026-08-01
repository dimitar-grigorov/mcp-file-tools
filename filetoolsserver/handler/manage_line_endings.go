// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HandleManageLineEndings dispatches to detection or conversion, mirroring manage_bom.
func (h *Handler) HandleManageLineEndings(ctx context.Context, req *mcp.CallToolRequest, input ManageLineEndingsInput) (*mcp.CallToolResult, ManageLineEndingsOutput, error) {
	switch strings.ToLower(strings.TrimSpace(input.Action)) {
	case "detect":
		res, out, err := h.HandleDetectLineEndings(ctx, req, DetectLineEndingsInput{
			Path: input.Path, Encoding: input.Encoding,
		})
		if err != nil || res.IsError {
			return res, ManageLineEndingsOutput{}, err
		}
		return res, ManageLineEndingsOutput{
			Style:             out.Style,
			TotalLines:        out.TotalLines,
			InconsistentLines: out.InconsistentLines,
		}, nil

	case "convert":
		if input.Style == "" {
			return errorResult(ErrLineEndingStyleRequired.Error()), ManageLineEndingsOutput{}, nil
		}
		res, out, err := h.HandleChangeLineEndings(ctx, req, ChangeLineEndingsInput{
			Path: input.Path, Style: input.Style, Encoding: input.Encoding,
		})
		if err != nil || res.IsError {
			return res, ManageLineEndingsOutput{}, err
		}
		return res, ManageLineEndingsOutput{
			Style:         out.NewStyle,
			OriginalStyle: out.OriginalStyle,
			LinesChanged:  out.LinesChanged,
			Message:       out.Message,
			Changed:       out.OriginalStyle != out.NewStyle,
		}, nil
	}

	return errorResult(fmt.Sprintf("%v: %q", ErrLineEndingActionInvalid, input.Action)), ManageLineEndingsOutput{}, nil
}
