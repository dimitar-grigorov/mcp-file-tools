package encoding

import (
	"testing"

	"golang.org/x/text/encoding/charmap"
)

// chardet guesses a single-byte charset for short or emoji-heavy UTF-8. Decoding
// those bytes as CP1252 turns "a​b" into "aâ€‹b", the mojibake class users
// report against UTF-8-only tooling.
func TestDetect_ValidUTF8BeatsSingleByteGuess(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{"zero width space", "a​b"},
		{"emoji in a sentence", "status: 🔴 offline, 🟢 online, retry in 5s please wait"},
		{"nordic", "æ ø å"},
		{"math symbols", "≈ ≠ ±"},
		{"astral only", "🔴 🈶"},
		{"cyrillic", "Ще проверим"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Detect([]byte(tc.text))
			if !IsUTF8(got.Charset) {
				t.Errorf("Detect(%q) = %s, want utf-8", tc.text, got.Charset)
			}
		})
	}
}

func TestDetect_RealLegacyBytesStaySingleByte(t *testing.T) {
	cases := []struct {
		name    string
		charmap *charmap.Charmap
		text    string
	}{
		{"cp1251 cyrillic", charmap.Windows1251, "Ще проверим това"},
		{"cp1251 russian", charmap.Windows1251, "Привет мир"},
		{"cp1252 nordic", charmap.Windows1252, "Jeg heter Kåre og bor i Tromsø"},
		{"cp1252 french", charmap.Windows1252, "café au lait"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.charmap.NewEncoder().Bytes([]byte(tc.text))
			if err != nil {
				t.Fatal(err)
			}

			got := Detect(data)
			if IsUTF8(got.Charset) {
				t.Errorf("Detect(%q as %s) = utf-8, want a single-byte charset", tc.text, tc.name)
			}
		})
	}
}

func TestHasMultiByteUTF8(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want bool
	}{
		{"empty", nil, false},
		{"pure ascii", []byte("hello world"), false},
		{"two byte sequence", []byte("å"), true},
		{"three byte sequence", []byte("​"), true},
		{"four byte sequence", []byte("🔴"), true},
		{"invalid utf-8", []byte{0xE5, 0x20, 0xF8}, false},
		{"truncated sequence", []byte{0xC3}, false},
		{"double encoded is still valid utf-8", []byte{0xC3, 0x83, 0xC2, 0xA5}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasMultiByteUTF8(tc.data); got != tc.want {
				t.Errorf("hasMultiByteUTF8(% x) = %v, want %v", tc.data, got, tc.want)
			}
		})
	}
}

func TestIsSingleByteCharset(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"windows-1252", true},
		{"cp1251", true},
		{"iso-8859-1", true},
		{"koi8-r", true},
		{"utf-8", false},
		{"utf-16-le", false},
		{"gbk", false},
		{"gb18030", false},
		{"not-a-charset", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isSingleByteCharset(tc.name); got != tc.want {
				t.Errorf("isSingleByteCharset(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
