package utils

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// SecureRandomSlug generates a random slug of given length using crypto/rand and a custom charset
func SecureRandomSlug(length int) (string, error) {
	if length < 3 || length > 32 {
		length = 5
	}
	charset := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no O, I, 0, 1
	result := make([]byte, length)
	for i := range result {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[idx.Int64()]
	}
	return string(result), nil
}

// GenerateSlugBatch returns a batch of candidate slugs
func GenerateSlugBatch(batchSize, length int) ([]string, error) {
	slugs := make([]string, batchSize)
	for i := 0; i < batchSize; i++ {
		slug, err := SecureRandomSlug(length)
		if err != nil {
			return nil, err
		}
		slugs[i] = slug
	}
	return slugs, nil
}

// IsValidSlug checks if a slug contains only valid characters
func IsValidSlug(slug string) bool {
	// Slug must be between 3 and 32 characters
	if len(slug) < 3 || len(slug) > 32 {
		return false
	}
	allowed := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for _, char := range slug {
		if !strings.ContainsRune(allowed, char) {
			return false
		}
	}
	return true
}
