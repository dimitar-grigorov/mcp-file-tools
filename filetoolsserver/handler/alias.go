// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"encoding/json"
	"strconv"
	"strings"
)

// aliasBuiltinArgs maps built-in shapes and guessed names with the exact same meaning; a canonical name always wins.
func aliasBuiltinArgs(tool string, raw json.RawMessage) json.RawMessage {
	switch tool {
	case "read_text_file", "write_file", "edit_file", "grep_text_files",
		"manage_line_endings", "convert_encoding", "tree":
	default:
		return raw
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil || m == nil {
		return raw
	}
	switch tool {
	case "grep_text_files":
		aliasGrepArgs(m)
	case "read_text_file":
		renameArg(m, "file_path", "path")
		renameArg(m, "head", "limit")
		renameArg(m, "maxLines", "limit")
		aliasLineRange(m)
	case "write_file":
		renameArg(m, "file_path", "path")
		renameArg(m, "lineEnding", "lineEndings")
	case "edit_file":
		renameArg(m, "file_path", "path")
		aliasFlatEdit(m, "old_string", "new_string", "replace_all")
		aliasFlatEdit(m, "oldText", "newText", "replaceAll")
	case "manage_line_endings":
		renameArg(m, "lineEnding", "style")
		renameArg(m, "lineEndings", "style")
	case "convert_encoding":
		renameArg(m, "fromEncoding", "from")
		renameArg(m, "from_encoding", "from")
		renameArg(m, "toEncoding", "to")
		renameArg(m, "to_encoding", "to")
		if on, ok := boolArg(m["addBom"]); ok {
			delete(m, "addBom")
			bom := `"never"`
			if on {
				bom = `"always"`
			}
			setIfAbsent(m, "bom", json.RawMessage(bom))
		}
	case "tree":
		renameArg(m, "depth", "maxDepth")
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

// boolArg reads a boolean sent either as JSON or as the string "true"/"false".
func boolArg(v json.RawMessage) (bool, bool) {
	if v == nil {
		return false, false
	}
	var b bool
	if json.Unmarshal(v, &b) == nil {
		return b, true
	}
	var s string
	if json.Unmarshal(v, &s) == nil {
		if s = strings.TrimSpace(s); s == "true" || s == "false" {
			return s == "true", true
		}
	}
	return false, false
}

// intArg reads an integer sent either as JSON or as a numeric string.
func intArg(v json.RawMessage) (int, bool) {
	if v == nil {
		return 0, false
	}
	var n int
	if json.Unmarshal(v, &n) == nil {
		return n, true
	}
	var s string
	if json.Unmarshal(v, &s) == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
			return n, true
		}
	}
	return 0, false
}

// aliasLineRange turns an inclusive startLine/endLine into offset/limit.
func aliasLineRange(m map[string]json.RawMessage) {
	start, hasStart := intArg(m["startLine"])
	if hasStart {
		if _, dup := m["offset"]; dup {
			return
		}
		delete(m, "startLine")
		m["offset"], _ = json.Marshal(start)
	}
	end, hasEnd := intArg(m["endLine"])
	if !hasEnd {
		return
	}
	if _, dup := m["limit"]; dup {
		return
	}
	// The read treats an offset below 1 as line 1.
	from := 1
	if n, ok := intArg(m["offset"]); ok {
		from = max(n, 1)
	}
	if end < from {
		return
	}
	delete(m, "endLine")
	m["limit"], _ = json.Marshal(end - from + 1)
}

// aliasFlatEdit makes a flat old/new/replaceAll triple one edits entry; beside edits or patch it stays and fails validation.
func aliasFlatEdit(m map[string]json.RawMessage, oldKey, newKey, allKey string) {
	_, hasEdits := m["edits"]
	_, hasPatch := m["patch"]
	oldS, okOld := m[oldKey]
	newS, okNew := m[newKey]
	if hasEdits || hasPatch || !okOld || !okNew {
		return
	}
	edit := map[string]json.RawMessage{"oldText": oldS, "newText": newS}
	if ra, ok := m[allKey]; ok {
		// Nested, so the top-level string repair never reaches it.
		if on, isBool := boolArg(ra); isBool {
			ra, _ = json.Marshal(on)
		}
		edit["replaceAll"] = ra
		delete(m, allKey)
	}
	if arr, err := json.Marshal([]map[string]json.RawMessage{edit}); err == nil {
		delete(m, oldKey)
		delete(m, newKey)
		m["edits"] = arr
	}
}

// aliasGrepArgs maps built-in Grep and commonly guessed names onto ours.
func aliasGrepArgs(m map[string]json.RawMessage) {
	renameArg(m, "-B", "contextBefore")
	renameArg(m, "-A", "contextAfter")
	renameArg(m, "-o", "matchesOnly")
	renameArg(m, "head_limit", "maxMatches")
	renameArg(m, "maxResults", "maxMatches")
	renameArg(m, "output_mode", "outputMode")
	renameArg(m, "filePattern", "include")
	// Matches always carry line numbers.
	delete(m, "-n")
	delete(m, "lineNumbers")
	// pattern is always a regex, so only a true flag is a no-op; false stays and is rejected.
	for _, k := range []string{"isRegex", "useRegex", "regex"} {
		if on, ok := boolArg(m[k]); ok && on {
			delete(m, k)
		}
	}
	// -i and ignoreCase invert to caseSensitive.
	for _, k := range []string{"-i", "ignoreCase"} {
		if insensitive, ok := boolArg(m[k]); ok {
			delete(m, k)
			if _, dup := m["caseSensitive"]; !dup {
				m["caseSensitive"], _ = json.Marshal(!insensitive)
			}
		}
	}
	// -C, context and contextLines fill both sides.
	for _, k := range []string{"-C", "context", "contextLines", "context_lines"} {
		if v, ok := m[k]; ok {
			delete(m, k)
			setIfAbsent(m, "contextBefore", v)
			setIfAbsent(m, "contextAfter", v)
		}
	}
	// Built-in Grep takes one path string; ours takes paths. An array under path is paths already.
	if v, ok := m["path"]; ok {
		if _, dup := m["paths"]; !dup {
			var s string
			var arr []string
			if json.Unmarshal(v, &s) == nil {
				m["paths"], _ = json.Marshal([]string{s})
				delete(m, "path")
			} else if json.Unmarshal(v, &arr) == nil {
				m["paths"] = v
				delete(m, "path")
			}
		}
	}
}
