// SPDX-License-Identifier: GPL-3.0-only
// Copyright (C) 2026 Dimitar Grigorov

package config

import (
	"slices"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv(EnvDefaultEncoding, "")
	t.Setenv(EnvMemoryThreshold, "")

	cfg := Load()

	if cfg.DefaultEncoding != DefaultEncoding {
		t.Errorf("expected default encoding %q, got %q", DefaultEncoding, cfg.DefaultEncoding)
	}

	if cfg.MemoryThreshold != DefaultMaxSize {
		t.Errorf("expected default memory threshold %d, got %d", DefaultMaxSize, cfg.MemoryThreshold)
	}

	if cfg.DefaultEncodingFromEnv {
		t.Error("expected DefaultEncodingFromEnv false when the env var is unset")
	}
}

// cp1251 doubles as the pre-2.0.0 default, so this also covers the migration path.
func TestLoad_CustomEncoding(t *testing.T) {
	t.Setenv(EnvDefaultEncoding, "cp1251")

	cfg := Load()

	if cfg.DefaultEncoding != "cp1251" {
		t.Errorf("expected encoding cp1251, got %q", cfg.DefaultEncoding)
	}

	if !cfg.DefaultEncodingFromEnv {
		t.Error("expected DefaultEncodingFromEnv true when the env var supplied the encoding")
	}
}

func TestLoad_InvalidEncoding(t *testing.T) {
	t.Setenv(EnvDefaultEncoding, "invalid-encoding-xyz")

	cfg := Load()

	// Should fall back to default when invalid
	if cfg.DefaultEncoding != DefaultEncoding {
		t.Errorf("expected fallback to %q for invalid encoding, got %q", DefaultEncoding, cfg.DefaultEncoding)
	}

	// A rejected value did not set the default, so it does not count as a choice.
	if cfg.DefaultEncodingFromEnv {
		t.Error("expected DefaultEncodingFromEnv false when the env value was rejected")
	}
}

func TestLoad_CustomMemoryThreshold(t *testing.T) {
	t.Setenv(EnvMemoryThreshold, "134217728") // 128MB

	cfg := Load()

	if cfg.MemoryThreshold != 134217728 {
		t.Errorf("expected memory threshold 134217728, got %d", cfg.MemoryThreshold)
	}
}

func TestLoad_InvalidMemoryThreshold(t *testing.T) {
	t.Setenv(EnvMemoryThreshold, "not-a-number")

	cfg := Load()

	// Should fall back to default when invalid
	if cfg.MemoryThreshold != DefaultMaxSize {
		t.Errorf("expected fallback to %d for invalid threshold, got %d", DefaultMaxSize, cfg.MemoryThreshold)
	}
}

func TestLoad_NegativeMemoryThreshold(t *testing.T) {
	t.Setenv(EnvMemoryThreshold, "-1000")

	cfg := Load()

	// Should fall back to default when negative
	if cfg.MemoryThreshold != DefaultMaxSize {
		t.Errorf("expected fallback to %d for negative threshold, got %d", DefaultMaxSize, cfg.MemoryThreshold)
	}
}

func TestLoad_DetectionCandidates(t *testing.T) {
	t.Setenv(EnvDetectionCandidates, " utf8 , windows-1252 ,, klingon-1 ")

	got := Load().DetectionCandidates
	want := []string{"utf-8", "windows-1252"}
	if !slices.Equal(got, want) {
		t.Fatalf("candidates = %v, want %v (canonicalized, blanks and unknowns dropped)", got, want)
	}
}

func TestLoad_DetectionCandidatesUnset(t *testing.T) {
	t.Setenv(EnvDetectionCandidates, "")

	if got := Load().DetectionCandidates; got != nil {
		t.Fatalf("candidates = %v, want nil (unrestricted)", got)
	}
}
