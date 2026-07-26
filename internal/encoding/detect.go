package encoding

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/wlynxg/chardet"
	"golang.org/x/text/encoding/charmap"
)

// Detection constants
const (
	ChunkSize               = 128 * 1024 // 128KB chunks for detection
	SmallFileThreshold      = 128 * 1024 // Files smaller than this are read entirely
	HighConfidenceThreshold = 80         // Confidence level to stop sampling early
	MinConfidenceThreshold  = 50         // Minimum confidence to trust detection
	utf8FallbackConfidence  = 80         // Confidence when UTF-8 is inferred from the bytes
	utf16NoBOMConfidence    = 90         // Confidence when BOM-less UTF-16 is inferred structurally
)

// UTF-16 column-structure detection.
const (
	utf16ScanMax  = 64 * 1024 // bytes examined for the column test (< ChunkSize keeps offsets even)
	utf16PageByte = 0x0F      // high bytes at or below this are plausible Unicode page bytes (Latin..Thai)
	utf16HighFrac = 0.80      // the high-byte column must be at least this fraction page bytes
	utf16LowFrac  = 0.40      // the low-byte column must be below this, so real text lives there
)

// GBK two-byte ranges: lead 0x81–0xFE, trail 0x40–0xFE except 0x7F.
const (
	gbkLeadMin       = 0x81
	gbkLeadMax       = 0xFE
	gbkTrailMin      = 0x40
	gbkTrailMax      = 0xFE
	gbkTrailGap      = 0x7F
	gbkConfidenceCap = 85 // cap when GBK is recovered from a Latin guess
)

// DetectionResult holds encoding detection result.
type DetectionResult struct {
	Charset    string
	Confidence int
	HasBOM     bool
}

// DetectBOM checks for Unicode BOMs and returns a result if found.
// Order matters: UTF-32 BOMs must be checked before UTF-16 since they share prefixes.
func DetectBOM(data []byte) (DetectionResult, bool) {
	if len(data) >= 4 {
		// UTF-32 BE: 00 00 FE FF
		if data[0] == 0x00 && data[1] == 0x00 && data[2] == 0xFE && data[3] == 0xFF {
			return DetectionResult{Charset: "utf-32-be", Confidence: 100, HasBOM: true}, true
		}
		// UTF-32 LE: FF FE 00 00
		if data[0] == 0xFF && data[1] == 0xFE && data[2] == 0x00 && data[3] == 0x00 {
			return DetectionResult{Charset: "utf-32-le", Confidence: 100, HasBOM: true}, true
		}
	}
	if len(data) >= 3 {
		// UTF-8 BOM: EF BB BF
		if data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
			return DetectionResult{Charset: "utf-8", Confidence: 100, HasBOM: true}, true
		}
	}
	if len(data) >= 2 {
		// UTF-16 BE: FE FF
		if data[0] == 0xFE && data[1] == 0xFF {
			return DetectionResult{Charset: "utf-16-be", Confidence: 100, HasBOM: true}, true
		}
		// UTF-16 LE: FF FE
		if data[0] == 0xFF && data[1] == 0xFE {
			return DetectionResult{Charset: "utf-16-le", Confidence: 100, HasBOM: true}, true
		}
	}
	return DetectionResult{}, false
}

// BOMBytesFor returns the BOM byte sequence for a given encoding name, or nil if unsupported.
// Supported: utf-8, utf-16-le, utf-16-be, utf-32-le, utf-32-be.
func BOMBytesFor(charset string) []byte {
	switch strings.ToLower(charset) {
	case "utf-8":
		return []byte{0xEF, 0xBB, 0xBF}
	case "utf-16-be":
		return []byte{0xFE, 0xFF}
	case "utf-16-le":
		return []byte{0xFF, 0xFE}
	case "utf-32-be":
		return []byte{0x00, 0x00, 0xFE, 0xFF}
	case "utf-32-le":
		return []byte{0xFF, 0xFE, 0x00, 0x00}
	default:
		return nil
	}
}

// BOMSize returns the byte length of a BOM for the given charset, or 0 if unknown.
func BOMSize(charset string) int {
	b := BOMBytesFor(charset)
	return len(b)
}

// --- Primary API (file-based, streaming) ---

// DetectFromFile detects encoding from a file path using streaming I/O.
// Modes: "sample" (~384KB max), "chunked" (streams entire file), "full" (loads entire file).
func DetectFromFile(path string, mode string) (DetectionResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return DetectionResult{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return DetectionResult{}, fmt.Errorf("failed to stat file: %w", err)
	}

	return detectFromReader(file, stat.Size(), mode)
}

