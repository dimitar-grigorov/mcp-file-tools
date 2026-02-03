package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any environment variables
	os.Unsetenv(EnvDefaultEncoding)
	os.Unsetenv(EnvMemoryThreshold)

	cfg := Load()

	if cfg.DefaultEncoding != DefaultEncoding {
		t.Errorf("expected default encoding %q, got %q", DefaultEncoding, cfg.DefaultEncoding)
	}

	if cfg.MemoryThreshold != DefaultMaxSize {
		t.Errorf("expected default memory threshold %d, got %d", DefaultMaxSize, cfg.MemoryThreshold)
	}
}

func TestLoad_CustomEncoding(t *testing.T) {
	os.Setenv(EnvDefaultEncoding, "utf-8")
	defer os.Unsetenv(EnvDefaultEncoding)

	cfg := Load()

	if cfg.DefaultEncoding != "utf-8" {
		t.Errorf("expected encoding utf-8, got %q", cfg.DefaultEncoding)
	}
}

func TestLoad_InvalidEncoding(t *testing.T) {
	os.Setenv(EnvDefaultEncoding, "invalid-encoding-xyz")
	defer os.Unsetenv(EnvDefaultEncoding)

	cfg := Load()

	// Should fall back to default when invalid
	if cfg.DefaultEncoding != DefaultEncoding {
		t.Errorf("expected fallback to %q for invalid encoding, got %q", DefaultEncoding, cfg.DefaultEncoding)
	}
}

func TestLoad_CustomMemoryThreshold(t *testing.T) {
	os.Setenv(EnvMemoryThreshold, "134217728") // 128MB
	defer os.Unsetenv(EnvMemoryThreshold)

	cfg := Load()

	if cfg.MemoryThreshold != 134217728 {
		t.Errorf("expected memory threshold 134217728, got %d", cfg.MemoryThreshold)
	}
}

func TestLoad_InvalidMemoryThreshold(t *testing.T) {
	os.Setenv(EnvMemoryThreshold, "not-a-number")
	defer os.Unsetenv(EnvMemoryThreshold)

	cfg := Load()

	// Should fall back to default when invalid
	if cfg.MemoryThreshold != DefaultMaxSize {
		t.Errorf("expected fallback to %d for invalid threshold, got %d", DefaultMaxSize, cfg.MemoryThreshold)
	}
}

func TestLoad_NegativeMemoryThreshold(t *testing.T) {
	os.Setenv(EnvMemoryThreshold, "-1000")
	defer os.Unsetenv(EnvMemoryThreshold)

	cfg := Load()

	// Should fall back to default when negative
	if cfg.MemoryThreshold != DefaultMaxSize {
		t.Errorf("expected fallback to %d for negative threshold, got %d", DefaultMaxSize, cfg.MemoryThreshold)
	}
}
