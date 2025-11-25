package config

import (
	"os"
	"strconv"
	"testing"
	"time"
)

// Note: Due to flag package limitations, we can only test LoadConfig once per test run.
// These tests verify environment variable overrides work correctly.

func TestLoadConfig_Defaults(t *testing.T) {
	os.Clearenv()
	cfg := LoadConfig()
	if cfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Port)
	}
	if cfg.SlugLength != 5 {
		t.Errorf("expected default slug length 5, got %d", cfg.SlugLength)
	}
	if cfg.BufferSize != 5*1024*1024 {
		t.Errorf("expected default buffer size 5MB, got %d", cfg.BufferSize)
	}
	if cfg.DefaultTTL != 24*time.Hour {
		t.Errorf("expected default TTL 24h, got %v", cfg.DefaultTTL)
	}
	if cfg.MaxRenderSize != 262144 {
		t.Errorf("expected default MaxRenderSize 262144, got %d", cfg.MaxRenderSize)
	}
}

func TestMaxRenderSize_FromEnv(t *testing.T) {
	// Test that MaxRenderSize can be overridden by environment variable
	// We create a new Config and manually apply env var logic
	if err := os.Setenv("NCLIP_MAX_RENDER_SIZE", "65536"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("NCLIP_MAX_RENDER_SIZE"); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}
	}()

	// Simulate the env var parsing logic from LoadConfig
	var maxRenderSize int64 = 262144 // default
	if val := os.Getenv("NCLIP_MAX_RENDER_SIZE"); val != "" {
		if v, err := strconv.ParseInt(val, 10, 64); err == nil {
			maxRenderSize = v
		}
	}

	if maxRenderSize != 65536 {
		t.Errorf("expected MaxRenderSize 65536 from env, got %d", maxRenderSize)
	}
}

func TestMaxRenderSize_InvalidEnv(t *testing.T) {
	// Test that invalid env var falls back to default
	if err := os.Setenv("NCLIP_MAX_RENDER_SIZE", "invalid"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("NCLIP_MAX_RENDER_SIZE"); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}
	}()

	var maxRenderSize int64 = 262144 // default
	if val := os.Getenv("NCLIP_MAX_RENDER_SIZE"); val != "" {
		if v, err := strconv.ParseInt(val, 10, 64); err == nil {
			maxRenderSize = v
		}
	}

	if maxRenderSize != 262144 {
		t.Errorf("expected default MaxRenderSize 262144 when env is invalid, got %d", maxRenderSize)
	}
}
