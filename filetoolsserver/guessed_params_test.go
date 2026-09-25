// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package filetoolsserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Guesses taken from real Claude Code sessions that called a tool without loading its schema.

func sessionIn(t *testing.T, dir string) *mcp.ClientSession {
	t.Helper()
	server := NewServer([]string{dir}, nil, nil)
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	t1, t2 := mcp.NewInMemoryTransports()
	if _, err := server.Connect(context.Background(), t1, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := client.Connect(context.Background(), t2, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

// call sends args verbatim, so a test can pass names and types the schema does not declare.
func call(t *testing.T, cs *mcp.ClientSession, tool, args string) (text string, isError bool) {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: tool, Arguments: json.RawMessage(args)})
	if err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			text += tc.Text
		}
	}
	return text, res.IsError
}

func jsonPath(p string) string {
	b, _ := json.Marshal(p)
	return string(b)
}

func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGuessedNamesReachTheHandler(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	writeFixture(t, f, "one\ntwo\nthree\nfour\n")
	p, d := jsonPath(f), jsonPath(dir)
	cs := sessionIn(t, dir)

	cases := []struct {
		name, tool, args string
		want             []string // substrings of the result text
		reject           []string
	}{
		{"head as a string", "read_text_file", `{"path":` + p + `,"head":"2"}`, []string{`"one\ntwo\n"`}, []string{"three"}},
		{"maxLines", "read_text_file", `{"path":` + p + `,"maxLines":1}`, []string{"one"}, []string{"two"}},
		{"startLine and endLine", "read_text_file", `{"path":` + p + `,"startLine":"2","endLine":"3"}`, []string{`"two\nthree\n"`}, []string{"one", "four"}},
		{"grep guesses", "grep_text_files", `{"path":` + d + `,"pattern":"TWO","filePattern":"*.txt","ignoreCase":"true","contextLines":"1","maxResults":"5","lineNumbers":"true","isRegex":"true"}`, []string{`"text":"two"`, `"one"`, `"three"`}, nil},
		{"-i as a string", "grep_text_files", `{"path":` + d + `,"pattern":"TWO","-i":"true"}`, []string{`"text":"two"`}, nil},
		{"startLine 0 reads from line 1", "read_text_file", `{"path":` + p + `,"startLine":0,"endLine":2}`, []string{`"one\ntwo\n"`}, []string{"three"}},
		{"path as a stringified array", "grep_text_files", `{"path":` + jsonPath("["+d+"]") + `,"pattern":"two"}`, []string{`"text":"two"`}, nil},
		{"patterns alone", "grep_text_files", `{"patterns":["one","four"],"paths":[` + d + `]}`, []string{`"totalMatches":2`}, nil},
		{"tree depth", "tree", `{"path":` + d + `,"depth":"1"}`, []string{"a.txt"}, nil},
		{"string boolean", "list_directory", `{"path":` + d + `,"reverse":"true"}`, []string{"a.txt"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text, isErr := call(t, cs, c.tool, c.args)
			if isErr {
				t.Fatalf("error: %s", text)
			}
			for _, w := range c.want {
				if !strings.Contains(text, w) {
					t.Errorf("missing %q in %s", w, text)
				}
			}
			for _, r := range c.reject {
				if strings.Contains(text, r) {
					t.Errorf("unexpected %q in %s", r, text)
				}
			}
		})
	}
}

