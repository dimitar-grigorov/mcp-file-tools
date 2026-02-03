package handler

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// LineEndingStyle constants for line ending types.
const (
	LineEndingCRLF  = "crlf"
	LineEndingLF    = "lf"
	LineEndingMixed = "mixed"
	LineEndingNone  = "none"
)

// LineEndingInfo holds detected line ending information.
type LineEndingInfo struct {
	Style     string // "crlf", "lf", "mixed", or "none"
	CRLFCount int
	LFCount   int // LF not preceded by CR
}

// DetectLineEndings analyzes data and returns line ending information.
// Works on byte slice for in-memory data.
func DetectLineEndings(data []byte) LineEndingInfo {
	info := LineEndingInfo{}

	for i := 0; i < len(data); i++ {
		if data[i] == '\r' && i+1 < len(data) && data[i+1] == '\n' {
			info.CRLFCount++
			i++ // skip the \n
		} else if data[i] == '\n' {
			info.LFCount++
		}
	}

	info.Style = determineStyle(info.CRLFCount, info.LFCount)
	return info
}

// DetectLineEndingsFromReader detects line endings by streaming from a reader.
// Uses buffered reading to avoid loading entire file into memory.
func DetectLineEndingsFromReader(r io.Reader) (LineEndingInfo, error) {
	info := LineEndingInfo{}
	br := bufio.NewReader(r)
	prevWasCR := false

	for {
		b, err := br.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return info, err
		}

		if b == '\n' {
			if prevWasCR {
				info.CRLFCount++
			} else {
				info.LFCount++
			}
		}
		prevWasCR = (b == '\r')
	}

	info.Style = determineStyle(info.CRLFCount, info.LFCount)
	return info, nil
}

// DetectLineEndingsFromFile detects line endings by streaming from a file.
func DetectLineEndingsFromFile(path string) (LineEndingInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return LineEndingInfo{}, err
	}
	defer f.Close()

	return DetectLineEndingsFromReader(f)
}

// determineStyle returns the line ending style based on counts.
func determineStyle(crlfCount, lfCount int) string {
	switch {
	case crlfCount == 0 && lfCount == 0:
		return LineEndingNone
	case crlfCount > 0 && lfCount == 0:
		return LineEndingCRLF
	case crlfCount == 0 && lfCount > 0:
		return LineEndingLF
	default:
		return LineEndingMixed
	}
}

// ConvertLineEndings converts text to the specified line ending style.
func ConvertLineEndings(text string, targetStyle string) string {
	hasCRLF := strings.Contains(text, "\r\n")

	if targetStyle == LineEndingCRLF {
		if !hasCRLF {
			// Only LF present, single pass: LF -> CRLF
			return strings.ReplaceAll(text, "\n", "\r\n")
		}
		// Has CRLF (might be mixed), normalize then convert
		normalized := strings.ReplaceAll(text, "\r\n", "\n")
		return strings.ReplaceAll(normalized, "\n", "\r\n")
	}

	// Target is LF (or other non-CRLF style)
	if !hasCRLF {
		return text // Already no CRLF
	}
	return strings.ReplaceAll(text, "\r\n", "\n")
}
