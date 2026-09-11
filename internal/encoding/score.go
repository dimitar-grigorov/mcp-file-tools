// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"
)

// One measure for every place that second-guesses the detector: read the bytes as a charset
// and ask how much of what comes out reads like running text. A wrong table keeps the ASCII
// intact and turns the rest into a rash of accents and capitals between the letters, so the
// question that separates tables is where the non-ASCII lands, not whether it decodes.

const (
	minWordRun   = 3   // three letters in a row is a word, not an accent
	minWordBytes = 12  // below this there is no text to judge
	minWordShare = 0.8 // nearly every non-ASCII rune belongs to one of those words
)

// scripts are the alphabets this server tells apart; a word mixing two of them is mojibake, not a word.
var scripts = []*unicode.RangeTable{
	unicode.Latin,
	unicode.Cyrillic,
	unicode.Greek,
	unicode.Han,
	unicode.Arabic,
	unicode.Hebrew,
	unicode.Thai,
	unicode.Hiragana,
	unicode.Katakana,
	unicode.Hangul,
	unicode.Armenian,
	unicode.Georgian,
}

const (
	latinScript = 0  // index into scripts, the answer for every ASCII letter
	otherScript = -1 // a letter in an alphabet this server does not track
)

// plausibility is what one charset makes of the bytes: whether they can be it at all, and how much of the non-ASCII it turns into words.
type plausibility struct {
	readable bool // the bytes decode with no replacement character
	high     int  // non-ASCII runes in the reading
	inWords  int  // of those, the ones sitting inside runs that read like words
}

// share is the fraction of the non-ASCII that reads as words; pure ASCII is 1, since every charset reads it alike.
func (p plausibility) share() float64 {
	if p.high == 0 {
		return 1
	}
	return float64(p.inWords) / float64(p.high)
}

// plausible reports whether the reading carries enough text, well enough formed, to overrule the detector.
func (p plausibility) plausible() bool {
	return p.readable && p.inWords >= minWordBytes && p.share() >= minWordShare
}

// scoreCharset reads data as charset and reports how much like running text it comes out.
func scoreCharset(charset string, data []byte) plausibility {
	var s scorer

	switch {
	case IsUTF8(charset):
		if !utf8.Valid(data) {
			return plausibility{}
		}
		for _, r := range string(data) {
			s.add(r)
		}
		return s.finish()
	case strings.HasPrefix(charset, "utf-16"), strings.HasPrefix(charset, "utf-32"):
		// These swallow almost any bytes, so only a BOM or the structural classifier may name them.
		return plausibility{}
	}

	// A single-byte table needs no decoder: an undefined byte maps to U+FFFD.
	if cm := charmapFor(charset); cm != nil {
		for _, b := range data {
			r := cm.DecodeByte(b)
			if r == utf8.RuneError {
				return plausibility{}
			}
			s.add(r)
		}
		return s.finish()
	}

	enc, ok := Get(charset)
	if !ok {
		return plausibility{}
	}
	decoded, err := enc.NewDecoder().Bytes(data)
	if err != nil || bytes.ContainsRune(decoded, utf8.RuneError) {
		return plausibility{}
	}
	for _, r := range string(decoded) {
		s.add(r)
	}
	return s.finish()
}

// scorer walks a decoded rune stream and tallies how much of the non-ASCII sits inside word-like runs.
type scorer struct {
	total    plausibility
	word     []rune
	wordHigh int
}

// add feeds one decoded rune.
func (s *scorer) add(r rune) {
	if r >= utf8.RuneSelf {
		s.total.high++
	}
	if unicode.IsLetter(r) {
		s.word = append(s.word, r)
		if r >= utf8.RuneSelf {
			s.wordHigh++
		}
		return
	}
	s.endWord()
}

// endWord banks the run just closed; only a run carrying non-ASCII can move the score, so an all-ASCII one is dropped unexamined.
func (s *scorer) endWord() {
	if s.wordHigh > 0 && wordLike(s.word) {
		s.total.inWords += s.wordHigh
	}
	s.word = s.word[:0]
	s.wordHigh = 0
}

// finish closes the last run and marks the reading readable, which it is by the time anything is scored.
func (s *scorer) finish() plausibility {
	s.endWord()
	s.total.readable = true
	return s.total
}

// wordLike reports whether the letters read as one word: a single alphabet, cased the way a word is rather than shouted back by the wrong table, and for Latin not spelled out of accents alone.
func wordLike(word []rune) bool {
	if len(word) < minWordRun {
		return false
	}
	script := scriptOf(word[0])
	if script == otherScript {
		return false
	}
	ascii := 0
	if word[0] < utf8.RuneSelf {
		ascii++
	}
	for _, r := range word[1:] {
		if unicode.IsUpper(r) || scriptOf(r) != script {
			return false
		}
		if r < utf8.RuneSelf {
			ascii++
		}
	}
	// Cyrillic under a Latin table comes out as a run of lowercase accents, and no European language writes a word without an ASCII letter in it.
	return script != latinScript || ascii > 0
}

// scriptOf identifies the alphabet a letter belongs to, otherScript for one this server does not track.
func scriptOf(r rune) int {
	if r < utf8.RuneSelf {
		return latinScript
	}
	for i, table := range scripts {
		if unicode.Is(table, r) {
			return i
		}
	}
	return otherScript
}
