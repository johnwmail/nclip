package config

import (
	"flag"
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Test default values
	if cfg.BaseURL != "http://localhost:8080/" {
		t.Errorf("Expected default BaseURL 'http://localhost:8080/', got %s", cfg.BaseURL)
	}

	if cfg.TCPPort != 8099 {
		t.Errorf("Expected default TCP port 8099, got %d", cfg.TCPPort)
	}

	if cfg.HTTPPort != 8080 {
		t.Errorf("Expected default HTTP port 8080, got %d", cfg.HTTPPort)
	}

	if cfg.StorageType != "filesystem" {
		t.Errorf("Expected default storage type 'filesystem', got %s", cfg.StorageType)
	}

	if cfg.SlugLength != 5 {
		t.Errorf("Expected default slug length 5, got %d", cfg.SlugLength)
	}

	if cfg.BufferSize != 1024*1024 {
		t.Errorf("Expected default buffer size 1048576, got %d", cfg.BufferSize)
	}

	if cfg.ExpireDays != 1 {
		t.Errorf("Expected default expire days 1, got %d", cfg.ExpireDays)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("Expected default log level 'info', got %s", cfg.LogLevel)
	}

	if !cfg.EnableWebUI {
		t.Error("Expected EnableWebUI to be true by default")
	}

	if !cfg.EnableMetrics {
		t.Error("Expected EnableMetrics to be true by default")
	}
}

func TestLoadFromFlags(t *testing.T) {
	// Save original command line args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Reset flag package for testing
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Set test arguments
	os.Args = []string{
		"program",
		"-url", "https://example.com/paste/",
		"-tcp-port", "9000",
		"-http-port", "8888",
		"-storage-type", "mongodb",
		"-output-dir", "/tmp/test",
		"-slug-length", "12",
		"-expire-days", "7",
		"-log-level", "debug",
		"-mongodb-uri", "mongodb://localhost:27017",
		"-mongodb-database", "testdb",
		"-dynamodb-table", "test-table",
		"-rate-limit", "10/minute",
	}

	cfg, err := LoadFromFlags()
	if err != nil {
		t.Fatalf("LoadFromFlags failed: %v", err)
	}

	// Test parsed values
	if cfg.BaseURL != "https://example.com/paste/" {
		t.Errorf("Expected BaseURL 'https://example.com/paste/', got %s", cfg.BaseURL)
	}

	if cfg.TCPPort != 9000 {
		t.Errorf("Expected TCP port 9000, got %d", cfg.TCPPort)
	}

	if cfg.HTTPPort != 8888 {
		t.Errorf("Expected HTTP port 8888, got %d", cfg.HTTPPort)
	}

	if cfg.StorageType != "mongodb" {
		t.Errorf("Expected storage type 'mongodb', got %s", cfg.StorageType)
	}

	if cfg.OutputDir != "/tmp/test" {
		t.Errorf("Expected output dir '/tmp/test', got %s", cfg.OutputDir)
	}

	if cfg.SlugLength != 12 {
		t.Errorf("Expected slug length 12, got %d", cfg.SlugLength)
	}

	if cfg.ExpireDays != 7 {
		t.Errorf("Expected expire days 7, got %d", cfg.ExpireDays)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("Expected log level 'debug', got %s", cfg.LogLevel)
	}

	if cfg.MongoDBURI != "mongodb://localhost:27017" {
		t.Errorf("Expected MongoDB URI 'mongodb://localhost:27017', got %s", cfg.MongoDBURI)
	}

	if cfg.MongoDBDatabase != "testdb" {
		t.Errorf("Expected MongoDB database 'testdb', got %s", cfg.MongoDBDatabase)
	}

	if cfg.DynamoDBTable != "test-table" {
		t.Errorf("Expected DynamoDB table 'test-table', got %s", cfg.DynamoDBTable)
	}

	if cfg.RateLimit != "10/minute" {
		t.Errorf("Expected rate limit '10/minute', got %s", cfg.RateLimit)
	}

	if !cfg.EnableWebUI {
		t.Error("Expected EnableWebUI to be true by default")
	}

	if !cfg.EnableMetrics {
		t.Error("Expected EnableMetrics to be true by default")
	}
}

func TestGetExpiration(t *testing.T) {
	testCases := []struct {
		name        string
		expireDays  int
		expectNever bool
	}{
		{
			name:        "never expire",
			expireDays:  0,
			expectNever: true,
		},
		{
			name:        "expire in 1 day",
			expireDays:  1,
			expectNever: false,
		},
		{
			name:        "expire in 30 days",
			expireDays:  30,
			expectNever: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.ExpireDays = tc.expireDays

			duration := cfg.GetExpiration()

			if tc.expectNever {
				if duration != 0 {
					t.Errorf("Expected 0 duration for never expire, got %v", duration)
				}
			} else {
				expected := time.Duration(tc.expireDays) * 24 * time.Hour
				if duration != expected {
					t.Errorf("Expected duration %v, got %v", expected, duration)
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		name        string
		modifier    func(*Config)
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			modifier: func(cfg *Config) {
				// Use default config, should be valid
			},
			expectError: false,
		},
		{
			name: "invalid tcp port - negative",
			modifier: func(cfg *Config) {
				cfg.TCPPort = -1
			},
			expectError: true,
			errorMsg:    "invalid TCP port: -1",
		},
		{
			name: "invalid tcp port - too high",
			modifier: func(cfg *Config) {
				cfg.TCPPort = 70000
			},
			expectError: true,
			errorMsg:    "invalid TCP port: 70000",
		},
		{
			name: "invalid http port - zero",
			modifier: func(cfg *Config) {
				cfg.HTTPPort = 0
			},
			expectError: true,
			errorMsg:    "invalid HTTP port: 0",
		},
		{
			name: "invalid storage type",
			modifier: func(cfg *Config) {
				cfg.StorageType = "invalid"
			},
			expectError: true,
			errorMsg:    "invalid storage type: invalid (valid: filesystem, mongodb, dynamodb)",
		},
		{
			name: "invalid slug length - zero",
			modifier: func(cfg *Config) {
				cfg.SlugLength = 0
			},
			expectError: true,
			errorMsg:    "slug length must be between 1 and 32: 0",
		},
		{
			name: "invalid slug length - too high",
			modifier: func(cfg *Config) {
				cfg.SlugLength = 100
			},
			expectError: true,
			errorMsg:    "slug length must be between 1 and 32: 100",
		},
		{
			name: "invalid expire days - negative",
			modifier: func(cfg *Config) {
				cfg.ExpireDays = -1
			},
			expectError: true,
			errorMsg:    "expire days cannot be negative: -1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tc.modifier(cfg)

			err := cfg.Validate()

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if err.Error() != tc.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tc.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestExpiration(t *testing.T) {
	testCases := []struct {
		name        string
		expireDays  int
		expectNever bool
	}{
		{
			name:        "never expire",
			expireDays:  0,
			expectNever: true,
		},
		{
			name:        "expire in 1 day",
			expireDays:  1,
			expectNever: false,
		},
		{
			name:        "expire in 30 days",
			expireDays:  30,
			expectNever: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.ExpireDays = tc.expireDays

			duration := cfg.GetExpiration()

			if tc.expectNever {
				if duration != 0 {
					t.Errorf("Expected 0 duration for never expire, got %v", duration)
				}
			} else {
				expected := time.Duration(tc.expireDays) * 24 * time.Hour
				if duration != expected {
					t.Errorf("Expected duration %v, got %v", expected, duration)
				}
			}
		})
	}
}
