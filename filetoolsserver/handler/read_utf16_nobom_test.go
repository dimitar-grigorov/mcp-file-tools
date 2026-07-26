package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
)

// Reading a BOM-less UTF-16 Cyrillic file with auto-detection must return the
// original text. Before structural detection, chardet reported "ascii" and the
// content came back as control-character garbage.
func TestHandleReadTextFile_BOMlessUTF16Cyrillic(t *testing.T) {
	for _, tc := range []struct {
		name string
		le   bool
	}{
		{"le", true},
		{"be", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			h := NewHandler([]string{tempDir})
			testFile := filepath.Join(tempDir, "cfg.txt")

			want := "Здравей свят!\nключ=стойност\nпорт=8080\n"
			units := utf16.Encode([]rune(want))
			data := make([]byte, 0, len(units)*2)
			for _, u := range units {
				if tc.le {
					data = append(data, byte(u), byte(u>>8))
				} else {
					data = append(data, byte(u>>8), byte(u))
				}
			}
			if err := os.WriteFile(testFile, data, 0644); err != nil {
				t.Fatal(err)
			}

			// No Encoding param: force auto-detection.
			result, output, err := h.HandleReadTextFile(context.Background(), nil, ReadTextFileInput{Path: testFile})
			if err != nil {
				t.Fatal(err)
			}
			if result.IsError {
				t.Fatalf("expected success, got error: %v", result.Content)
			}
			if output.Content != want {
				t.Errorf("auto-detect read mismatch:\n want %q\n  got %q", want, output.Content)
			}
		})
	}
}
