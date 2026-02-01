package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any environment variables
	os.Unsetenv(EnvDefaultEncoding)
	os.Unsetenv(EnvMaxFileSize)

	cfg := Load()

	if cfg.DefaultEncoding != DefaultEncoding {
		t.Errorf("expected default encoding %q, got %q", DefaultEncoding, cfg.DefaultEncoding)
	}

	if cfg.MaxFileSize != DefaultMaxSize {
		t.Errorf("expected default max size %d, got %d", DefaultMaxSize, cfg.MaxFileSize)
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

func TestLoad_CustomMaxFileSize(t *testing.T) {
	os.Setenv(EnvMaxFileSize, "5242880")
	defer os.Unsetenv(EnvMaxFileSize)

	cfg := Load()

	if cfg.MaxFileSize != 5242880 {
		t.Errorf("expected max size 5242880, got %d", cfg.MaxFileSize)
	}
}

func TestLoad_InvalidMaxFileSize(t *testing.T) {
	os.Setenv(EnvMaxFileSize, "not-a-number")
	defer os.Unsetenv(EnvMaxFileSize)

	cfg := Load()

	// Should fall back to default when invalid
	if cfg.MaxFileSize != DefaultMaxSize {
		t.Errorf("expected fallback to %d for invalid size, got %d", DefaultMaxSize, cfg.MaxFileSize)
	}
}

func TestLoad_NegativeMaxFileSize(t *testing.T) {
	os.Setenv(EnvMaxFileSize, "-1000")
	defer os.Unsetenv(EnvMaxFileSize)

	cfg := Load()

	// Should fall back to default when negative
	if cfg.MaxFileSize != DefaultMaxSize {
		t.Errorf("expected fallback to %d for negative size, got %d", DefaultMaxSize, cfg.MaxFileSize)
	}
}
