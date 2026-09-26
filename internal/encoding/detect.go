// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
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

// gbkConfidenceCap caps GBK when it is recovered from a Latin guess rather than named outright.
const gbkConfidenceCap = 85

// tableMargin keeps noise out: cp1251 and MacCyrillic read most Cyrillic alike, so a later table must beat an earlier one by this much.
const tableMargin = 0.05

// cyrillicCharsets are the tables Cyrillic is written in, cp1251 first because it is what the Windows world writes.
var cyrillicCharsets = []string{
	"windows-1251",
	"koi8-r",
	"koi8-u",
	"ibm866",
	"iso-8859-5",
	"x-mac-cyrillic",
}

type DetectionResult struct {
	Charset    string
	Confidence int
	HasBOM     bool
}

// Conclusive reports whether the result settles which encoding to use. "ascii" never does: it fits every encoding here.
func (d DetectionResult) Conclusive() bool {
	if d.Charset == "" || d.Charset == ASCII || d.Confidence < MinConfidenceThreshold {
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

// BOMBytesFor returns the BOM bytes for charset, or nil if unsupported.
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

func Detect(data []byte) DetectionResult {
	return decideVerdict(data,
		func() (DetectionResult, bool) { return detectUTF16Whole(data) },
		func() DetectionResult { return detectLegacy(data) })
}

// decideVerdict is the ladder every mode climbs: BOM, then structural UTF-16, then the legacy detector under the pin; modes differ only in the bytes they pass.
func decideVerdict(head []byte, utf16 func() (DetectionResult, bool), legacy func() DetectionResult) DetectionResult {
	if result, ok := DetectBOM(head); ok {
		return result
	}
	if result, handled := utf16(); handled && charsetAllowed(result.Charset) {
		return result
	}
	return legacy()
}

// detectUTF16Whole is the structural classifier behind its cheap gate; chardet never sees clean UTF-16.
func detectUTF16Whole(data []byte) (DetectionResult, bool) {
	if !mayContainUTF16(data) {
		return DetectionResult{}, false
	}
	return detectUTF16(data)
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

// detectLegacy is the chardet path for single-byte and other legacy codecs, and the one funnel every statistical verdict takes — so a pin is enforced here.
func detectLegacy(data []byte) DetectionResult {
	guess := guessLegacy(data)
	if charsetAllowed(guess.Charset) {
		return guess
	}
	// The guess fell outside the pin: exactly the statistical accident it exists to override.
	return pinnedVerdict(data)
}

// guessLegacy is the unrestricted chardet verdict.
func guessLegacy(data []byte) DetectionResult {
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

// correctCharset turns one detector label into a verdict; an empty name means no answer this server can act on.
func correctCharset(label string, confidence int, data []byte) (string, int) {
	charset, readable := detectorCharset(label)

	// Valid multi-byte UTF-8 outweighs any byte-table guess: legacy text is virtually never valid UTF-8.
	if (!readable || isSingleByteCharset(charset)) && hasMultiByteUTF8(data) {
		return "utf-8", max(confidence, utf8FallbackConfidence)
	}

	// A Latin table wins by default on mostly-ASCII text, and a label with no codec can only end in a garbled fallback, so both ask the bytes.
	if !readable || isLatinCharset(charset) {
		if cyrillic := cyrillicCodepage(data); cyrillic != "" {
			return cyrillic, confidence
		}
	}
	// The detector often mislabels GBK as single-byte Latin, and a Latin table reads almost any bytes, so only there.
	if isLatinCharset(charset) && scoreCharset("gbk", data).plausible() {
		return "gbk", min(confidence, gbkConfidenceCap)
	}
	if !readable {
		return "", 0
	}

	// MacCyrillic is the detector's catch-all for Cyrillic and all but extinct, so the bytes pick the table; overruling other Cyrillic labels measured worse.
	if charset == "x-mac-cyrillic" {
		if best, _ := bestCyrillicTable(data); best != "" {
			return best, confidence
		}
	}

	return charset, confidence
}

// isLatinCharset reports whether charset is a Western table, the default winner on mostly-ASCII text.
func isLatinCharset(charset string) bool {
	return charset == "iso-8859-1" || charset == "windows-1252"
}

// isSingleByteCharset reports whether name is a registered single-byte codec.
func isSingleByteCharset(name string) bool {
	return charmapFor(name) != nil
}

// charmapFor returns the byte table behind name, nil for multi-byte codecs and unknown names.
func charmapFor(name string) *charmap.Charmap {
	canonical, ok := Canonical(name)
	if !ok {
		return nil
	}
	cm, _ := encodings[canonical].Encoding.(*charmap.Charmap)
	return cm
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

// cyrillicCodepage names the Cyrillic table the bytes spell words in, or empty; overruling another script's label takes the plausibility bar.
func cyrillicCodepage(data []byte) string {
	name, best := bestCyrillicTable(data)
	if !best.plausible() {
		return ""
	}
	return name
}

// bestCyrillicTable ranks the Cyrillic tables by how much of the text each one reads as words.
func bestCyrillicTable(data []byte) (string, plausibility) {
	name, best, bestShare := "", plausibility{}, -1.0
	for _, charset := range cyrillicCharsets {
		if p := scoreCharset(charset, data); p.readable && p.share() > bestShare+tableMargin {
			name, best, bestShare = charset, p, p.share()
		}
	}
	return name, best
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
	return decideFromSamples(detectionSamplesFromData(data), int64(len(data)))
}

// decideFromSamples is the ladder over the sample chunks; the first sample starts at offset 0, so it carries any BOM.
func decideFromSamples(samples []byteSample, size int64) DetectionResult {
	return decideVerdict(samples[0].data,
		func() (DetectionResult, bool) { return detectUTF16Samples(samples, size) },
		func() DetectionResult { return legacyFromSamples(samples) })
}

// legacyFromSamples trusts the head only when it is conclusive, else reads every sample: an ASCII head would hide Cyrillic further in.
func legacyFromSamples(samples []byteSample) DetectionResult {
	if head := detectLegacy(samples[0].data); head.Conclusive() && head.Confidence >= HighConfidenceThreshold {
		return head
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
	for i, sample := range samples {
		joined = append(joined, utf8Whole(sample.data, i > 0, i < len(samples)-1)...)
	}
	return joined
}

// utf8Whole trims a UTF-8 sequence cut by a sample edge: one broken sequence makes the detector give up on valid UTF-8. Any other text loses at most three bytes.
func utf8Whole(data []byte, trimStart, trimEnd bool) []byte {
	if trimStart {
		for skipped := 0; skipped < utf8.UTFMax-1 && len(data) > 0 && !utf8.RuneStart(data[0]); skipped++ {
			data = data[1:]
		}
	}
	if trimEnd {
		for i := len(data) - 1; i >= 0 && i >= len(data)-(utf8.UTFMax-1); i-- {
			if utf8.RuneStart(data[i]) {
				if !utf8.FullRune(data[i:]) {
					data = data[:i]
				}
				break
			}
		}
	}
	return data
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

	head := make([]byte, 4) // longest BOM is UTF-32's
	n, _ := r.ReadAt(head, 0)

	scan := &chunkScan{r: r, size: size}
	result := decideVerdict(head[:n], scan.utf16, scan.vote)
	if scan.err != nil {
		return DetectionResult{}, scan.err
	}
	return result, nil
}

// chunkResult is one chunk's verdict and the bytes standing behind it.
type chunkResult struct {
	charset    string
	confidence int
	weight     int
}

// chunkScan walks the file once, feeding the UTF-16 analyzers and collecting a verdict per chunk. It runs on the ladder's first question, so a BOM means it never runs at all.
type chunkScan struct {
	r       io.ReaderAt
	size    int64
	done    bool
	err     error
	le, be  utf16Evidence
	results []chunkResult
}

// run reads the file once; every later call is free.
func (s *chunkScan) run() {
	if s.done {
		return
	}
	s.done = true

	le := newUTF16Analyzer(utf16LESpec)
	be := newUTF16Analyzer(utf16BESpec)
	chunk := make([]byte, ChunkSize)
	for offset := int64(0); offset < s.size; {
		n, err := s.r.ReadAt(chunk, offset)
		if err != nil && err != io.EOF {
			s.err = fmt.Errorf("failed to read chunk at %d: %w", offset, err)
			return
		}
		if n == 0 {
			break
		}
		data := chunk[:n]
		le.Write(data)
		be.Write(data)
		if detected := detectLegacy(data); detected.Charset != "" {
			// The detector can pass on plain ASCII, and the UTF-8 fallback must not make that evidence.
			if detected.Charset == "utf-8" && !slices.ContainsFunc(data, func(b byte) bool { return b >= utf8.RuneSelf }) {
				detected.Charset = ASCII
			}
			s.results = append(s.results, chunkResult{charset: detected.Charset, confidence: detected.Confidence, weight: n})
		}
		offset += int64(n)
	}
	s.le, s.be = le.Finish(), be.Finish()
}

// utf16 answers the structural question out of the single pass.
func (s *chunkScan) utf16() (DetectionResult, bool) {
	if s.run(); s.err != nil {
		return DetectionResult{}, false
	}
	return decideUTF16(s.le, s.be)
}

// vote weighs chunk verdicts by bytes, the name breaking ties for a stable answer; ASCII chunks count only when no chunk found anything else.
func (s *chunkScan) vote() DetectionResult {
	s.run()
	results := s.results
	if evidence := withEvidence(results); len(evidence) > 0 {
		results = evidence
	}
	if len(results) == 0 {
		return DetectionResult{}
	}

	weights := make(map[string]int)
	confidence := make(map[string]int)
	for _, result := range results {
		weights[result.charset] += result.weight
		confidence[result.charset] += result.confidence * result.weight
	}

	best, bestWeight := "", 0
	for charset, weight := range weights {
		if weight > bestWeight || (weight == bestWeight && charset < best) {
			best, bestWeight = charset, weight
		}
	}
	return DetectionResult{Charset: best, Confidence: confidence[best] / bestWeight}
}

// withEvidence drops the chunks that were plain ASCII, which reads the same under every encoding here.
func withEvidence(results []chunkResult) []chunkResult {
	evidence := make([]chunkResult, 0, len(results))
	for _, result := range results {
		if result.charset != ASCII {
			evidence = append(evidence, result)
		}
	}
	return evidence
}

func detectFullFromReader(r io.ReaderAt, size int64) (DetectionResult, error) {
	data := make([]byte, size)
	if _, err := r.ReadAt(data, 0); err != nil && err != io.EOF {
		return DetectionResult{}, fmt.Errorf("failed to read file: %w", err)
	}
	return Detect(data), nil
}
