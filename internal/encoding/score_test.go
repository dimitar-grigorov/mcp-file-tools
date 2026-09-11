// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"
)

// frenchCP1252 has enough lowercase accents to judge, which is the point: they sit inside words.
const frenchCP1252 = "Le caf\xe9 \xe9tait tr\xe8s agr\xe9able, \xe0 c\xf4t\xe9 de l'h\xf4tel o\xf9 j'ai d\xe9jeun\xe9 hier.\r\n" +
	"La r\xe9union a \xe9t\xe9 report\xe9e \xe0 la semaine prochaine, apr\xe8s les f\xeates.\r\n"

// The right table spells words out of the high bytes; every wrong one scatters them between the ASCII.
func TestScoreCharset_OnlyTheRightTableSpellsWords(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		right    string
		wrong    []string
		minShare float64
	}{
		{"cyrillic cp1251", charmapEncode(t, charmap.Windows1251, cyrillicFixture), "windows-1251",
			[]string{"windows-1252", "iso-8859-1", "koi8-r", "ibm866"}, 0.8},
		{"french cp1252", []byte(frenchCP1252), "windows-1252",
			[]string{"windows-1251", "koi8-r", "ibm866", "iso-8859-5"}, 0.8},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			right := scoreCharset(tc.right, tc.data)
			if !right.plausible() || right.share() < tc.minShare {
				t.Fatalf("%s: share %.3f over %d high bytes, want at least %.2f", tc.right, right.share(), right.high, tc.minShare)
			}
			for _, charset := range tc.wrong {
				if wrong := scoreCharset(charset, tc.data); wrong.plausible() {
					t.Errorf("%s reads as %s at share %.3f, want implausible", tc.name, charset, wrong.share())
				}
			}
		})
	}
}

// A byte outside the table is proof the bytes are not it, and cp1251 has exactly one: 0x98.
func TestScoreCharset_UndefinedByteIsUnreadable(t *testing.T) {
	data := append(charmapEncode(t, charmap.Windows1251, cyrillicFixture), 0x98)
	if p := scoreCharset("windows-1251", data); p.readable {
		t.Errorf("readable with an undefined byte: %+v", p)
	}
}

// Delphi source is not Chinese, however many of its accents happen to form valid GBK pairs.
func TestScoreCharset_WesternSourceIsNotChinese(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"spanish cp1252", []byte(spanishCP1252)},
		{"french cp1252", []byte(frenchCP1252)},
		{"cyrillic cp1251", charmapEncode(t, charmap.Windows1251, strings.Repeat(cyrillicFixture, 4))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if scoreCharset("gbk", tc.data).plausible() {
				t.Error("reads as plausible GBK")
			}
		})
	}
}

// Pure ASCII reads the same under every table, so it settles nothing and must never carry a score.
func TestScoreCharset_ASCIIProvesNothing(t *testing.T) {
	data := []byte("procedure DoSomething(const AValue: Integer);\r\n")
	for _, charset := range []string{"windows-1251", "windows-1252", "koi8-r"} {
		p := scoreCharset(charset, data)
		if !p.readable || p.high != 0 || p.plausible() {
			t.Errorf("%s: %+v, want readable with no evidence either way", charset, p)
		}
	}
}

// cp1251 and MacCyrillic read the same words out of Bulgarian source, so the margin has to keep the answer off the noise.
func TestBestCyrillicTable_NearTieKeepsCP1251(t *testing.T) {
	data := charmapEncode(t, charmap.Windows1251, cyrillicFixture)

	cp, mac := scoreCharset("windows-1251", data), scoreCharset("x-mac-cyrillic", data)
	if diff := mac.share() - cp.share(); diff > tableMargin {
		t.Skipf("this sample separates the two tables by %.3f, so it no longer tests the margin", diff)
	}
	if best, _ := bestCyrillicTable(data); best != "windows-1251" {
		t.Errorf("best = %q at cp1251 %.3f vs mac %.3f, want windows-1251", best, cp.share(), mac.share())
	}
}

// A table that genuinely reads the bytes better still has to win, or the margin would just freeze the list order.
func TestBestCyrillicTable_ClearWinnerTakesIt(t *testing.T) {
	for _, tc := range []struct {
		cm   *charmap.Charmap
		want string
	}{
		{charmap.KOI8R, "koi8-r"},
		{charmap.CodePage866, "ibm866"},
		{charmap.MacintoshCyrillic, "x-mac-cyrillic"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			if best, _ := bestCyrillicTable(charmapEncode(t, tc.cm, cyrillicFixture)); best != tc.want {
				t.Errorf("best = %q, want %s", best, tc.want)
			}
		})
	}
}
