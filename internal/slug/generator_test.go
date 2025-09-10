package slug

import (
	"fmt"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	generator := New(8)

	if generator.length != 8 {
		t.Errorf("Expected length 8, got %d", generator.length)
	}

	if generator.symbols != defaultSymbols {
		t.Errorf("Expected default symbols, got %s", generator.symbols)
	}
}

func TestNewWithSymbols(t *testing.T) {
	customSymbols := "abc123"
	generator := NewWithSymbols(6, customSymbols)

	if generator.length != 6 {
		t.Errorf("Expected length 6, got %d", generator.length)
	}

	if generator.symbols != customSymbols {
		t.Errorf("Expected custom symbols %s, got %s", customSymbols, generator.symbols)
	}
}

func TestNewWithSymbols_EmptySymbols(t *testing.T) {
	generator := NewWithSymbols(4, "")

	if generator.symbols != defaultSymbols {
		t.Errorf("Expected default symbols when empty provided, got %s", generator.symbols)
	}
}

func TestGenerate(t *testing.T) {
	generator := New(10)

	slug, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(slug) != 10 {
		t.Errorf("Expected slug length 10, got %d", len(slug))
	}

	// Check that all characters are from the allowed set
	for _, char := range slug {
		if !strings.ContainsRune(defaultSymbols, char) {
			t.Errorf("Slug contains invalid character: %c", char)
		}
	}
}

func TestGenerateLength(t *testing.T) {
	generator := New(5)

	testCases := []int{1, 3, 8, 15, 20}

	for _, length := range testCases {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			slug, err := generator.GenerateLength(length)
			if err != nil {
				t.Fatalf("GenerateLength(%d) failed: %v", length, err)
			}

			if len(slug) != length {
				t.Errorf("Expected slug length %d, got %d", length, len(slug))
			}

			// Check that all characters are from the allowed set
			for _, char := range slug {
				if !strings.ContainsRune(defaultSymbols, char) {
					t.Errorf("Slug contains invalid character: %c", char)
				}
			}
		})
	}
}

func TestGenerateLength_ZeroLength(t *testing.T) {
	generator := New(7)

	slug, err := generator.GenerateLength(0)
	if err != nil {
		t.Fatalf("GenerateLength(0) failed: %v", err)
	}

	// Should fall back to generator's default length
	if len(slug) != 7 {
		t.Errorf("Expected slug length 7 (fallback), got %d", len(slug))
	}
}

func TestGenerateUniqueness(t *testing.T) {
	generator := New(8)

	// Generate multiple slugs and check they're different
	slugs := make(map[string]bool)
	for i := 0; i < 100; i++ {
		slug, err := generator.Generate()
		if err != nil {
			t.Fatalf("Generate failed on iteration %d: %v", i, err)
		}

		if slugs[slug] {
			t.Errorf("Generated duplicate slug: %s", slug)
		}
		slugs[slug] = true
	}
}

func TestGenerateWithCollisionCheck(t *testing.T) {
	generator := New(4)

	// Create a collision checker that fails for specific slugs
	existingSlugs := map[string]bool{
		"test": true,
		"abc1": true,
		"xyz9": true,
	}

	checkExists := func(slug string) bool {
		return existingSlugs[slug]
	}

	// Generate multiple slugs and ensure none collide
	for i := 0; i < 20; i++ {
		slug, err := generator.GenerateWithCollisionCheck(checkExists)
		if err != nil {
			t.Fatalf("GenerateWithCollisionCheck failed: %v", err)
		}

		if existingSlugs[slug] {
			t.Errorf("Generated colliding slug: %s", slug)
		}

		// Add to existing slugs to test continued uniqueness
		existingSlugs[slug] = true
	}
}

func TestGenerateWithCollisionCheck_HighCollisionRate(t *testing.T) {
	// Use very short slugs with limited symbols to force collisions
	generator := NewWithSymbols(2, "ab")

	existingSlugs := map[string]bool{
		"aa": true,
		"ab": true,
		"ba": true,
		// Only "bb" is free initially
	}

	checkExists := func(slug string) bool {
		return existingSlugs[slug]
	}

	// Should still be able to generate "bb"
	slug, err := generator.GenerateWithCollisionCheck(checkExists)
	if err != nil {
		t.Fatalf("GenerateWithCollisionCheck failed: %v", err)
	}

	if len(slug) < 2 {
		t.Errorf("Expected slug length >= 2, got %d", len(slug))
	}

	if existingSlugs[slug] {
		t.Errorf("Generated colliding slug: %s", slug)
	}
}

func TestGenerateWithCollisionCheck_AllCollisions(t *testing.T) {
	// Force all possible collisions for very short length
	generator := NewWithSymbols(1, "a")

	// "a" already exists, so should generate longer slug
	checkExists := func(slug string) bool {
		return slug == "a"
	}

	slug, err := generator.GenerateWithCollisionCheck(checkExists)
	if err != nil {
		t.Fatalf("GenerateWithCollisionCheck failed: %v", err)
	}

	// Should generate a longer slug when all short ones collide
	if len(slug) <= 1 {
		t.Errorf("Expected slug length > 1 due to collisions, got %d", len(slug))
	}
}

func BenchmarkGenerate(b *testing.B) {
	generator := New(8)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Generate()
		if err != nil {
			b.Fatalf("Generate failed: %v", err)
		}
	}
}

func BenchmarkGenerateWithCollisionCheck(b *testing.B) {
	generator := New(8)

	// Simulate 10% collision rate
	existingSlugs := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		slug, _ := generator.Generate()
		if i%10 == 0 { // 10% collision rate
			existingSlugs[slug] = true
		}
	}

	checkExists := func(slug string) bool {
		return existingSlugs[slug]
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.GenerateWithCollisionCheck(checkExists)
		if err != nil {
			b.Fatalf("GenerateWithCollisionCheck failed: %v", err)
		}
	}
}
