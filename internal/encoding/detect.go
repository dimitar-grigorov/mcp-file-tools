package encoding

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/wlynxg/chardet"
	"golang.org/x/text/encoding/charmap"
)

const (
	ChunkSize               = 128 * 1024 // 128KB chunks for detection
	SmallFileThreshold      = 128 * 1024 // Files smaller than this are read entirely
	HighConfidenceThreshold = 80         // Confidence level to stop sampling early
	MinConfidenceThreshold  = 50         // Minimum confidence to trust detection
	utf8FallbackConfidence  = 80         // Confidence when UTF-8 is inferred from the bytes
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
		if data[0] == 0x00 && data[1] == 0x00 && data[2] == 0xFE && data[3] == 0xFF {
			return DetectionResult{Charset: "utf-32-be", Confidence: 100, HasBOM: true}, true
		}
		if data[0] == 0xFF && data[1] == 0xFE && data[2] == 0x00 && data[3] == 0x00 {
			return DetectionResult{Charset: "utf-32-le", Confidence: 100, HasBOM: true}, true
		}
	}
	if len(data) >= 3 {
		if data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
			return DetectionResult{Charset: "utf-8", Confidence: 100, HasBOM: true}, true
		}
	}
	if len(data) >= 2 {
		if data[0] == 0xFE && data[1] == 0xFF {
			return DetectionResult{Charset: "utf-16-be", Confidence: 100, HasBOM: true}, true
		}
		if data[0] == 0xFF && data[1] == 0xFE {
			return DetectionResult{Charset: "utf-16-le", Confidence: 100, HasBOM: true}, true
		}
	}
	return DetectionResult{}, false
}

// BOMBytesFor returns the BOM byte sequence for a given encoding name, or nil if unsupported.
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

	// BOM-less UTF-16 is classified structurally; chardet never sees clean UTF-16.
	if mayContainUTF16(data) {
		if result, handled := detectUTF16(data); handled {
			return result
		}
	}
	return detectLegacy(data)
}

// mayContainUTF16 cheaply rules out clean UTF-8/ASCII before the structural pass.
// A NUL or invalid UTF-8 is the obvious signal; sub-0x80 UTF-16 (Cyrillic 0x04xx,
// Greek 0x03xx…) is valid-UTF-8 control soup, so a high C0-control fraction counts too.
func mayContainUTF16(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return true
	}
	controls := 0
	for _, b := range data {
		if b < 0x20 && b != '\t' && b != '\n' && b != '\r' {
			controls++
		}
	}
	return controls*100 >= len(data)*20
}

// detectLegacy is the chardet-based path for single-byte and other legacy codecs.
func detectLegacy(data []byte) DetectionResult {
	detected := chardet.Detect(data)
	if detected.Encoding == "" {
		if utf8.Valid(data) {
			return DetectionResult{Charset: "utf-8", Confidence: utf8FallbackConfidence}
		}
		return DetectionResult{}
	}

	charset := strings.ToLower(detected.Encoding)
	confidence := int(detected.Confidence * 100)

	// BOM-less UTF-16 is accepted only by the structural classifier above.
	if charset == "utf-16-le" || charset == "utf-16-be" || charset == "utf-16le" || charset == "utf-16be" {
		return DetectionResult{}
	}

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
	if result, ok := DetectBOM(data); ok {
		return result, true
	}

	samples := detectionSamplesFromData(data)
	if result, handled := detectUTF16Samples(samples, int64(size)); handled {
		return result, result.Confidence >= MinConfidenceThreshold
	}

	result := detectLegacy(samples[0].data)
	if result.Confidence >= HighConfidenceThreshold {
		return result, true
	}
	result = detectLegacy(joinDetectionSamples(samples))
	return result, result.Confidence >= MinConfidenceThreshold
}

// detectionSamplesFromData slices even-aligned beginning, middle, and end chunks.
func detectionSamplesFromData(data []byte) []byteSample {
	size := len(data)
	samples := []byteSample{{data: data[:min(ChunkSize, size)], offset: 0}}

	if size > ChunkSize*2 {
		middle := (size - ChunkSize) / 2
		middle -= middle % 2
		samples = append(samples, byteSample{data: data[middle : middle+ChunkSize], offset: int64(middle)})
	}
	if size > ChunkSize {
		end := size - ChunkSize
		end -= end % 2
		samples = append(samples, byteSample{data: data[end:], offset: int64(end)})
	}
	return samples
}

func joinDetectionSamples(samples []byteSample) []byte {
	total := 0
	for _, sample := range samples {
		total += len(sample.data)
	}
	joined := make([]byte, 0, total)
	for _, sample := range samples {
		joined = append(joined, sample.data...)
	}
	return joined
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

	samples, err := readDetectionSamples(r, size)
	if err != nil {
		return DetectionResult{}, err
	}
	if result, ok := DetectBOM(samples[0].data); ok {
		return result, nil
	}
	if result, handled := detectUTF16Samples(samples, size); handled {
		return result, nil
	}

	result := detectLegacy(samples[0].data)
	if result.Confidence >= HighConfidenceThreshold {
		return result, nil
	}
	return detectLegacy(joinDetectionSamples(samples)), nil
}

// readDetectionSamples reads even-aligned beginning, middle, and end samples.
func readDetectionSamples(r io.ReaderAt, size int64) ([]byteSample, error) {
	offsets := []int64{0}
	if size > int64(ChunkSize*2) {
		middle := (size - int64(ChunkSize)) / 2
		middle -= middle % 2
		offsets = append(offsets, middle)
	}
	if size > int64(ChunkSize) {
		end := size - int64(ChunkSize)
		end -= end % 2
		offsets = append(offsets, end)
	}

	samples := make([]byteSample, 0, len(offsets))
	for i, offset := range offsets {
		length := min(int64(ChunkSize), size-offset)
		if i == len(offsets)-1 {
			length = size - offset // final sample runs to EOF
		}
		data := make([]byte, int(length))
		n, err := r.ReadAt(data, offset)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read sample at %d: %w", offset, err)
		}
		samples = append(samples, byteSample{data: data[:n], offset: offset})
	}
	return samples, nil
}

func detectChunkedFromReader(r io.ReaderAt, size int64) (DetectionResult, error) {
	if size <= int64(ChunkSize) {
		data := make([]byte, size)
		if _, err := r.ReadAt(data, 0); err != nil && err != io.EOF {
			return DetectionResult{}, fmt.Errorf("failed to read file: %w", err)
		}
		return Detect(data), nil
	}

	bomCheck := make([]byte, 4) // longest BOM is UTF-32's
	if n, _ := r.ReadAt(bomCheck, 0); n >= 2 {
		if result, ok := DetectBOM(bomCheck[:n]); ok {
			return result, nil
		}
	}

	type chunkResult struct {
		encoding   string
		confidence int
		weight     int
	}

	leAnalyzer := newUTF16Analyzer(utf16LESpec)
	beAnalyzer := newUTF16Analyzer(utf16BESpec)
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

		data := chunk[:n]
		leAnalyzer.Write(data)
		beAnalyzer.Write(data)
		detected := detectLegacy(data)
		if detected.Charset != "" {
			results = append(results, chunkResult{
				encoding:   detected.Charset,
				confidence: detected.Confidence,
				weight:     n,
			})
		}
		offset += int64(n)
	}

	if result, handled := decideUTF16(leAnalyzer.Finish(), beAnalyzer.Finish()); handled {
		return result, nil
	}
	if len(results) == 0 {
		return DetectionResult{}, nil
	}

	// Weight each chunk's verdict by its byte count.
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
