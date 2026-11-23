// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package filetoolsserver

import (
	"context"
	"encoding/json"
	"testing"
)

// Strict OpenAI-compatible clients (Cline + LM Studio) validate tools/list
// with Zod and require "properties" on every inputSchema, even when the tool
// takes no arguments. A nil Properties map is omitted from the wire format,
// which used to break list_allowed_directories and list_encodings.
func TestEveryToolInputSchemaHasProperties(t *testing.T) {
	cs := sessionIn(t, t.TempDir())
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tools) != 20 {
		t.Fatalf("tools/list returned %d tools, want 20", len(res.Tools))
	}

	for _, tool := range res.Tools {
		t.Run(tool.Name, func(t *testing.T) {
			schema, err := json.Marshal(tool.InputSchema)
			if err != nil {
				t.Fatalf("marshaling inputSchema: %v", err)
			}
			var s struct {
				Type       string          `json:"type"`
				Properties json.RawMessage `json:"properties"`
			}
			if err := json.Unmarshal(schema, &s); err != nil {
				t.Fatalf("unmarshaling inputSchema %s: %v", schema, err)
			}
			if s.Type != "object" {
				t.Errorf("inputSchema type = %q, want \"object\"", s.Type)
			}
			if len(s.Properties) == 0 {
				t.Fatalf("inputSchema has no \"properties\" object: %s", schema)
			}
			if s.Properties[0] != '{' {
				t.Errorf("properties is not a JSON object: %s", s.Properties)
			}
			if tool.Name == "list_allowed_directories" || tool.Name == "list_encodings" {
				// These tools take no arguments: properties must be the empty object.
				if got := string(s.Properties); got != "{}" {
					t.Errorf("properties = %s, want {}", got)
				}
			}
		})
	}
}
