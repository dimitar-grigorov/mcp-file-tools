// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

//go:build corpus

// Manual only: without -tags corpus this file is never built.
package encoding

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// Point MCP_FILE_TOOLS_CORPUS at a real tree; the path stays in the environment.
const (
	envCorpus      = "MCP_FILE_TOOLS_CORPUS"
	envCorpusExts  = "MCP_FILE_TOOLS_CORPUS_EXTS" // default ".pas,.dfm,.inc,.dpr"
	corpusMaxFiles = 4000
)

func corpusFiles(t testing.TB) []string {
	t.Helper()
	root := os.Getenv(envCorpus)
	if root == "" {
		t.Skipf("set %s to a real source tree to run this", envCorpus)
	}

	exts := strings.Split(".pas,.dfm,.inc,.dpr", ",")
	if custom := os.Getenv(envCorpusExts); custom != "" {
		exts = strings.Split(strings.ToLower(custom), ",")
	}
	wanted := func(name string) bool {
		ext := strings.ToLower(filepath.Ext(name))
		for _, e := range exts {
			if ext == strings.TrimSpace(e) {
				return true
			}
		}
		return false
	}

	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || len(files) >= corpusMaxFiles {
			return nil //nolint:nilerr // an unreadable subtree is not this test's problem
		}
		if d.IsDir() {
			if name := d.Name(); name == ".git" || name == "__history" || name == "__recovery" {
				return filepath.SkipDir
			}
			return nil
		}
		if wanted(d.Name()) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	if len(files) == 0 {
		t.Skipf("no matching files under %s", root)
	}
	return files
}

// cp1251 maps almost every high byte to Cyrillic; only word runs discriminate.
const minCyrillicWords = 5

// One repeated letter is 0xFF blob padding, not a word: require two distinct.
func cyrillicWords(text string) int {
	words := 0
	run, distinct := 0, map[rune]struct{}{}
	flush := func() {
		if run >= 3 && len(distinct) >= 2 {
			words++
		}
		run, distinct = 0, map[rune]struct{}{}
	}
	for _, r := range text {
		if (r >= 'А' && r <= 'я') || r == 'Ё' || r == 'ё' {
			run++
			distinct[r] = struct{}{}
			continue
		}
		flush()
	}
	flush()
	return words
}

func TestCorpus_DetectionIsUsable(t *testing.T) {
	files := corpusFiles(t)
	t.Logf("corpus: %d files", len(files))

	counts := map[string]int{}
	lowConfidence := map[string]int{}
	unsupportedBy := map[string]int{}
	var unsupported, nondeterministic, replacementChars, lossyRoundTrip int
	var westernOnCyrillic []string

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil || len(data) == 0 {
			continue
		}

		got := Detect(data)
		counts[got.Charset]++
		if got.Confidence < MinConfidenceThreshold {
			lowConfidence[got.Charset]++
		}

		if again := Detect(data); again.Charset != got.Charset {
			nondeterministic++
			t.Errorf("%s: detection is not deterministic (%s then %s)", path, got.Charset, again.Charset)
		}

		if got.Charset == "" {
			continue
		}
		enc, ok := Get(got.Charset)
		if !ok {
			// Not a defect: the read path falls back to the configured default.
			unsupported++
			unsupportedBy[got.Charset]++
			continue
		}

		text := string(data)
		if enc != nil {
			decoded, err := enc.NewDecoder().Bytes(data)
			if err != nil {
				t.Errorf("%s: detected %s but decoding failed: %v", path, got.Charset, err)
				continue
			}
			text = string(decoded)

			// A single-byte charset round-trips unless the guess was wrong.
			if _, isCharmap := enc.(*charmap.Charmap); isCharmap {
				if back, err := Encode(text, enc, got.Charset); err != nil || string(back) != string(data) {
					lossyRoundTrip++
				}
			}
		}
		if strings.ContainsRune(text, utf8.RuneError) {
			replacementChars++
		}

		// A Cyrillic source read as Western European is the headline failure.
		if isWestern(got.Charset) && cyrillicWords(text) == 0 {
			if alt, ok := Get("windows-1251"); ok {
				if cp1251, err := alt.NewDecoder().Bytes(data); err == nil && cyrillicWords(string(cp1251)) >= minCyrillicWords {
					westernOnCyrillic = append(westernOnCyrillic, path)
				}
			}
		}
	}

	report(t, "detected", counts)
	report(t, "below the confidence threshold", lowConfidence)
	report(t, "detected but absent from the registry", unsupportedBy)
	t.Logf("unsupported=%d nondeterministic=%d U+FFFD=%d lossy-roundtrip=%d",
		unsupported, nondeterministic, replacementChars, lossyRoundTrip)

	// Reported, not failed: oddities in someone's tree are not a regression here.
	if n := len(westernOnCyrillic); n > 0 {
		t.Logf("%d files hold Cyrillic words under cp1251 but were detected as Western:", n)
		for _, p := range westernOnCyrillic[:min(10, n)] {
			t.Logf("  %s", p)
		}
	}
}

func isWestern(charset string) bool {
	switch charset {
	case "windows-1252", "iso-8859-1", "iso-8859-15", "windows-1250", "iso-8859-2":
		return true
	}
	return false
}

func report(t *testing.T, label string, counts map[string]int) {
	t.Helper()
	if len(counts) == 0 {
		t.Logf("%s: none", label)
		return
	}
	type row struct {
		name string
		n    int
	}
	rows := make([]row, 0, len(counts))
	total := 0
	for name, n := range counts {
		if name == "" {
			name = "(undetected)"
		}
		rows = append(rows, row{name, n})
		total += n
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	t.Logf("%s (%d files):", label, total)
	for _, r := range rows {
		t.Logf("  %-16s %5d  %5.1f%%", r.name, r.n, 100*float64(r.n)/float64(total))
	}
}

func BenchmarkCorpusDetect(b *testing.B) {
	files := corpusFiles(b)
	contents := make([][]byte, 0, len(files))
	var bytes int64
	for _, path := range files {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			contents = append(contents, data)
			bytes += int64(len(data))
		}
	}
	b.SetBytes(bytes)
	b.ResetTimer()
	for b.Loop() {
		for _, data := range contents {
			Detect(data)
		}
	}
}
