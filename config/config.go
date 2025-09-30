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
	Version    string        `json:"version"`
	BuildTime  string        `json:"build_time"`
	CommitHash string        `json:"commit_hash"`
}

// LoadConfig loads configuration from environment variables and CLI flags
func LoadConfig() *Config {
	config := &Config{
		Port:       8080,
		URL:        "",
		SlugLength: 5,
		BufferSize: 5 * 1024 * 1024, // 5MB
		DefaultTTL: 24 * time.Hour,
		S3Bucket:   "",
		S3Prefix:   "",
	}

	// Parse CLI flags
	flag.IntVar(&config.Port, "port", config.Port, "Port to listen on")
	flag.StringVar(&config.URL, "url", config.URL, "Base URL for paste links")
	flag.IntVar(&config.SlugLength, "slug-length", config.SlugLength, "Length of generated slugs")
	flag.Int64Var(&config.BufferSize, "buffer-size", config.BufferSize, "Maximum upload size in bytes")
	flag.DurationVar(&config.DefaultTTL, "ttl", config.DefaultTTL, "Default paste expiration time")
	flag.StringVar(&config.S3Bucket, "s3-bucket", config.S3Bucket, "S3 bucket for Lambda mode")
	flag.StringVar(&config.S3Prefix, "s3-prefix", config.S3Prefix, "S3 key prefix for Lambda mode")
	flag.Parse()

	// Override with environment variables if present
	if val := os.Getenv("NCLIP_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil {
			config.Port = port
		}
	}
	if val := os.Getenv("NCLIP_URL"); val != "" {
		config.URL = val
	}
	if val := os.Getenv("NCLIP_SLUG_LENGTH"); val != "" {
		if length, err := strconv.Atoi(val); err == nil {
			config.SlugLength = length
		}
	}
	if val := os.Getenv("NCLIP_BUFFER_SIZE"); val != "" {
		if size, err := strconv.ParseInt(val, 10, 64); err == nil {
			config.BufferSize = size
		}
	}
	if val := os.Getenv("NCLIP_TTL"); val != "" {
		if ttl, err := time.ParseDuration(val); err == nil {
			config.DefaultTTL = ttl
		}
	}

	if val := os.Getenv("NCLIP_S3_BUCKET"); val != "" {
		config.S3Bucket = val
	}

	if val := os.Getenv("NCLIP_S3_PREFIX"); val != "" {
		config.S3Prefix = val
	}

	return config
}
