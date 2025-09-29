package utils

import (
	"os"
	"testing"
)

func TestIsDebugEnabled(t *testing.T) {
	orig := os.Getenv("GIN_MODE")
	defer os.Setenv("GIN_MODE", orig)

	os.Setenv("GIN_MODE", "release")
	if IsDebugEnabled() {
		t.Error("IsDebugEnabled() should be false when GIN_MODE=release")
	}

	os.Setenv("GIN_MODE", "debug")
	if !IsDebugEnabled() {
		t.Error("IsDebugEnabled() should be true when GIN_MODE=debug")
	}

	os.Setenv("GIN_MODE", "")
	if !IsDebugEnabled() {
		t.Error("IsDebugEnabled() should be true when GIN_MODE is unset")
	}
}
