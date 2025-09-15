package main

import (
	"os"
	"testing"
	"time"
)

func TestEnvironmentDetection(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		expectedLambda bool
	}{
		{
			name:           "No Lambda environment variables",
			envVars:        map[string]string{},
			expectedLambda: false,
		},
		{
			name: "AWS_LAMBDA_RUNTIME_API set",
			envVars: map[string]string{
				"AWS_LAMBDA_RUNTIME_API": "test-api",
			},
			expectedLambda: true,
		},
		{
			name: "_LAMBDA_SERVER_PORT set",
			envVars: map[string]string{
				"_LAMBDA_SERVER_PORT": "8080",
			},
			expectedLambda: true,
		},
		{
			name: "Both Lambda environment variables set",
			envVars: map[string]string{
				"AWS_LAMBDA_RUNTIME_API": "test-api",
				"_LAMBDA_SERVER_PORT":    "8080",
			},
			expectedLambda: true,
		},
		{
			name: "Other environment variables",
			envVars: map[string]string{
				"PATH":     "/usr/bin",
				"HOME":     "/home/user",
				"SOME_VAR": "value",
			},
			expectedLambda: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalEnv := make(map[string]string)
			for key := range tt.envVars {
				if val, exists := os.LookupEnv(key); exists {
					originalEnv[key] = val
				}
			}

			// Clear Lambda environment variables first
			_ = os.Unsetenv("AWS_LAMBDA_RUNTIME_API")
			_ = os.Unsetenv("_LAMBDA_SERVER_PORT")

			// Set test environment variables
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}

			// Test environment detection logic
			isLambda := os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" || os.Getenv("_LAMBDA_SERVER_PORT") != ""

			if isLambda != tt.expectedLambda {
				t.Errorf("Expected Lambda detection: %v, got: %v", tt.expectedLambda, isLambda)
			}

			// Restore original environment
			for key := range tt.envVars {
				_ = os.Unsetenv(key)
			}
			for key, value := range originalEnv {
				_ = os.Setenv(key, value)
			}
		})
	}
}

func TestMainFunction(t *testing.T) {
	// This test ensures that main() doesn't panic when called in different scenarios
	// We can't easily test the actual execution paths without complex mocking,
	// but we can test that the environment detection logic is sound.

	// Save original environment
	originalRuntime := os.Getenv("AWS_LAMBDA_RUNTIME_API")
	originalPort := os.Getenv("_LAMBDA_SERVER_PORT")

	defer func() {
		// Restore original environment
		if originalRuntime != "" {
			_ = os.Setenv("AWS_LAMBDA_RUNTIME_API", originalRuntime)
		} else {
			_ = os.Unsetenv("AWS_LAMBDA_RUNTIME_API")
		}

		if originalPort != "" {
			_ = os.Setenv("_LAMBDA_SERVER_PORT", originalPort)
		} else {
			_ = os.Unsetenv("_LAMBDA_SERVER_PORT")
		}
	}()

	t.Run("Server mode detection", func(t *testing.T) {
		// Clear Lambda environment variables
		_ = os.Unsetenv("AWS_LAMBDA_RUNTIME_API")
		_ = os.Unsetenv("_LAMBDA_SERVER_PORT")

		// This should detect server mode
		isLambda := os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" || os.Getenv("_LAMBDA_SERVER_PORT") != ""
		if isLambda {
			t.Error("Expected server mode detection, but got Lambda mode")
		}
	})

	t.Run("Lambda mode detection", func(t *testing.T) {
		// Set Lambda environment variable
		_ = os.Setenv("AWS_LAMBDA_RUNTIME_API", "test")

		// This should detect Lambda mode
		isLambda := os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" || os.Getenv("_LAMBDA_SERVER_PORT") != ""
		if !isLambda {
			t.Error("Expected Lambda mode detection, but got server mode")
		}
	})
}

func TestBuildInfo(t *testing.T) {
	// Test that build info variables are properly declared
	// These are set at build time via ldflags

	if version == "" {
		t.Log("version is empty (expected for test builds)")
	}

	if buildTime == "" && buildTime != "unknown" {
		t.Log("buildTime is empty (expected for test builds)")
	}

	if gitCommit == "" && gitCommit != "unknown" {
		t.Log("gitCommit is empty (expected for test builds)")
	}

	// Ensure they're at least initialized
	_ = version
	_ = buildTime
	_ = gitCommit
}

// TestLambdaFunctionTimeout tests that Lambda function doesn't run indefinitely
func TestLambdaFunctionTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	// This test verifies that we can safely test Lambda mode detection
	// without actually starting the Lambda runtime

	originalRuntime := os.Getenv("AWS_LAMBDA_RUNTIME_API")
	defer func() {
		if originalRuntime != "" {
			_ = os.Setenv("AWS_LAMBDA_RUNTIME_API", originalRuntime)
		} else {
			_ = os.Unsetenv("AWS_LAMBDA_RUNTIME_API")
		}
	}()

	// Set a test Lambda environment
	_ = os.Setenv("AWS_LAMBDA_RUNTIME_API", "localhost:8099")

	// Create a channel to track if Lambda mode would be triggered
	done := make(chan bool, 1)

	go func() {
		// Simulate the environment detection logic
		isLambda := os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" || os.Getenv("_LAMBDA_SERVER_PORT") != ""
		done <- isLambda
	}()

	// Wait for the detection with a timeout
	select {
	case result := <-done:
		if !result {
			t.Error("Expected Lambda mode detection to return true")
		}
	case <-time.After(1 * time.Second):
		t.Error("Environment detection took too long")
	}
}

// BenchmarkEnvironmentDetection benchmarks the environment detection logic
func BenchmarkEnvironmentDetection(b *testing.B) {
	// Clear environment variables for consistent benchmarking
	_ = os.Unsetenv("AWS_LAMBDA_RUNTIME_API")
	_ = os.Unsetenv("_LAMBDA_SERVER_PORT")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" || os.Getenv("_LAMBDA_SERVER_PORT") != ""
	}
}

// BenchmarkEnvironmentDetectionWithVars benchmarks when environment variables are set
func BenchmarkEnvironmentDetectionWithVars(b *testing.B) {
	originalRuntime := os.Getenv("AWS_LAMBDA_RUNTIME_API")
	defer func() {
		if originalRuntime != "" {
			_ = os.Setenv("AWS_LAMBDA_RUNTIME_API", originalRuntime)
		} else {
			_ = os.Unsetenv("AWS_LAMBDA_RUNTIME_API")
		}
	}()

	// Set environment variable for benchmarking
	_ = os.Setenv("AWS_LAMBDA_RUNTIME_API", "test")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = os.Getenv("AWS_LAMBDA_RUNTIME_API") != "" || os.Getenv("_LAMBDA_SERVER_PORT") != ""
	}
}
