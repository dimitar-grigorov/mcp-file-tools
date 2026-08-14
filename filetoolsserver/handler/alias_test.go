// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"encoding/json"
	"testing"
)

func aliased(t *testing.T, tool, in string) map[string]any {
	t.Helper()
	out := aliasBuiltinArgs(tool, json.RawMessage(in))
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("bad output %s: %v", out, err)
	}
	return m
}

func TestAliasGrepBuiltinNames(t *testing.T) {
	m := aliased(t, "grep_text_files",
		`{"pattern":"x","path":"src","-A":2,"-B":1,"-i":true,"-o":true,"-n":true,"head_limit":50,"output_mode":"count"}`)
	want := map[string]any{
		"contextAfter": 2.0, "contextBefore": 1.0, "caseSensitive": false,
		"matchesOnly": true, "maxMatches": 50.0, "outputMode": "count",
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("%s = %v, want %v", k, m[k], v)
		}
	}
	if p, ok := m["paths"].([]any); !ok || len(p) != 1 || p[0] != "src" {
		t.Errorf("paths = %v, want [src]", m["paths"])
	}
	for _, k := range []string{"-A", "-B", "-i", "-o", "-n", "head_limit", "output_mode", "path"} {
		if _, ok := m[k]; ok {
			t.Errorf("%s should be renamed away", k)
		}
	}
}

func TestAliasGrepContextFillsBothSides(t *testing.T) {
	m := aliased(t, "grep_text_files", `{"pattern":"x","paths":["src"],"-C":3,"contextBefore":1}`)
	if m["contextBefore"] != 1.0 || m["contextAfter"] != 3.0 {
		t.Errorf("before=%v after=%v, want 1 and 3", m["contextBefore"], m["contextAfter"])
	}
}

func TestAliasNeverClobbersCanonicalName(t *testing.T) {
	m := aliased(t, "grep_text_files", `{"pattern":"x","paths":["a"],"path":"b","head_limit":50,"maxMatches":10}`)
	if m["maxMatches"] != 10.0 {
		t.Errorf("maxMatches = %v, want 10", m["maxMatches"])
	}
	if _, ok := m["path"]; !ok {
		t.Error("ambiguous path should be left for schema validation")
	}
}

func TestAliasFlatEditShape(t *testing.T) {
	m := aliased(t, "edit_file", `{"file_path":"f.pas","old_string":"a","new_string":"b","replace_all":true}`)
	if m["path"] != "f.pas" {
		t.Errorf("path = %v", m["path"])
	}
	edits, ok := m["edits"].([]any)
	if !ok || len(edits) != 1 {
		t.Fatalf("edits = %v", m["edits"])
	}
	e := edits[0].(map[string]any)
	if e["oldText"] != "a" || e["newText"] != "b" || e["replaceAll"] != true {
		t.Errorf("edit = %v", e)
	}
}

func TestAliasFlatEditKeepsExplicitEdits(t *testing.T) {
	in := `{"path":"f","edits":[{"oldText":"a","newText":"b"}],"old_string":"x","new_string":"y"}`
	m := aliased(t, "edit_file", in)
	if _, ok := m["old_string"]; !ok {
		t.Error("old_string should stay and fail validation, not be merged")
	}
}

func TestAliasFilePathReadWrite(t *testing.T) {
	for _, tool := range []string{"read_text_file", "write_file"} {
		m := aliased(t, tool, `{"file_path":"f.txt"}`)
		if m["path"] != "f.txt" {
			t.Errorf("%s: path = %v", tool, m["path"])
		}
	}
}

func TestAliasLeavesOtherToolsAlone(t *testing.T) {
	in := json.RawMessage(`{"file_path":"f","head_limit":5}`)
	if out := aliasBuiltinArgs("tree", in); string(out) != string(in) {
		t.Errorf("tree args changed: %s", out)
	}
}
