package encoding

import (
	"strings"
	"unicode/utf8"

	"github.com/wlynxg/chardet"
)

const (
	// ChunkSize is the size of chunks to read for encoding detection
	ChunkSize = 128 * 1024 // 128KB
	// SmallFileThreshold is the max size to read entirely for detection
	SmallFileThreshold = 128 * 1024 // 128KB
	// HighConfidenceThreshold is the confidence level to stop sampling
	HighConfidenceThreshold = 80
	// MinConfidenceThreshold is the minimum confidence to use detected encoding
	MinConfidenceThreshold = 50
)

// DetectionResult holds encoding detection result.
type DetectionResult struct {
	Charset    string
	Confidence int
	HasBOM     bool
}

// Detect detects the encoding of the given data.
func Detect(data []byte) DetectionResult {
	// Check UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return DetectionResult{Charset: "utf-8", Confidence: 100, HasBOM: true}
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
// For small files (< 128KB), it uses all data.
// For larger files, it samples beginning, middle, and end (3 samples).
// Returns the detection result and whether the detected encoding should be trusted.
func DetectSample(data []byte) (DetectionResult, bool) {
	fileSize := len(data)

	// For small files, detect on entire content
	if fileSize <= SmallFileThreshold {
		result := Detect(data)
		trusted := result.Confidence >= MinConfidenceThreshold
		return result, trusted
	}

	// For larger files, sample chunks from beginning, middle, and end
	var samples []byte

	// Beginning chunk
	endOfFirst := ChunkSize
	if endOfFirst > fileSize {
		endOfFirst = fileSize
	}
	samples = append(samples, data[:endOfFirst]...)

	// Check beginning chunk first - if high confidence, use it
	result := Detect(samples)
	if result.Confidence >= HighConfidenceThreshold {
		return result, true
	}

	// Middle chunk (if file is large enough)
	if fileSize > ChunkSize*2 {
		midStart := (fileSize - ChunkSize) / 2
		midEnd := midStart + ChunkSize
		if midEnd > fileSize {
			midEnd = fileSize
		}
		samples = append(samples, data[midStart:midEnd]...)
	}

	// End chunk (if file is large enough)
	if fileSize > ChunkSize {
		endStart := fileSize - ChunkSize
		if endStart < 0 {
			endStart = 0
		}
		samples = append(samples, data[endStart:]...)
	}

	// Detect on combined samples
	result = Detect(samples)
	trusted := result.Confidence >= MinConfidenceThreshold
	return result, trusted
}

// DetectFromChunks is an alias for DetectSample for backwards compatibility.
func DetectFromChunks(data []byte) (DetectionResult, bool) {
	return DetectSample(data)
}

// DetectChunked detects encoding by reading all chunks and calculating weighted average confidence.
// Each chunk is detected independently, and results are aggregated.
// Weight is based on chunk size (larger chunks have more weight).
// Returns the detection result with weighted average confidence.
func DetectChunked(data []byte) DetectionResult {
	fileSize := len(data)

	// For small files, detect on entire content
	if fileSize <= ChunkSize {
		return Detect(data)
	}

	// Check for BOM first (only in first chunk)
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return DetectionResult{Charset: "utf-8", Confidence: 100, HasBOM: true}
	}

	// Process file in chunks
	type chunkResult struct {
		encoding   string
		confidence int
		weight     int // chunk size as weight
	}

	var results []chunkResult
	offset := 0

	for offset < fileSize {
		end := offset + ChunkSize
		if end > fileSize {
			end = fileSize
		}

		chunk := data[offset:end]
		chunkSize := len(chunk)

		detected := Detect(chunk)
		if detected.Charset != "" {
			results = append(results, chunkResult{
				encoding:   detected.Charset,
				confidence: detected.Confidence,
				weight:     chunkSize,
			})
		}

		offset = end
	}

	if len(results) == 0 {
		return DetectionResult{}
	}

	// Find the most common encoding
	encodingCounts := make(map[string]int)
	encodingWeights := make(map[string]int)
	encodingConfidenceSum := make(map[string]int)

	for _, r := range results {
		encodingCounts[r.encoding]++
		encodingWeights[r.encoding] += r.weight
		encodingConfidenceSum[r.encoding] += r.confidence * r.weight
	}

	// Find encoding with highest total weight
	var bestEncoding string
	var bestWeight int
	for enc, weight := range encodingWeights {
		if weight > bestWeight {
			bestWeight = weight
			bestEncoding = enc
		}
	}

	// Calculate weighted average confidence for the best encoding
	weightedConfidence := encodingConfidenceSum[bestEncoding] / encodingWeights[bestEncoding]

	return DetectionResult{
		Charset:    bestEncoding,
		Confidence: weightedConfidence,
	}
}
