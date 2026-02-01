package encoding

import (
	"testing"

	"golang.org/x/text/encoding/charmap"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name     string
		wantOk   bool
		wantNil  bool // true if encoding should be nil (UTF-8)
	}{
		{"utf-8", true, true},
		{"UTF-8", true, true},
		{"utf8", true, true},
		{"windows-1251", true, false},
		{"cp1251", true, false},
		{"CP1251", true, false},
		{"koi8-r", true, false},
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

func TestGetInfo(t *testing.T) {
	info, ok := GetInfo("cp1251")
	if !ok {
		t.Fatal("GetInfo(cp1251) failed")
	}
	if info.DisplayName != "Windows-1251" {
		t.Errorf("DisplayName = %q, want Windows-1251", info.DisplayName)
	}
	if info.Encoding != charmap.Windows1251 {
		t.Error("Encoding mismatch for cp1251")
	}

	_, ok = GetInfo("nonexistent")
	if ok {
		t.Error("GetInfo(nonexistent) should return false")
	}
}

func TestGetCanonicalName(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"cp1251", "windows-1251", true},
		{"CP1251", "windows-1251", true},
		{"windows-1251", "windows-1251", true},
		{"utf8", "utf-8", true},
		{"invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := GetCanonicalName(tt.input)
			if ok != tt.ok {
				t.Errorf("GetCanonicalName(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("GetCanonicalName(%q) = %q, want %q", tt.input, got, tt.want)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUTF8(tt.name); got != tt.want {
				t.Errorf("IsUTF8(%q) = %v, want %v", tt.name, got, tt.want)
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

	// Verify we have the expected number of encodings (20)
	if len(items) != 20 {
		t.Errorf("ListEncodings() returned %d items, want 20", len(items))
	}
}
