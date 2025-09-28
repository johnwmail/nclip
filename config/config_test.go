package config

import (
	"os"
	"testing"
	"time"
)

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
}
