package handler

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHandleListEncodings(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	params := &mcp.CallToolParamsFor[ListEncodingsInput]{
		Arguments: ListEncodingsInput{},
	}

	result, err := h.HandleListEncodings(context.Background(), nil, params)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	text := extractText(result.Content)

	// Check for UTF-8
	if !strings.Contains(text, "utf-8") {
		t.Errorf("expected utf-8 in encodings list, got %q", text)
	}

	// Check for CP1251
	if !strings.Contains(text, "cp1251") {
		t.Errorf("expected cp1251 in encodings list, got %q", text)
	}
}
