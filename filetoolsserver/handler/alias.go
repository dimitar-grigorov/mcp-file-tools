// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Models habitually send the parameter shapes of the client's built-in
// Read/Write/Edit/Grep tools. AliasBuiltinParams translates those shapes to
// ours before schema validation, only where the semantics match exactly.
// The schema advertises canonical names only; a canonical name always wins
// over its alias, and anything untranslatable still fails loudly.
func AliasBuiltinParams(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		if r, ok := req.(*mcp.CallToolRequest); ok && r.Params != nil {
			r.Params.Arguments = aliasBuiltinArgs(r.Params.Name, r.Params.Arguments)
		}
		return next(ctx, method, req)
	}
}

func aliasBuiltinArgs(tool string, raw json.RawMessage) json.RawMessage {
	switch tool {
	case "read_text_file", "write_file", "edit_file", "grep_text_files":
	default:
		return raw
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil || m == nil {
		return raw
	}
	if tool == "grep_text_files" {
		aliasGrepArgs(m)
	} else {
		renameArg(m, "file_path", "path")
		if tool == "edit_file" {
			aliasFlatEdit(m)
		}
	}
	if out, err := json.Marshal(m); err == nil {
		return out
	}
	return raw
}

// renameArg moves from to to; an explicit to always wins.
func renameArg(m map[string]json.RawMessage, from, to string) {
	if v, ok := m[from]; ok {
		delete(m, from)
		setIfAbsent(m, to, v)
	}
}

func setIfAbsent(m map[string]json.RawMessage, key string, v json.RawMessage) {
	if _, dup := m[key]; !dup {
		m[key] = v
	}
}

// aliasFlatEdit turns Edit's flat old_string/new_string/replace_all into a
// one-entry edits array. With edits or patch present the flat keys stay, so
// validation fails loudly instead of merging two shapes.
func aliasFlatEdit(m map[string]json.RawMessage) {
	_, hasEdits := m["edits"]
	_, hasPatch := m["patch"]
	oldS, okOld := m["old_string"]
	newS, okNew := m["new_string"]
	if hasEdits || hasPatch || !okOld || !okNew {
		return
	}
	edit := map[string]json.RawMessage{"oldText": oldS, "newText": newS}
	if ra, ok := m["replace_all"]; ok {
		edit["replaceAll"] = ra
		delete(m, "replace_all")
	}
	if arr, err := json.Marshal([]map[string]json.RawMessage{edit}); err == nil {
		delete(m, "old_string")
		delete(m, "new_string")
		m["edits"] = arr
	}
}

// aliasGrepArgs maps built-in Grep parameter names onto ours.
func aliasGrepArgs(m map[string]json.RawMessage) {
	renameArg(m, "-B", "contextBefore")
	renameArg(m, "-A", "contextAfter")
	renameArg(m, "-o", "matchesOnly")
	renameArg(m, "head_limit", "maxMatches")
	renameArg(m, "output_mode", "outputMode")
	delete(m, "-n") // matches always carry line numbers
	// -i inverts to caseSensitive.
	if v, ok := m["-i"]; ok {
		delete(m, "-i")
		var insensitive bool
		if _, dup := m["caseSensitive"]; !dup && json.Unmarshal(v, &insensitive) == nil {
			m["caseSensitive"], _ = json.Marshal(!insensitive)
		}
	}
	// -C / context fill both sides.
	for _, k := range []string{"-C", "context"} {
		if v, ok := m[k]; ok {
			delete(m, k)
			setIfAbsent(m, "contextBefore", v)
			setIfAbsent(m, "contextAfter", v)
		}
	}
	// Built-in Grep takes one path string; ours takes paths.
	if v, ok := m["path"]; ok {
		var s string
		if _, dup := m["paths"]; !dup && json.Unmarshal(v, &s) == nil {
			m["paths"], _ = json.Marshal([]string{s})
			delete(m, "path")
		}
	}
}
