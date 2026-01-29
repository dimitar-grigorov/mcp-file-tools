package handler

import (
	"context"
	"strings"
	"testing"
)

func TestHandleListEncodings(t *testing.T) {
	tempDir := t.TempDir()
	h := NewHandler([]string{tempDir})

	input := ListEncodingsInput{}

	result, output, err := h.HandleListEncodings(context.Background(), nil, input)
	if err != nil {
		t.Fatal(err)
	}

	if result.IsError {
		t.Errorf("expected success, got error: %v", result.Content)
	}

	// Check encodings list
	if len(output.Encodings) == 0 {
		t.Fatal("expected encodings list, got empty")
	}

	text := strings.Join(output.Encodings, " ")

	// Check for UTF-8
	if !strings.Contains(text, "utf-8") {
		t.Errorf("expected utf-8 in encodings list, got %q", text)
	}

	// Check for CP1251
	if !strings.Contains(text, "cp1251") {
		t.Errorf("expected cp1251 in encodings list, got %q", text)
	}
}