// Detect detects encoding from a byte slice.
func Detect(data []byte) DetectionResult {
	if result, ok := DetectBOM(data); ok {
		return result
	}

	// BOM-less UTF-16 is structurally certain when it fires and gives the right
	// endianness; chardet misses non-Latin scripts (reports "ascii"/single-byte).
	if charset, ok := looksLikeUTF16(data); ok {
		return DetectionResult{Charset: charset, Confidence: utf16NoBOMConfidence}
	}

	detected := chardet.Detect(data)
	if detected.Encoding == "" {
		if utf8.Valid(data) {
			return DetectionResult{Charset: "utf-8", Confidence: utf8FallbackConfidence}
		}
		return DetectionResult{}
	}

	charset := strings.ToLower(detected.Encoding)
	confidence := int(detected.Confidence * 100)

	switch charset {
	case "gb2312", "hz-gb-2312":
		charset = "gbk" // GBK is the superset real-world files use
	case "iso-8859-1", "latin-1", "latin1", "windows-1252", "cp1252":
		// chardet often mislabels GBK as single-byte Latin; correct it.
		if looksLikeGBK(data) {
			return DetectionResult{Charset: "gbk", Confidence: min(confidence, gbkConfidenceCap)}
		}
	}

	// chardet guesses a single-byte charset for short or emoji-heavy UTF-8, which
	// would decode valid UTF-8 into mojibake. Legacy text is virtually never valid
	// UTF-8, so a well-formed multi-byte sequence outweighs the guess.
	if isSingleByteCharset(charset) && hasMultiByteUTF8(data) {
		return DetectionResult{Charset: "utf-8", Confidence: max(confidence, utf8FallbackConfidence)}
	}

	return DetectionResult{Charset: charset, Confidence: confidence}
}

// isSingleByteCharset reports whether name is a registered single-byte codec.
func isSingleByteCharset(name string) bool {
	canonical, ok := Canonical(name)
	if !ok {
		return false
	}
	_, isCharmap := encodings[canonical].Encoding.(*charmap.Charmap)
	return isCharmap
}

// hasMultiByteUTF8 reports whether data is valid UTF-8 and holds at least one
// multi-byte sequence. Pure ASCII returns false: every candidate agrees on it.
func hasMultiByteUTF8(data []byte) bool {
	if !utf8.Valid(data) {
		return false
	}
	for _, b := range data {
		if b >= utf8.RuneSelf {
			return true
		}
	}
	return false
}

// looksLikeUTF16 detects BOM-less UTF-16 from its column structure. Every code
// unit is a (low, high) byte pair; for text the high-byte column is dominated by
// small Unicode page bytes (0x00 for Latin, 0x04 for Cyrillic, 0x03 for Greek…)
// while the low-byte column carries the letters. Real single-byte and UTF-8 text
// never has a byte column that is 80%+ control-range bytes, so this is high
// precision. Callers pass even-aligned data (offset 0); we scan only the first
// 64KB so the parity holds. Returns the canonical charset when the signal fires.
func looksLikeUTF16(data []byte) (string, bool) {
	n := len(data) &^ 1 // even length; a stray final byte can't complete a unit
	if n > utf16ScanMax {
		n = utf16ScanMax
	}
	if n < 16 {
		return "", false // too little to distinguish from noise
	}
	units := n / 2
	var evenPage, oddPage int
	for i := 0; i < n; i += 2 {
		if data[i] <= utf16PageByte {
			evenPage++
		}
		if data[i+1] <= utf16PageByte {
			oddPage++
		}
	}
	evenFrac := float64(evenPage) / float64(units)
	oddFrac := float64(oddPage) / float64(units)
	switch {
	case oddFrac >= utf16HighFrac && evenFrac <= utf16LowFrac:
		return "utf-16-le", true // high bytes land on odd offsets
	case evenFrac >= utf16HighFrac && oddFrac <= utf16LowFrac:
		return "utf-16-be", true // high bytes land on even offsets
	default:
		return "", false
	}
}

// looksLikeGBK reports whether data holds enough valid GBK two-byte sequences,
// biased toward the common-hanzi lead range (0xB0–0xD7), to trust it over Latin.
func looksLikeGBK(data []byte) bool {
	const minSequences = 5
	const minCommonRatio = 0.2

	var total, common int
	for i := 0; i+1 < len(data); {
		lead, trail := data[i], data[i+1]
		if lead >= gbkLeadMin && lead <= gbkLeadMax &&
			trail >= gbkTrailMin && trail <= gbkTrailMax && trail != gbkTrailGap {
			total++
			if lead >= 0xB0 && lead <= 0xD7 {
				common++
			}
			i += 2
			continue
		}
		i++
	}

	return total >= minSequences && float64(common)/float64(total) > minCommonRatio
}

// DetectSample detects encoding by sampling beginning, middle, and end of data.
// Returns the result and whether it should be trusted.
// TODO: Make private or remove when grep.go and convert_encoding.go use streaming I/O.
func DetectSample(data []byte) (DetectionResult, bool) {
	size := len(data)

	if size <= SmallFileThreshold {
		result := Detect(data)
		return result, result.Confidence >= MinConfidenceThreshold
	}

	// Sample chunks from beginning, middle, and end
	var samples []byte

	// Beginning chunk
	endOfFirst := min(ChunkSize, size)
	samples = append(samples, data[:endOfFirst]...)

	// Check beginning first - if high confidence, return early
	result := Detect(samples)
	if result.Confidence >= HighConfidenceThreshold {
		return result, true
	}

	// Middle chunk
	if size > ChunkSize*2 {
		midStart := (size - ChunkSize) / 2
		midEnd := min(midStart+ChunkSize, size)
		samples = append(samples, data[midStart:midEnd]...)
	}

	// End chunk
	if size > ChunkSize {
		endStart := max(0, size-ChunkSize)
		samples = append(samples, data[endStart:]...)
	}

	result = Detect(samples)
	return result, result.Confidence >= MinConfidenceThreshold
}

