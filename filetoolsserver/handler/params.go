// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ParamIndex holds each tool's parameters, read from the struct the SDK derives the schema from.
type ParamIndex map[string]*toolParams

type toolParams struct {
	names  []string // declaration order, for the error message
	byName map[string]paramSpec
}

type paramSpec struct {
	kind     reflect.Kind // pointers dereferenced
	required bool
}

// IndexParams records In's JSON fields under tool.
func IndexParams[In any](idx ParamIndex, tool string) {
	tp := &toolParams{byName: map[string]paramSpec{}}
	for f := range reflect.TypeFor[In]().Fields() {
		name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
		if !f.IsExported() || name == "-" {
			continue
		}
		if name == "" {
			name = f.Name
		}
		ft := f.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		optional := strings.Contains(opts, "omitempty") || strings.Contains(opts, "omitzero")
		tp.names = append(tp.names, name)
		tp.byName[name] = paramSpec{kind: ft.Kind(), required: !optional}
	}
	// An alias that rewrote a real parameter would silently change correct calls; some fire only on one type.
	for _, name := range tp.names {
		for _, v := range []any{1, true, "x"} {
			probe, _ := json.Marshal(map[string]any{name: v})
			if out := aliasBuiltinArgs(tool, probe); string(out) != string(probe) {
				panic(fmt.Sprintf("%s: an alias rewrites the real parameter %q", tool, name))
			}
		}
	}
	idx[tool] = tp
}

// RepairGuessedParams fixes exact synonyms and stringified values; other unknown names fail with the parameter list.
func RepairGuessedParams(idx ParamIndex) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if r, ok := req.(*mcp.CallToolRequest); ok && r.Params != nil {
				args := aliasBuiltinArgs(r.Params.Name, r.Params.Arguments)
				if tp, known := idx[r.Params.Name]; known {
					var msg string
					if args, msg = tp.check(r.Params.Name, args); msg != "" {
						return errorResult(msg), nil
					}
				}
				r.Params.Arguments = args
			}
			return next(ctx, method, req)
		}
	}
}

// check decodes stringified values and reports names the tool does not take.
func (tp *toolParams) check(tool string, raw json.RawMessage) (json.RawMessage, string) {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil || m == nil {
		return raw, ""
	}
	var unknown []string
	changed := false
	for k, v := range m {
		spec, ok := tp.byName[k]
		if !ok {
			unknown = append(unknown, k)
			continue
		}
		if fixed, ok := unstringValue(spec.kind, v); ok {
			m[k] = fixed
			changed = true
		}
	}
	if len(unknown) > 0 {
		return raw, tp.unknownMessage(tool, unknown, m)
	}
	if !changed {
		return raw, ""
	}
	out, err := json.Marshal(m)
	if err != nil {
		return raw, ""
	}
	return out, ""
}

// unstringValue decodes "12", "true" or "[...]" only for a parameter of that type; a string parameter keeps its text.
func unstringValue(kind reflect.Kind, v json.RawMessage) (json.RawMessage, bool) {
	var s string
	if json.Unmarshal(v, &s) != nil {
		return nil, false
	}
	s = strings.TrimSpace(s)
	switch kind {
	case reflect.Int, reflect.Int64, reflect.Int32:
		if n, err := strconv.Atoi(s); err == nil {
			return json.RawMessage(strconv.Itoa(n)), true
		}
	case reflect.Float64:
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return json.RawMessage(strconv.FormatFloat(f, 'g', -1, 64)), true
		}
	case reflect.Bool:
		if s == "true" || s == "false" {
			return json.RawMessage(s), true
		}
	case reflect.Slice:
		if strings.HasPrefix(s, "[") && json.Valid([]byte(s)) {
			return json.RawMessage(s), true
		}
	}
	return nil, false
}

// paramHints answer guesses that have no exact translation, keyed tool.name.
var paramHints = map[string]string{
	"read_text_file.tail":           "every read returns totalLines; read the end with offset=totalLines-N+1",
	"read_text_file.length":         "use limit (lines) or maxCharacters",
	"manage_line_endings.convert":   `use action="convert" with style "lf" or "crlf"`,
	"manage_line_endings.operation": "use action",
	"manage_line_endings.paths":     "it takes one path per call",
	"manage_line_endings.backup":    "it has no backup; copy_file the file first",
	"grep_text_files.isRegex":       "pattern is always a regex; escape metacharacters for a literal match",
	"grep_text_files.regex":         "put the regex in pattern, which is always a regex; escape metacharacters for a literal match",
	"grep_text_files.useRegex":      "pattern is always a regex; escape metacharacters for a literal match",
	"search_files.query":            "pattern is a glob on file names; search file contents with grep_text_files",
}

func (tp *toolParams) unknownMessage(tool string, unknown []string, m map[string]json.RawMessage) string {
	slices.Sort(unknown)
	var b strings.Builder
	for _, k := range unknown {
		fmt.Fprintf(&b, "%s does not take %q", tool, k)
		if h, ok := paramHints[tool+"."+k]; ok {
			b.WriteString(": ")
			b.WriteString(h)
		}
		b.WriteString(". ")
	}
	var list, missing []string
	for _, n := range tp.names {
		if tp.byName[n].required {
			list = append(list, n+" (required)")
			if _, ok := m[n]; !ok {
				missing = append(missing, n)
			}
		} else {
			list = append(list, n)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(&b, "missing required: %s. ", strings.Join(missing, ", "))
	}
	fmt.Fprintf(&b, "Parameters: %s.", strings.Join(list, ", "))
	return b.String()
}