func TestGuessedNamesChangeTheFile(t *testing.T) {
	dir := t.TempDir()
	cs := sessionIn(t, dir)

	eol := filepath.Join(dir, "eol.txt")
	writeFixture(t, eol, "a\nb\n")
	if text, isErr := call(t, cs, "manage_line_endings", `{"path":`+jsonPath(eol)+`,"action":"convert","lineEnding":"crlf"}`); isErr {
		t.Fatalf("manage_line_endings: %s", text)
	}
	if b, _ := os.ReadFile(eol); string(b) != "a\r\nb\r\n" {
		t.Errorf("manage_line_endings lineEnding: got %q", b)
	}

	w := filepath.Join(dir, "w.txt")
	if text, isErr := call(t, cs, "write_file", `{"path":`+jsonPath(w)+`,"content":"a\nb\n","lineEnding":"crlf"}`); isErr {
		t.Fatalf("write_file: %s", text)
	}
	if b, _ := os.ReadFile(w); string(b) != "a\r\nb\r\n" {
		t.Errorf("write_file lineEnding: got %q", b)
	}

	conv := filepath.Join(dir, "conv.txt")
	writeFixture(t, conv, "Здравей\n")
	if text, isErr := call(t, cs, "convert_encoding", `{"path":`+jsonPath(conv)+`,"fromEncoding":"utf-8","toEncoding":"windows-1251","addBom":"false"}`); isErr {
		t.Fatalf("convert_encoding: %s", text)
	}
	if b, _ := os.ReadFile(conv); string(b) != "\xc7\xe4\xf0\xe0\xe2\xe5\xe9\n" {
		t.Errorf("convert_encoding: got % x", b)
	}

	ed := filepath.Join(dir, "ed.txt")
	writeFixture(t, ed, "old\n")
	if text, isErr := call(t, cs, "edit_file", `{"path":`+jsonPath(ed)+`,"oldText":"old","newText":"new"}`); isErr {
		t.Fatalf("edit_file: %s", text)
	}
	if b, _ := os.ReadFile(ed); string(b) != "new\n" {
		t.Errorf("edit_file flat oldText: got %q", b)
	}

	all := filepath.Join(dir, "all.txt")
	writeFixture(t, all, "x x\n")
	if text, isErr := call(t, cs, "edit_file", `{"file_path":`+jsonPath(all)+`,"old_string":"x","new_string":"y","replace_all":"true"}`); isErr {
		t.Fatalf("edit_file replace_all as a string: %s", text)
	}
	if b, _ := os.ReadFile(all); string(b) != "y y\n" {
		t.Errorf("edit_file replace_all as a string: got %q", b)
	}
}

// A name with no exact meaning fails in one round trip that says what to send instead.
func TestUnknownNameErrorNamesTheFix(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	writeFixture(t, f, "x\n")
	p := jsonPath(f)
	cs := sessionIn(t, dir)

	cases := []struct {
		tool, args string
		want       []string
	}{
		{"read_text_file", `{"path":` + p + `,"tail":"5"}`, []string{"tail", "totalLines", "offset", "path (required)", "limit"}},
		{"manage_line_endings", `{"path":` + p + `,"action":"convert","convert":"to_crlf"}`, []string{"convert", `style`, `"crlf"`}},
		{"grep_text_files", `{"path":` + p + `,"pattern":"a.b","isRegex":"false"}`, []string{"isRegex", "always a regex"}},
		{"search_files", `{"path":` + jsonPath(dir) + `,"query":"2427"}`, []string{"query", "grep_text_files", "missing required: pattern"}},
		{"tree", `{"path":` + jsonPath(dir) + `,"levels":2}`, []string{"levels", "maxDepth"}},
	}
	for _, c := range cases {
		t.Run(c.tool, func(t *testing.T) {
			text, isErr := call(t, cs, c.tool, c.args)
			if !isErr {
				t.Fatalf("want an error, got %s", text)
			}
			for _, w := range c.want {
				if !strings.Contains(text, w) {
					t.Errorf("missing %q in %q", w, text)
				}
			}
		})
	}
}

// The repair middleware must know every published property, or a correct call is refused.
func TestEveryPublishedParameterIsKnown(t *testing.T) {
	cs := sessionIn(t, t.TempDir())
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range res.Tools {
		schema, _ := json.Marshal(tool.InputSchema)
		var s struct {
			Properties map[string]json.RawMessage `json:"properties"`
		}
		if err := json.Unmarshal(schema, &s); err != nil {
			t.Fatalf("%s: %v", tool.Name, err)
		}
		if len(s.Properties) == 0 {
			continue
		}
		// Nulls fail type validation, so no handler runs.
		args := map[string]any{}
		for p := range s.Properties {
			args[p] = nil
		}
		raw, _ := json.Marshal(args)
		if text, _ := call(t, cs, tool.Name, string(raw)); strings.Contains(text, "does not take") {
			t.Errorf("%s: %s", tool.Name, text)
		}
	}
}
