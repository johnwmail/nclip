package utils

import (
	"os"
	"testing"
)

func TestIsDebugEnabled(t *testing.T) {
	orig := os.Getenv("GIN_MODE")
	defer func() {
		if err := os.Setenv("GIN_MODE", orig); err != nil {
			t.Fatalf("failed to restore GIN_MODE: %v", err)
		}
	}()

	if err := os.Setenv("GIN_MODE", "release"); err != nil {
		t.Fatalf("failed to set GIN_MODE: %v", err)
	}
	if IsDebugEnabled() {
		t.Error("IsDebugEnabled() should be false when GIN_MODE=release")
	}

	if err := os.Setenv("GIN_MODE", "debug"); err != nil {
		t.Fatalf("failed to set GIN_MODE: %v", err)
	}
	if !IsDebugEnabled() {
		t.Error("IsDebugEnabled() should be true when GIN_MODE=debug")
	}

	if err := os.Setenv("GIN_MODE", ""); err != nil {
		t.Fatalf("failed to unset GIN_MODE: %v", err)
	}
	if !IsDebugEnabled() {
		t.Error("IsDebugEnabled() should be true when GIN_MODE is unset")
	}
}
