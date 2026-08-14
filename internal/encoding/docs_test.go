// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The Supported Encodings table in TOOLS.md is hand-maintained, and an encoding
// added to the registry without a row there is invisible to every caller who
// reads the docs instead of calling list_encodings. MacCyrillic shipped that way
// in 3.4.0; this keeps the two in step.
func TestToolsDocListsEveryEncoding(t *testing.T) {
	path := filepath.Join("..", "..", "TOOLS.md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	doc := string(data)

	for _, item := range ListEncodings() {
		if !strings.Contains(doc, "| "+item.Name+" |") {
			t.Errorf("TOOLS.md has no Supported Encodings row for %q", item.Name)
		}
		for _, alias := range item.Aliases {
			if !strings.Contains(doc, alias) {
				t.Errorf("TOOLS.md does not mention alias %q of %q", alias, item.Name)
			}
		}
	}
}
