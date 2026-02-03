package handler

import (
	"bufio"
	"context"
	"io"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

// HandleDetectLineEndings detects line ending style and returns inconsistent line numbers.
func (h *Handler) HandleDetectLineEndings(ctx context.Context, req *mcp.CallToolRequest, input DetectLineEndingsInput) (*mcp.CallToolResult, DetectLineEndingsOutput, error) {
	v := h.ValidatePath(input.Path)
	if !v.Ok() {
		return v.Result, DetectLineEndingsOutput{}, nil
	}

	f, err := os.Open(v.Path)
	if err != nil {
		return errorResult("failed to open file: " + err.Error()), DetectLineEndingsOutput{}, nil
	}
	defer f.Close()

	// Track each line's ending type
	type lineEnding struct {
		lineNum int
		isCRLF  bool
	}
	var lineEndings []lineEnding

	br := bufio.NewReader(f)
	lineNum := 1
	prevWasCR := false

	for {
		b, err := br.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errorResult("failed to read file: " + err.Error()), DetectLineEndingsOutput{}, nil
		}

		if b == '\n' {
			lineEndings = append(lineEndings, lineEnding{lineNum: lineNum, isCRLF: prevWasCR})
			lineNum++
		}
		prevWasCR = (b == '\r')
	}

	// Count totals
	crlfCount := 0
	lfCount := 0
	for _, le := range lineEndings {
		if le.isCRLF {
			crlfCount++
		} else {
			lfCount++
		}
	}

	// Determine style and find inconsistent lines
	style := determineStyle(crlfCount, lfCount)
	var inconsistentLines []int

	if style == LineEndingMixed {
		// Dominant is the one with more occurrences
		dominantIsCRLF := crlfCount >= lfCount
		for _, le := range lineEndings {
			if le.isCRLF != dominantIsCRLF {
				inconsistentLines = append(inconsistentLines, le.lineNum)
			}
		}
	}

	// Total lines = last line number (includes last line even without trailing newline)
	totalLines := lineNum
	if len(lineEndings) > 0 {
		totalLines = lineNum // lineNum is 1 more than number of newlines found
	} else {
		totalLines = 1 // File has content but no newlines = 1 line
	}

	// Ensure we don't return nil slice (return empty array for JSON)
	if inconsistentLines == nil {
		inconsistentLines = []int{}
	}

	return &mcp.CallToolResult{}, DetectLineEndingsOutput{
		Style:             style,
		TotalLines:        totalLines,
		InconsistentLines: inconsistentLines,
	}, nil
}