// --- Internal streaming implementation ---

func detectFromReader(r io.ReaderAt, size int64, mode string) (DetectionResult, error) {
	switch mode {
	case "sample":
		return detectSampleFromReader(r, size)
	case "chunked":
		return detectChunkedFromReader(r, size)
	case "full":
		return detectFullFromReader(r, size)
	default:
		return DetectionResult{}, fmt.Errorf("invalid mode: %s (valid: sample, chunked, full)", mode)
	}
}

func detectSampleFromReader(r io.ReaderAt, size int64) (DetectionResult, error) {
	if size <= SmallFileThreshold {
		data := make([]byte, size)
		if _, err := r.ReadAt(data, 0); err != nil && err != io.EOF {
			return DetectionResult{}, fmt.Errorf("failed to read file: %w", err)
		}
		return Detect(data), nil
	}

	// Read beginning chunk
	beginChunk := make([]byte, ChunkSize)
	n, err := r.ReadAt(beginChunk, 0)
	if err != nil && err != io.EOF {
		return DetectionResult{}, fmt.Errorf("failed to read beginning: %w", err)
	}
	beginChunk = beginChunk[:n]

	if result, ok := DetectBOM(beginChunk); ok {
		return result, nil
	}

	// Check beginning chunk - if high confidence, return early
	result := Detect(beginChunk)
	if result.Confidence >= HighConfidenceThreshold {
		return result, nil
	}

	// Collect samples for combined detection
	samples := make([]byte, 0, ChunkSize*3)
	samples = append(samples, beginChunk...)

	// Middle chunk
	if size > int64(ChunkSize*2) {
		midStart := (size - int64(ChunkSize)) / 2
		midChunk := make([]byte, ChunkSize)
		n, err := r.ReadAt(midChunk, midStart)
		if err != nil && err != io.EOF {
			return DetectionResult{}, fmt.Errorf("failed to read middle: %w", err)
		}
		samples = append(samples, midChunk[:n]...)
	}

	// End chunk
	if size > int64(ChunkSize) {
		endStart := size - int64(ChunkSize)
		endChunk := make([]byte, ChunkSize)
		n, err := r.ReadAt(endChunk, endStart)
		if err != nil && err != io.EOF {
			return DetectionResult{}, fmt.Errorf("failed to read end: %w", err)
		}
		samples = append(samples, endChunk[:n]...)
	}

	return Detect(samples), nil
}

func detectChunkedFromReader(r io.ReaderAt, size int64) (DetectionResult, error) {
	if size <= int64(ChunkSize) {
		data := make([]byte, size)
		if _, err := r.ReadAt(data, 0); err != nil && err != io.EOF {
			return DetectionResult{}, fmt.Errorf("failed to read file: %w", err)
		}
		return Detect(data), nil
	}

	// Check for BOM (need 4 bytes for UTF-32)
	bomCheck := make([]byte, 4)
	if n, _ := r.ReadAt(bomCheck, 0); n >= 2 {
		if result, ok := DetectBOM(bomCheck[:n]); ok {
			return result, nil
		}
	}

	// Process file in chunks
	type chunkResult struct {
		encoding   string
		confidence int
		weight     int
	}

	var results []chunkResult
	chunk := make([]byte, ChunkSize)

	for offset := int64(0); offset < size; {
		n, err := r.ReadAt(chunk, offset)
		if err != nil && err != io.EOF {
			return DetectionResult{}, fmt.Errorf("failed to read chunk at %d: %w", offset, err)
		}
		if n == 0 {
			break
		}

		detected := Detect(chunk[:n])
		if detected.Charset != "" {
			results = append(results, chunkResult{
				encoding:   detected.Charset,
				confidence: detected.Confidence,
				weight:     n,
			})
		}
		offset += int64(n)
	}

	if len(results) == 0 {
		return DetectionResult{}, nil
	}

	// Aggregate results with weighted confidence
	encodingWeights := make(map[string]int)
	encodingConfidenceSum := make(map[string]int)

	for _, r := range results {
		encodingWeights[r.encoding] += r.weight
		encodingConfidenceSum[r.encoding] += r.confidence * r.weight
	}

	var bestEncoding string
	var bestWeight int
	for enc, weight := range encodingWeights {
		if weight > bestWeight {
			bestWeight = weight
			bestEncoding = enc
		}
	}

	return DetectionResult{
		Charset:    bestEncoding,
		Confidence: encodingConfidenceSum[bestEncoding] / encodingWeights[bestEncoding],
	}, nil
}

func detectFullFromReader(r io.ReaderAt, size int64) (DetectionResult, error) {
	data := make([]byte, size)
	if _, err := r.ReadAt(data, 0); err != nil && err != io.EOF {
		return DetectionResult{}, fmt.Errorf("failed to read file: %w", err)
	}
	return Detect(data), nil
}
