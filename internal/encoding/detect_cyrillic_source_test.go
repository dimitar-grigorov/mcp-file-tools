// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"
)

var russianLiterals = []string{"Не удалось подключиться к серверу.", "Греция"}

// sparseCyrillicSource builds a source unit that is ASCII apart from a few literals, fillerLines setting how thinly the Cyrillic is spread.
func sparseCyrillicSource(t *testing.T, cm *charmap.Charmap, fillerLines int, literals ...string) []byte {
	t.Helper()

	if len(literals) == 0 {
		literals = []string{"Неуспешна връзка с Sroutz GR. Опитайте по-късно.", "Гърция"}
	}

	var b strings.Builder
	b.WriteString("{$I CompMode.inc}\r\n\r\nunit SkroutzTools;\r\n\r\ninterface\r\nuses\r\n")
	b.WriteString("  CSDateUtils, Classes, BarTypes, IBDatabase, Controls, SuperObject;\r\n\r\n")
	for range fillerLines {
		b.WriteString("procedure DoSomethingUseful(const AValue: Integer; out AResult: String);\r\n")
	}
	b.WriteString("resourcestring\r\n")
	for i, literal := range literals {
		fmt.Fprintf(&b, "  MSG_%d = '", i)
		b.Write(charmapEncode(t, cm, literal))
		b.WriteString("';\r\n")
	}
	b.WriteString("end.\r\n")
	return []byte(b.String())
}

// The thinner the Cyrillic, the further chardet slides down its Latin tables: Windows-1251, then ISO-8859-1, then MacRoman.
func TestDetect_SparseCyrillicInSource(t *testing.T) {
	for _, fillerLines := range []int{40, 120, 240, 340} {
		t.Run(fmt.Sprintf("filler=%d", fillerLines), func(t *testing.T) {
			data := sparseCyrillicSource(t, charmap.Windows1251, fillerLines)

			result := Detect(data)
			if result.Charset != "windows-1251" {
				t.Errorf("Charset = %q, want windows-1251", result.Charset)
			}
			if !result.Conclusive() {
				t.Errorf("result %+v is not conclusive, so the file is read as the fallback and garbled", result)
			}
		})
	}
}

// Cyrillic in the other codepages must keep its own, not be pulled to the commonest one.
func TestDetect_SparseCyrillicKeepsItsOwnCodepage(t *testing.T) {
	tests := []struct {
		cm   *charmap.Charmap
		want string
	}{
		{charmap.KOI8R, "koi8-r"},
		{charmap.CodePage866, "ibm866"},
		{charmap.ISO8859_5, "iso-8859-5"},
		{charmap.MacintoshCyrillic, "x-mac-cyrillic"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			result := Detect(sparseCyrillicSource(t, tt.cm, 340, russianLiterals...))
			canonical, ok := Canonical(result.Charset)
			if !ok || canonical != tt.want {
				t.Errorf("Charset = %q (canonical %q), want %s", result.Charset, canonical, tt.want)
			}
		})
	}
}

// Genuine Western text has no Cyrillic shape, so the Latin verdicts stand.
func TestDetect_WesternTextKeepsLatinVerdict(t *testing.T) {
	tests := []struct {
		name string
		cm   *charmap.Charmap
		text string
	}{
		{"latin-1 french", charmap.ISO8859_1, "Le café était très agréable, à côté de l'hôtel où j'ai déjeuné hier."},
		{"macroman western", charmap.Macintosh, "Café, naïve, résumé — Zürich, Ångström, œuvre, piñata, jalapeño."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := []byte(strings.Repeat(string(charmapEncode(t, tt.cm, tt.text))+"\r\n", 8))

			if result := Detect(data); result.Charset == "windows-1251" {
				t.Errorf("Charset = windows-1251 for Western text: %+v", result)
			}
		})
	}
}

func TestCandidates_SparseCyrillicInSource(t *testing.T) {
	ranked := candidates(sparseCyrillicSource(t, charmap.Windows1251, 340))
	if len(ranked) == 0 {
		t.Fatal("no candidates ranked")
	}
	if ranked[0].Charset != "windows-1251" || !ranked[0].Supported {
		t.Errorf("top candidate = %+v, want supported windows-1251", ranked[0])
	}
	for _, candidate := range ranked {
		if candidate.Charset == "gbk" {
			t.Errorf("gbk ranked for Cyrillic source: %v", ranked)
		}
	}
}

// The scorer answers on the bytes alone, so it must name the right Cyrillic table and stay silent on every other script.
func TestCyrillicCodepage(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"cp1251 prose", charmapEncode(t, charmap.Windows1251, cyrillicFixture), "windows-1251"},
		{"cp1251 sparse in source", sparseCyrillicSource(t, charmap.Windows1251, 340), "windows-1251"},
		{"koi8-r sparse", sparseCyrillicSource(t, charmap.KOI8R, 340, russianLiterals...), "koi8-r"},
		{"ibm866 sparse", sparseCyrillicSource(t, charmap.CodePage866, 340, russianLiterals...), "ibm866"},
		{"iso-8859-5 sparse", sparseCyrillicSource(t, charmap.ISO8859_5, 340, russianLiterals...), "iso-8859-5"},
		{"maccyrillic sparse", sparseCyrillicSource(t, charmap.MacintoshCyrillic, 340, russianLiterals...), "x-mac-cyrillic"},
		{"french latin-1", charmapEncode(t, charmap.ISO8859_1, "Le café était très agréable, à côté de l'hôtel où j'ai déjeuné."), ""},
		{"spanish cp1252", charmapEncode(t, charmap.Windows1252, "MÓDULO FÍSICAMENTE ÚNICO, según la configuración señalada."), ""},
		{"macroman western", charmapEncode(t, charmap.Macintosh, "Café, naïve, résumé — Zürich, Ångström, œuvre."), ""},
		{"cp1252 punctuation", charmapEncode(t, charmap.Windows1252, "He said “hello” — it’s fine… ½ price, 25° C."), ""},
		{"arabic cp1256", charmapEncode(t, charmap.Windows1256, "مرحبا بالعالم هذا نص تجريبي للاختبار الترميز"), ""},
		{"turkish cp1254", charmapEncode(t, charmap.Windows1254, "Türkçe karakterler: ğüşiöç ĞÜŞİÖÇ İstanbul'da güzel."), ""},
		{"polish cp1250", charmapEncode(t, charmap.Windows1250, "Zażółć gęślą jaźń, mówił wesoły łoś na łące."), ""},
		{"latvian cp1257", charmapEncode(t, charmap.Windows1257, "Rīgā šodien ir ļoti skaists laiks, un mēs gribam iet ārā."), ""},
		{"gbk chinese", gbkEncode(t, "汉字编码检测测试内容字符串样例中华人民共和国国家标准"), ""},
		{"utf-8 cyrillic", []byte(cyrillicFixture), ""},
		{"plain ascii", []byte("Hello, World! This is plain ASCII."), ""},
		{"empty", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cyrillicCodepage(tt.data); got != tt.want {
				t.Errorf("cyrillicCodepage = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWordLike(t *testing.T) {
	tests := []struct {
		word string
		want bool
	}{
		{"дума", true},
		{"ДУМА", false},
		{"Дума", true},
		{"дУмА", false},
		{"ДуМа", false},
		{"на", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.word, func(t *testing.T) {
			if got := wordLike([]rune(tt.word)); got != tt.want {
				t.Errorf("wordLike(%q) = %v, want %v", tt.word, got, tt.want)
			}
		})
	}
}
