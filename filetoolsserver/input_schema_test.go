// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package filetoolsserver

import (
	"encoding/json"
	"testing"
)

// Strict clients validate tools/list and need "properties" even on a tool without parameters.
func TestInputSchemasHaveProperties(t *testing.T) {
	for _, tool := range listTools(t) {
		raw, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatal(err)
		}
		var s struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(raw, &s); err != nil {
			t.Fatal(err)
		}
		if s.Properties == nil {
			t.Errorf("%s: inputSchema has no properties: %s", tool.Name, raw)
		}
	}
}
