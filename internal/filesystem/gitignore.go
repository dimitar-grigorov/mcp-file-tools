// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package filesystem

import (
	"path"
	"strings"
)

// ignorePattern is one parsed .gitignore line.
type ignorePattern struct {
	segs     []string // slash-split pattern, "**" segments cross directories
	negate   bool     // "!" prefix: re-include
	dirOnly  bool     // trailing "/": matches directories only
	anchored bool     // contains an interior "/": relative to the .gitignore's dir
}

// ignoreScope is the parsed .gitignore of one directory on the walk path.
type ignoreScope struct {
	relDir   string // slash-relative to the walk root, "" for the root itself
	patterns []ignorePattern
}

type ignoreStack []ignoreScope

// parseGitignore parses .gitignore content. Unsupported or empty lines are dropped.
func parseGitignore(data []byte) []ignorePattern {
	var out []ignorePattern
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSuffix(line, "\r")
		// Trailing spaces are ignored unless backslash-escaped
		for strings.HasSuffix(line, " ") && !strings.HasSuffix(line, "\\ ") {
			line = line[:len(line)-1]
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		var p ignorePattern
		if strings.HasPrefix(line, "!") {
			p.negate = true
			line = line[1:]
		}
		line = strings.TrimPrefix(line, "\\") // "\#" and "\!" literals
		if strings.HasSuffix(line, "/") {
			p.dirOnly = true
			line = strings.TrimSuffix(line, "/")
		}
		// A leading or interior slash anchors the pattern to the .gitignore's dir
		if strings.Contains(line, "/") {
			p.anchored = true
			line = strings.TrimPrefix(line, "/")
		}
		if line == "" {
			continue
		}
		p.segs = strings.Split(line, "/")
		out = append(out, p)
	}
	return out
}

// Ignored reports whether relPath (slash-relative to the walk root) is ignored.
// Deeper .gitignore files take precedence; within a file the last match wins.
func (s ignoreStack) Ignored(relPath string, isDir bool) bool {
	ignored := false
	for _, scope := range s {
		rel := relPath
		if scope.relDir != "" {
			rel = strings.TrimPrefix(relPath, scope.relDir+"/")
		}
		segs := strings.Split(rel, "/")
		for _, p := range scope.patterns {
			if p.dirOnly && !isDir {
				continue
			}
			if p.matches(segs) {
				ignored = !p.negate
			}
		}
	}
	return ignored
}

func (p ignorePattern) matches(pathSegs []string) bool {
	if p.anchored {
		return matchSegs(p.segs, pathSegs)
	}
	// Unanchored: match against the basename at any depth
	return matchSegs(p.segs, pathSegs[len(pathSegs)-1:])
}

// matchSegs matches pattern segments against path segments; "**" crosses directories.
func matchSegs(pat, segs []string) bool {
	if len(pat) == 0 {
		return len(segs) == 0
	}
	if pat[0] == "**" {
		if len(pat) == 1 { // trailing /**: everything inside, not the dir itself
			return len(segs) > 0
		}
		for i := 0; i <= len(segs); i++ {
			if matchSegs(pat[1:], segs[i:]) {
				return true
			}
		}
		return false
	}
	if len(segs) == 0 {
		return false
	}
	if ok, err := path.Match(pat[0], segs[0]); err != nil || !ok {
		return false
	}
	return matchSegs(pat[1:], segs[1:])
}
