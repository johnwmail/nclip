package slug

import (
	"crypto/rand"
	"math/big"
)

// Symbols used for slug generation (alphanumeric, lowercase for consistency)
const defaultSymbols = "abcdefghijklmnopqrstuvwxyz0123456789"

// Generator handles slug generation with collision detection
type Generator struct {
	symbols string
	length  int
}

// New creates a new slug generator
func New(length int) *Generator {
	return &Generator{
		symbols: defaultSymbols,
		length:  length,
	}
}

// NewWithSymbols creates a slug generator with custom symbols
func NewWithSymbols(length int, symbols string) *Generator {
	if symbols == "" {
		symbols = defaultSymbols
	}
	return &Generator{
		symbols: symbols,
		length:  length,
	}
}

// Generate creates a new random slug of the configured length
func (g *Generator) Generate() (string, error) {
	return g.GenerateLength(g.length)
}

// GenerateLength creates a random slug of the specified length
func (g *Generator) GenerateLength(length int) (string, error) {
	if length <= 0 {
		length = g.length
	}

	result := make([]byte, length)
	symbolsLen := big.NewInt(int64(len(g.symbols)))

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, symbolsLen)
		if err != nil {
			return "", err
		}
		result[i] = g.symbols[n.Int64()]
	}

	return string(result), nil
}

// GenerateWithCollisionCheck generates slugs until one doesn't collide
func (g *Generator) GenerateWithCollisionCheck(checkExists func(string) bool) (string, error) {
	length := g.length
	maxAttempts := 1000 // Prevent infinite loops

	for attempt := 0; attempt < maxAttempts; attempt++ {
		slug, err := g.GenerateLength(length)
		if err != nil {
			return "", err
		}

		if !checkExists(slug) {
			return slug, nil
		}

		// If we've tried many times with the base length, increase it
		if attempt > 0 && attempt%100 == 0 {
			length++
		}
	}

	// If we still haven't found a unique slug, generate one more with extra length
	return g.GenerateLength(length + 2)
}
