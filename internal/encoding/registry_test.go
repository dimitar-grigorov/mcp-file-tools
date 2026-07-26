// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package encoding

import (
	"testing"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name    string
		wantOk  bool
		wantNil bool // true if encoding should be nil (UTF-8)
	}{
		{"utf-8", true, true},
		{"UTF-8", true, true},
		{"utf8", true, true},
		{"windows-1251", true, false},
		{"cp1251", true, false},
		{"CP1251", true, false},
		{"koi8-r", true, false},
		{"utf-16-le", true, false},
		{"utf-16-be", true, false},
		{"utf16le", true, false},
		{"utf16be", true, false},
		{"gbk", true, false},
		{"gb2312", true, false},
		{"gb-2312", true, false},
		{"cp936", true, false},
		{"gb18030", true, false},
		{"invalid", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, ok := Get(tt.name)
			if ok != tt.wantOk {
				t.Errorf("Get(%q) ok = %v, want %v", tt.name, ok, tt.wantOk)
			}
			if tt.wantOk && tt.wantNil && enc != nil {
				t.Errorf("Get(%q) = %v, want nil (UTF-8)", tt.name, enc)
			}
			if tt.wantOk && !tt.wantNil && enc == nil {
				t.Errorf("Get(%q) = nil, want non-nil encoding", tt.name)
			}
		})
	}
}

func TestIsUTF8(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"utf-8", true},
		{"UTF-8", true},
		{"utf8", true},
		{"ascii", true},
		{"cp1251", false},
		{"windows-1251", false},
		{"gbk", false},
		{"gb18030", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUTF8(tt.name); got != tt.want {
				t.Errorf("IsUTF8(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestCanonical(t *testing.T) {
	tests := []struct {
		name   string
		want   string
		wantOk bool
	}{
		{"utf-16-le", "utf-16-le", true},
		{"utf16le", "utf-16-le", true},
		{"utf-16LE", "utf-16-le", true},
		{"utf16be", "utf-16-be", true},
		{"cp1251", "windows-1251", true},
		{"ascii", "utf-8", true},
		{"invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Canonical(tt.name)
			if ok != tt.wantOk {
				t.Errorf("Canonical(%q) ok = %v, want %v", tt.name, ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("Canonical(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestListEncodings(t *testing.T) {
	items := ListEncodings()
	if len(items) == 0 {
		t.Fatal("ListEncodings() returned empty list")
	}

	// Check that items are sorted by DisplayName
	for i := 1; i < len(items); i++ {
		if items[i-1].DisplayName > items[i].DisplayName {
			t.Errorf("ListEncodings not sorted: %q > %q", items[i-1].DisplayName, items[i].DisplayName)
		}
	}

	// Verify we have the expected number of encodings (24)
	if len(items) != 24 {
		t.Errorf("ListEncodings() returned %d items, want 24", len(items))
	}
}
