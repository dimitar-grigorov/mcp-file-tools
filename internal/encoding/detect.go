package encoding

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/wlynxg/chardet"
)

// Detection constants
const (
	ChunkSize               = 128 * 1024 // 128KB chunks for detection
	SmallFileThreshold      = 128 * 1024 // Files smaller than this are read entirely
	HighConfidenceThreshold = 80         // Confidence level to stop sampling early
	MinConfidenceThreshold  = 50         // Minimum confidence to trust detection
)

// DetectionResult holds encoding detection result.
type DetectionResult struct {
	Charset    string
	Confidence int
	HasBOM     bool
}

// detectBOM checks for Unicode BOMs and returns a result if found.
// Order matters: UTF-32 BOMs must be checked before UTF-16 since they share prefixes.
func detectBOM(data []byte) (DetectionResult, bool) {
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
	if result, ok := detectBOM(data); ok {
		return result
	}

	detected := chardet.Detect(data)
	if detected.Encoding == "" {
		if utf8.Valid(data) {
			return DetectionResult{Charset: "utf-8", Confidence: 80}
		}
		return DetectionResult{}
	}

	return DetectionResult{
		Charset:    strings.ToLower(detected.Encoding),
		Confidence: int(detected.Confidence * 100),
	}
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

	if result, ok := detectBOM(beginChunk); ok {
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
		if result, ok := detectBOM(bomCheck[:n]); ok {
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
