package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the nclip service
type Config struct {
	Port       int           `json:"port"`
	URL        string        `json:"url"`
	SlugLength int           `json:"slug_length"`
	BufferSize int64         `json:"buffer_size"`
	DefaultTTL time.Duration `json:"default_ttl"`
	S3Bucket   string        `json:"s3_bucket"`
	S3Prefix   string        `json:"s3_prefix"`
	// UploadAuth enables API key authentication on upload endpoints
	UploadAuth bool `json:"upload_auth"`
	// APIKeys is a comma-separated list of valid API keys
	APIKeys       string `json:"api_keys"`
	Version       string `json:"version"`
	BuildTime     string `json:"build_time"`
	CommitHash    string `json:"commit_hash"`
	MaxRenderSize int64  `json:"max_render_size"`
}

// LoadConfig loads configuration from environment variables and CLI flags
func LoadConfig() *Config {
	config := &Config{
		Port:          8080,
		URL:           "",
		SlugLength:    5,
		BufferSize:    5 * 1024 * 1024, // 5MB
		DefaultTTL:    24 * time.Hour,
		S3Bucket:      "",
		S3Prefix:      "",
		MaxRenderSize: 262144, // 256 KiB
	}

	// Parse CLI flags
	flag.IntVar(&config.Port, "port", config.Port, "Port to listen on")
	flag.StringVar(&config.URL, "url", config.URL, "Base URL for paste links")
	flag.IntVar(&config.SlugLength, "slug-length", config.SlugLength, "Length of generated slugs")
	flag.Int64Var(&config.BufferSize, "buffer-size", config.BufferSize, "Maximum upload size in bytes")
	flag.Int64Var(&config.MaxRenderSize, "max-render-size", config.MaxRenderSize, "Maximum size (bytes) to render inline in the HTML view")
	flag.DurationVar(&config.DefaultTTL, "ttl", config.DefaultTTL, "Default paste expiration time")
	flag.StringVar(&config.S3Bucket, "s3-bucket", config.S3Bucket, "S3 bucket for Lambda mode")
	flag.StringVar(&config.S3Prefix, "s3-prefix", config.S3Prefix, "S3 key prefix for Lambda mode")
	flag.BoolVar(&config.UploadAuth, "upload-auth", config.UploadAuth, "Require API key for upload endpoints")
	flag.StringVar(&config.APIKeys, "api-keys", config.APIKeys, "Comma-separated API keys for upload authentication")
	flag.Parse()

	// Override with environment variables if present
	setIntEnv := func(env string, dest *int) {
		if val := os.Getenv(env); val != "" {
			if v, err := strconv.Atoi(val); err == nil {
				*dest = v
			}
		}
	}
	setInt64Env := func(env string, dest *int64) {
		if val := os.Getenv(env); val != "" {
			if v, err := strconv.ParseInt(val, 10, 64); err == nil {
				*dest = v
			}
		}
	}
	setBoolEnv := func(env string, dest *bool) {
		if val := os.Getenv(env); val != "" {
			if v, err := strconv.ParseBool(val); err == nil {
				*dest = v
			}
		}
	}
	setStringEnv := func(env string, dest *string) {
		if val := os.Getenv(env); val != "" {
			*dest = val
		}
	}

	setIntEnv("NCLIP_PORT", &config.Port)
	setStringEnv("NCLIP_URL", &config.URL)
	setIntEnv("NCLIP_SLUG_LENGTH", &config.SlugLength)
	setInt64Env("NCLIP_BUFFER_SIZE", &config.BufferSize)
	if val := os.Getenv("NCLIP_TTL"); val != "" {
		if ttl, err := time.ParseDuration(val); err == nil {
			config.DefaultTTL = ttl
		}
	}
	setStringEnv("NCLIP_S3_BUCKET", &config.S3Bucket)
	setStringEnv("NCLIP_S3_PREFIX", &config.S3Prefix)
	setBoolEnv("NCLIP_UPLOAD_AUTH", &config.UploadAuth)
	setStringEnv("NCLIP_API_KEYS", &config.APIKeys)
	// NCLIP_MAX_RENDER_SIZE configures MaxRenderSize; preview length equals MaxRenderSize.
	setInt64Env("NCLIP_MAX_RENDER_SIZE", &config.MaxRenderSize)

	return config
}
