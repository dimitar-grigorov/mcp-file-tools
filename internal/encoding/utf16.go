package encoding

import (
	"encoding/binary"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// BOM-less UTF-16 is classified by decoding as both endiannesses and scoring
// which reading yields well-formed, printable text with a dominant page column.
const (
	utf16MinRunes          = 4
	utf16MinTextPercent    = 95
	utf16MaxControlPercent = 2
	utf16MinScore          = 82
	utf16MinScoreLead      = 12
	utf16MinPageDominance  = 0.60 // real scripts keep the letter column out of ≤0x0F (Cyrillic ~0.9, ASCII ~0.98); binary noise sits well below
	utf16PageByte          = 0x0F // high-byte column values at or below this mark a BMP page (Latin 0x00, Cyrillic 0x04, Greek 0x03…)
)

type utf16Spec struct {
	charset          string
	order            binary.ByteOrder
	expectedZeroByte int // pair index that is 0x00 for BMP text (1 for LE, 0 for BE)
}

var (
	utf16LESpec = utf16Spec{charset: "utf-16-le", order: binary.LittleEndian, expectedZeroByte: 1}
	utf16BESpec = utf16Spec{charset: "utf-16-be", order: binary.BigEndian, expectedZeroByte: 0}
)

// byteSample is a chunk of file data with its absolute offset.
type byteSample struct {
	data   []byte
	offset int64
}

type utf16Evidence struct {
	charset string

	unitCount        int
	runeCount        int
	goodRuneCount    int
	controlCount     int
	nulCount         int
	replacementRunes int
	noncharacters    int
	surrogatePairs   int
	expectedZeros    int
	unexpectedZeros  int
	expectedPage     int
	unexpectedPage   int

	structuralValid bool
	roundTrip       bool
}

// utf16Analyzer consumes bytes incrementally so a file can be scored across
// streamed chunks without buffering it whole.
type utf16Analyzer struct {
	spec     utf16Spec
	evidence utf16Evidence

	pendingByte    byte
	hasPendingByte bool
	pendingHigh    uint16
	hasPendingHigh bool
}

func newUTF16Analyzer(spec utf16Spec) *utf16Analyzer {
	return &utf16Analyzer{
		spec: spec,
		evidence: utf16Evidence{
			charset:         spec.charset,
			structuralValid: true,
			roundTrip:       true,
		},
	}
}

func (a *utf16Analyzer) Write(data []byte) {
	for _, current := range data {
		if !a.hasPendingByte {
			a.pendingByte = current
			a.hasPendingByte = true
			continue
		}
		first := a.pendingByte
		a.hasPendingByte = false
		a.consumePair(first, current)
	}
}

func (a *utf16Analyzer) consumePair(first, second byte) {
	a.evidence.unitCount++

	pair := [2]byte{first, second}
	if pair[a.spec.expectedZeroByte] == 0 {
		a.evidence.expectedZeros++
	}
	if pair[1-a.spec.expectedZeroByte] == 0 {
		a.evidence.unexpectedZeros++
	}
	if pair[a.spec.expectedZeroByte] <= utf16PageByte {
		a.evidence.expectedPage++
	}
	if pair[1-a.spec.expectedZeroByte] <= utf16PageByte {
		a.evidence.unexpectedPage++
	}

	unit := a.spec.order.Uint16(pair[:])

	if a.hasPendingHigh {
		if unit >= 0xDC00 && unit <= 0xDFFF {
			high := a.pendingHigh
			a.hasPendingHigh = false
			decoded := utf16.DecodeRune(rune(high), rune(unit))
			encodedHigh, encodedLow := utf16.EncodeRune(decoded)
			if uint16(encodedHigh) != high || uint16(encodedLow) != unit {
				a.evidence.roundTrip = false
			}
			a.evidence.surrogatePairs++
			a.observeRune(decoded)
			return
		}
		a.evidence.structuralValid = false
		a.hasPendingHigh = false
	}

	switch {
	case unit >= 0xD800 && unit <= 0xDBFF:
		a.pendingHigh = unit
		a.hasPendingHigh = true
	case unit >= 0xDC00 && unit <= 0xDFFF:
		a.evidence.structuralValid = false
	default:
		decoded := rune(unit)
		if uint16(decoded) != unit {
			a.evidence.roundTrip = false
		}
		a.observeRune(decoded)
	}
}

func (a *utf16Analyzer) observeRune(value rune) {
	a.evidence.runeCount++

	switch {
	case value == utf8.RuneError:
		a.evidence.replacementRunes++
	case value == 0:
		a.evidence.nulCount++
	case isUnicodeNoncharacter(value):
		a.evidence.noncharacters++
	case value == '\n' || value == '\r' || value == '\t' || unicode.IsSpace(value):
		a.evidence.goodRuneCount++
	case unicode.IsPrint(value):
		a.evidence.goodRuneCount++
	case unicode.IsControl(value):
		a.evidence.controlCount++
	}
}

func (a *utf16Analyzer) Finish() utf16Evidence {
	if a.hasPendingByte {
		a.evidence.structuralValid = false
		a.hasPendingByte = false
	}
	if a.hasPendingHigh {
		a.evidence.structuralValid = false
		a.hasPendingHigh = false
	}
	return a.evidence
}

func isUnicodeNoncharacter(value rune) bool {
	return value >= 0xFDD0 && value <= 0xFDEF || value >= 0 && value <= utf8.MaxRune && value&0xFFFF >= 0xFFFE
}

func analyzeUTF16(data []byte, spec utf16Spec) utf16Evidence {
	analyzer := newUTF16Analyzer(spec)
	analyzer.Write(data)
	return analyzer.Finish()
}

// analyzeUTF16Samples scores discontiguous samples, trimming each so partial
// units and split surrogate pairs at sample edges do not skew the evidence.
func analyzeUTF16Samples(samples []byteSample, totalSize int64, spec utf16Spec) utf16Evidence {
	combined := utf16Evidence{charset: spec.charset, structuralValid: true, roundTrip: true}
	for _, sample := range samples {
		data := sample.data
		offset := sample.offset

		if offset%2 != 0 && len(data) > 0 {
			data = data[1:]
			offset++
		}
		if len(data)%2 != 0 && offset+int64(len(data)) < totalSize {
			data = data[:len(data)-1]
		}
		if offset > 0 && len(data) >= 2 {
			first := spec.order.Uint16(data[:2])
			if first >= 0xDC00 && first <= 0xDFFF {
				data = data[2:]
				offset += 2
			}
		}
		if offset+int64(len(data)) < totalSize && len(data) >= 2 {
			last := spec.order.Uint16(data[len(data)-2:])
			if last >= 0xD800 && last <= 0xDBFF {
				data = data[:len(data)-2]
			}
		}

		mergeUTF16Evidence(&combined, analyzeUTF16(data, spec))
	}
	if totalSize%2 != 0 {
		combined.structuralValid = false
	}
	return combined
}

func mergeUTF16Evidence(target *utf16Evidence, source utf16Evidence) {
	target.unitCount += source.unitCount
	target.runeCount += source.runeCount
	target.goodRuneCount += source.goodRuneCount
	target.controlCount += source.controlCount
	target.nulCount += source.nulCount
	target.replacementRunes += source.replacementRunes
	target.noncharacters += source.noncharacters
	target.surrogatePairs += source.surrogatePairs
	target.expectedZeros += source.expectedZeros
	target.unexpectedZeros += source.unexpectedZeros
	target.expectedPage += source.expectedPage
	target.unexpectedPage += source.unexpectedPage
	target.structuralValid = target.structuralValid && source.structuralValid
	target.roundTrip = target.roundTrip && source.roundTrip
}

// eligible gates evidence to well-formed, overwhelmingly printable text.
func (e utf16Evidence) eligible() bool {
	if !e.structuralValid || !e.roundTrip || e.unitCount < utf16MinRunes || e.runeCount < utf16MinRunes {
		return false
	}
	if e.nulCount != 0 || e.replacementRunes != 0 || e.noncharacters != 0 {
		return false
	}
	if e.goodRuneCount*100 < e.runeCount*utf16MinTextPercent {
		return false
	}
	return e.controlCount*100 <= e.runeCount*utf16MaxControlPercent
}

// pageDominance is how strongly the expected high-byte column sits in the page
// range (≤0x0F) versus the other; near zero means ambiguous or non-BMP (e.g. CJK).
func (e utf16Evidence) pageDominance() float64 {
	if e.unitCount == 0 {
		return 0
	}
	return float64(e.expectedPage-e.unexpectedPage) / float64(e.unitCount)
}

func (e utf16Evidence) score() int {
	if e.runeCount == 0 {
		return 0
	}
	quality := e.goodRuneCount * 50 / e.runeCount
	pageBonus := int(maxFloat64(0, e.pageDominance()) * 30)
	if pageBonus > 30 {
		pageBonus = 30
	}
	lengthBonus := e.runeCount / 8
	if lengthBonus > 10 {
		lengthBonus = 10
	}
	surrogateBonus := e.surrogatePairs
	if surrogateBonus > 3 {
		surrogateBonus = 3
	}
	return 20 + quality + pageBonus + lengthBonus + surrogateBonus
}

func detectUTF16(data []byte) (DetectionResult, bool) {
	le := analyzeUTF16(data, utf16LESpec)
	be := analyzeUTF16(data, utf16BESpec)
	return decideUTF16(le, be)
}

func detectUTF16Samples(samples []byteSample, totalSize int64) (DetectionResult, bool) {
	le := analyzeUTF16Samples(samples, totalSize, utf16LESpec)
	be := analyzeUTF16Samples(samples, totalSize, utf16BESpec)
	return decideUTF16(le, be)
}

// decideUTF16 picks an endianness only when one reading is eligible, scores high
// enough, and (when both are) clearly beats the other. On a raw zero-column
// signal without a clean decode it returns handled with an empty result, so the
// caller reports "undetected" rather than letting chardet mislabel the bytes.
func decideUTF16(le, be utf16Evidence) (DetectionResult, bool) {
	leEligible := le.eligible()
	beEligible := be.eligible()
	rawSignal := hasUTF16RawSignal(le, be)

	if !leEligible && !beEligible {
		return DetectionResult{}, rawSignal
	}

	if leEligible && beEligible {
		winner, loser := le, be
		if be.score() > le.score() {
			winner, loser = be, le
		}
		if winner.score() >= utf16MinScore && winner.score()-loser.score() >= utf16MinScoreLead && winner.pageDominance() >= utf16MinPageDominance {
			return utf16Result(winner), true
		}
		return DetectionResult{}, rawSignal
	}

	winner := le
	if beEligible {
		winner = be
	}
	if winner.score() >= utf16MinScore && winner.pageDominance() >= utf16MinPageDominance {
		return utf16Result(winner), true
	}
	return DetectionResult{}, rawSignal
}

// hasUTF16RawSignal reports whether one byte column is so dominated by zeros that
// the bytes can only be UTF-16, even when they do not decode to clean text.
func hasUTF16RawSignal(le, be utf16Evidence) bool {
	units := le.unitCount
	if be.unitCount > units {
		units = be.unitCount
	}
	if units == 0 {
		return false
	}

	evenZeros := be.expectedZeros
	oddZeros := le.expectedZeros
	maxZeros, minZeros := evenZeros, oddZeros
	if oddZeros > evenZeros {
		maxZeros, minZeros = oddZeros, evenZeros
	}
	maxRatio := float64(maxZeros) / float64(units)
	dominance := float64(maxZeros-minZeros) / float64(units)

	if units >= 2 && maxRatio >= 0.5 && dominance >= 0.5 {
		return true
	}
	return units >= 4 && maxRatio >= 0.2 && dominance >= 0.15
}

func utf16Result(evidence utf16Evidence) DetectionResult {
	confidence := evidence.score()
	if confidence < HighConfidenceThreshold {
		confidence = HighConfidenceThreshold
	}
	if confidence > 95 {
		confidence = 95
	}
	return DetectionResult{Charset: evidence.charset, Confidence: confidence}
}

func maxFloat64(first, second float64) float64 {
	if first > second {
		return first
	}
	return second
}
