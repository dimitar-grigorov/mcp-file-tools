// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

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

// Conclusive reports whether the result settles which encoding to use. "ascii" never does: it fits every encoding here.
func (d DetectionResult) Conclusive() bool {
	if d.Charset == "" || d.Charset == "ascii" || d.Confidence < MinConfidenceThreshold {
		return false
	}
	_, ok := Get(d.Charset)
	return ok
}

// DetectBOM checks for Unicode BOMs; UTF-32 goes before UTF-16 (shared prefixes).
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

// DetectFromFile detects encoding via streaming I/O; modes: sample (~384KB), chunked, full.
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

// mayContainUTF16 cheaply rules out clean UTF-8/ASCII; sub-0x80 UTF-16 shows up as C0-control soup.
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

	charset, confidence := correctCharset(strings.ToLower(detected.Encoding), int(detected.Confidence*100), data)
	if charset == "" {
		return DetectionResult{}
	}
	return DetectionResult{Charset: charset, Confidence: confidence}
}

// correctCharset fixes up one chardet verdict for both the single-verdict and ranked paths; an empty name means unusable.
func correctCharset(charset string, confidence int, data []byte) (string, int) {
	// BOM-less UTF-16 is accepted only by the structural classifier.
	if charset == "utf-16-le" || charset == "utf-16-be" || charset == "utf-16le" || charset == "utf-16be" {
		return "", 0
	}

	switch charset {
	case "gb2312", "hz-gb-2312":
		charset = "gbk" // GBK is the superset real-world files use
	case "iso-8859-1", "latin-1", "latin1", "windows-1252", "cp1252":
		// chardet often mislabels GBK as single-byte Latin; correct it.
		if looksLikeGBK(data) {
			return "gbk", min(confidence, gbkConfidenceCap)
		}
	case "maccyrillic", "x-mac-cyrillic":
		// chardet confuses MacCyrillic with Windows-1251; keep whichever decodes better.
		if cyrillicLetters(data, charmap.Windows1251) >= cyrillicLetters(data, charmap.MacintoshCyrillic) {
			return "windows-1251", confidence
		}
	}

	// Valid multi-byte UTF-8 outweighs a single-byte guess: legacy text is virtually never valid UTF-8.
	if isSingleByteCharset(charset) && hasMultiByteUTF8(data) {
		return "utf-8", max(confidence, utf8FallbackConfidence)
	}

	return charset, confidence
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

// hasMultiByteUTF8 reports valid UTF-8 with at least one multi-byte sequence; pure ASCII is false.
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

// cyrillicLetters counts high bytes that decode to modern Cyrillic letters under cm.
func cyrillicLetters(data []byte, cm *charmap.Charmap) int {
	n := 0
	for _, b := range data {
		if b < 0x80 {
			continue
		}
		if r := cm.DecodeByte(b); (r >= 'А' && r <= 'я') || r == 'Ё' || r == 'ё' {
			n++
		}
	}
	return n
}

// looksLikeGBK reports enough valid GBK pairs, biased toward common hanzi, to trust it over Latin.
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

// DetectSample samples beginning, middle and end; reports whether to trust it.
// TODO: make private once grep and convert_encoding stream instead of buffering.
func DetectSample(data []byte) (DetectionResult, bool) {
	result := detectSampleFromData(data)
	return result, result.Confidence >= MinConfidenceThreshold
}

func detectSampleFromData(data []byte) DetectionResult {
	if len(data) <= SmallFileThreshold {
		return Detect(data)
	}
	if result, ok := DetectBOM(data); ok {
		return result
	}
	return decideFromSamples(detectionSamplesFromData(data), int64(len(data)))
}

// decideFromSamples is the shared verdict over samples: UTF-16 structurally, else chardet on the head then on all of them.
func decideFromSamples(samples []byteSample, size int64) DetectionResult {
	if result, handled := detectUTF16Samples(samples, size); handled {
		return result
	}
	if result := detectLegacy(samples[0].data); result.Confidence >= HighConfidenceThreshold {
		return result
	}
	return detectLegacy(joinDetectionSamples(samples))
}

// detectionOffsets returns even-aligned begin/middle/end starts, shared so both paths sample the same bytes.
func detectionOffsets(size int64) []int64 {
	offsets := []int64{0}
	if size > ChunkSize*2 {
		middle := (size - ChunkSize) / 2
		offsets = append(offsets, middle-middle%2)
	}
	if size > ChunkSize {
		end := size - ChunkSize
		offsets = append(offsets, end-end%2)
	}
	return offsets
}

// detectionSamplesFromData slices the sample chunks out of an in-memory buffer.
func detectionSamplesFromData(data []byte) []byteSample {
	size := int64(len(data))
	offsets := detectionOffsets(size)
	samples := make([]byteSample, 0, len(offsets))
	for i, offset := range offsets {
		end := min(offset+ChunkSize, size)
		if i == len(offsets)-1 {
			end = size // final sample runs to EOF
		}
		samples = append(samples, byteSample{data: data[offset:end], offset: offset})
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
	return decideFromSamples(samples, size), nil
}

// readDetectionSamples reads the sample chunks through a ReaderAt.
func readDetectionSamples(r io.ReaderAt, size int64) ([]byteSample, error) {
	offsets := detectionOffsets(size)
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

	// Name breaks a tie: ranging a map alone would pick a different one per run.
	var bestEncoding string
	var bestWeight int
	for enc, weight := range encodingWeights {
		if weight > bestWeight || (weight == bestWeight && enc < bestEncoding) {
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
