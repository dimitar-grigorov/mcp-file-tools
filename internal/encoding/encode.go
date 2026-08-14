// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
)

// maxReportedRunes caps the offending characters an UnsupportedError names.
const maxReportedRunes = 10

// UnsupportedRune locates one character a target encoding cannot represent.
type UnsupportedRune struct {
	Char   string `json:"char"`
	Code   string `json:"code"` // U+XXXX
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// UnsupportedError says which characters charset cannot represent, and where.
type UnsupportedError struct {
	Charset string
	Runes   []UnsupportedRune // capped at maxReportedRunes
	Total   int               // may exceed len(Runes)
}

func (e *UnsupportedError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s cannot represent ", e.Charset)
	if e.Total == 1 {
		b.WriteString("1 character: ")
	} else {
		fmt.Fprintf(&b, "%d characters", e.Total)
		if e.Total > len(e.Runes) {
			fmt.Fprintf(&b, ", first %d", len(e.Runes))
		}
		b.WriteString(": ")
	}
	for i, r := range e.Runes {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%q (%s) at line %d, column %d", r.Char, r.Code, r.Line, r.Column)
	}
	b.WriteString(". Convert to utf-8 instead, or remove these characters")
	if e.Total > len(e.Runes) {
		b.WriteString(" (list_encodings shows what each encoding covers)")
	}
	b.WriteString(". To see the lines, read_text_file with lineNumbers=true.")
	return b.String()
}

// Decode converts data in charset to UTF-8. A BOM decodes as content; strip it first.
func Decode(data []byte, charset string) (string, error) {
	if IsUTF8(charset) {
		return string(data), nil
	}
	enc, ok := Get(charset)
	if !ok {
		return "", fmt.Errorf("unsupported encoding: %s", charset)
	}
	decoded, err := enc.NewDecoder().Bytes(data)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

// Encode converts UTF-8 content to charset; the error is an *UnsupportedError.
func Encode(content string, enc encoding.Encoding, charset string) ([]byte, error) {
	out, err := enc.NewEncoder().Bytes([]byte(content))
	if err == nil {
		return out, nil
	}
	if bad := FindUnsupported(content, enc, charset); bad != nil {
		return nil, bad
	}
	return nil, err
}

// FindUnsupported reports what charset cannot represent, nil if it covers content.
// Tests one rune at a time, so call it only after an encode has failed.
func FindUnsupported(content string, enc encoding.Encoding, charset string) *UnsupportedError {
	encoder := enc.NewEncoder()
	res := &UnsupportedError{Charset: charset}

	line, col := 1, 1
	for _, r := range content {
		if r == '\n' {
			line++
			col = 1
			continue
		}
		// Every encoding in the registry covers ASCII, so only test above it.
		if r >= utf8.RuneSelf {
			if _, err := encoder.Bytes([]byte(string(r))); err != nil {
				res.Total++
				if len(res.Runes) < maxReportedRunes {
					res.Runes = append(res.Runes, UnsupportedRune{
						Char:   string(r),
						Code:   fmt.Sprintf("U+%04X", r),
						Line:   line,
						Column: col,
					})
				}
			}
		}
		col++
	}

	if res.Total == 0 {
		return nil
	}
	return res
}
