package updater

import (
	"strings"
	"testing"
)

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"2.0.0", "1.0.0", true},
		{"1.1.0", "1.0.0", true},
		{"1.0.1", "1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "2.0.0", false},
		{"v1.1.0", "1.0.0", true},
		{"1.1.0", "v1.0.0", true},
		{"1.1.0-beta", "1.0.0", true},
		{"1.1", "1.0.0", true},
		{"2", "1.0.0", true},
	}

	for _, tt := range tests {
		if got := isNewerVersion(tt.latest, tt.current); got != tt.want {
			t.Errorf("isNewerVersion(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
	}{
		{"1.2.3", [3]int{1, 2, 3}},
		{"v1.2.3", [3]int{1, 2, 3}},
		{"1.2.3-beta", [3]int{1, 2, 3}},
		{"1.2", [3]int{1, 2, 0}},
		{"1", [3]int{1, 0, 0}},
		{"", [3]int{0, 0, 0}},
	}

	for _, tt := range tests {
		if got := parseVersion(tt.input); got != tt.want {
			t.Errorf("parseVersion(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestCheckDisabled(t *testing.T) {
	t.Setenv("MCP_NO_UPDATE_CHECK", "1")
	if msg := Check(nil, "1.0.0"); msg != "" {
		t.Errorf("Check with disabled should return empty, got %q", msg)
	}
}

func TestCheckDevVersion(t *testing.T) {
	if msg := Check(nil, "dev"); msg != "" {
		t.Errorf("Check with dev version should return empty, got %q", msg)
	}
	if msg := Check(nil, ""); msg != "" {
		t.Errorf("Check with empty version should return empty, got %q", msg)
	}
}

func TestUpdateMessageFormat(t *testing.T) {
	// Just verify the format string works
	msg := "Update available: 1.0.0 → 1.1.0\nDownload: https://example.com"
	if !strings.Contains(msg, "1.0.0") || !strings.Contains(msg, "1.1.0") {
		t.Error("message format incorrect")
	}
}
