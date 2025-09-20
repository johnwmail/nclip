package utils

import (
	"strings"
	"testing"
)

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name   string
		length int
		want   int // expected length
	}{
		{
			name:   "default length",
			length: 5,
			want:   5,
		},
		{
			name:   "custom length",
			length: 10,
			want:   10,
		},
		{
			name:   "min valid length",
			length: 3,
			want:   3,
		},
		{
			name:   "max valid length",
			length: 32,
			want:   32,
		},
		{
			name:   "below min length defaults to 5",
			length: 2,
			want:   5,
		},
		{
			name:   "above max length defaults to 5",
			length: 33,
			want:   5,
		},
		{
			name:   "zero length defaults to 5",
			length: 0,
			want:   5,
		},
		{
			name:   "negative length defaults to 5",
			length: -1,
			want:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug, err := GenerateSlug(tt.length)
			if err != nil {
				t.Errorf("GenerateSlug() error = %v", err)
				return
			}
			if len(slug) != tt.want {
				t.Errorf("GenerateSlug() length = %v, want %v", len(slug), tt.want)
			}

			// Verify slug contains only valid characters
			if !IsValidSlug(slug) {
				t.Errorf("GenerateSlug() generated invalid slug: %v", slug)
			}
		})
	}
}

func TestIsValidSlug(t *testing.T) {
	tests := []struct {
		name string
		slug string
		want bool
	}{
		{
			name: "too short",
			slug: "AB",
			want: false,
		},
		{
			name: "too long",
			slug: strings.Repeat("A", 33),
			want: false,
		},
		{
			name: "min valid length",
			slug: "ABC",
			want: true,
		},
		{
			name: "max valid length",
			slug: strings.Repeat("A", 32),
			want: true,
		},
		{
			name: "empty string",
			slug: "",
			want: false,
		},
		{
			name: "valid alphanumeric",
			slug: "ABC234",
			want: true,
		},
		{
			name: "contains invalid character",
			slug: "ABC-234",
			want: false,
		},
		{
			name: "contains space",
			slug: "ABC 234",
			want: false,
		},
		{
			name: "contains special characters",
			slug: "ABC@234",
			want: false,
		},
		{
			name: "contains lowercase",
			slug: "abc234",
			want: false,
		},
		{
			name: "contains confusing chars 0,1,O,I",
			slug: "AB01OI",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSlug(tt.slug); got != tt.want {
				t.Errorf("IsValidSlug() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlugCharacterSet(t *testing.T) {
	// Generate many slugs to test character distribution
	slugs := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		slug, err := GenerateSlug(10)
		if err != nil {
			t.Fatalf("GenerateSlug() error = %v", err)
		}
		slugs[i] = slug
	}

	// Collect all characters used
	charCount := make(map[rune]int)
	for _, slug := range slugs {
		for _, char := range slug {
			charCount[char]++
		}
	}

	// Verify only expected characters are used
	for char := range charCount {
		if !strings.ContainsRune(charset, char) {
			t.Errorf("Unexpected character found in generated slugs: %c", char)
		}
	}
}

func TestSlugUniqueness(t *testing.T) {
	// Generate multiple slugs and verify they're different
	slugs := make(map[string]bool)
	duplicates := 0

	for i := 0; i < 1000; i++ {
		slug, err := GenerateSlug(5)
		if err != nil {
			t.Fatalf("GenerateSlug() error = %v", err)
		}

		if slugs[slug] {
			duplicates++
		}
		slugs[slug] = true
	}

	// With a 5-character slug from a charset of reasonable size,
	// we shouldn't see many duplicates in 1000 attempts
	if duplicates > 10 {
		t.Errorf("Too many duplicate slugs generated: %d out of 1000", duplicates)
	}
}

func TestSlugDoesNotContainConfusingCharacters(t *testing.T) {
	// Generate many slugs to verify no confusing characters are used
	confusingChars := "01OI" // chars that can be confused with other chars

	for i := 0; i < 1000; i++ {
		slug, err := GenerateSlug(10)
		if err != nil {
			t.Fatalf("GenerateSlug() error = %v", err)
		}

		for _, char := range slug {
			if strings.ContainsRune(confusingChars, char) {
				t.Errorf("Slug contains confusing character %c: %s", char, slug)
			}
		}
	}
}

func TestSlugOnlyContainsUppercaseAndNumbers(t *testing.T) {
	// Generate many slugs to verify only uppercase letters and numbers (excluding 0,1,O,I) are used
	for i := 0; i < 1000; i++ {
		slug, err := GenerateSlug(10)
		if err != nil {
			t.Fatalf("GenerateSlug() error = %v", err)
		}

		for _, char := range slug {
			if char >= 'a' && char <= 'z' {
				t.Errorf("Slug contains lowercase character %c: %s", char, slug)
			}
			if !((char >= 'A' && char <= 'Z') || (char >= '2' && char <= '9')) {
				t.Errorf("Slug contains invalid character %c: %s", char, slug)
			}
			// Check for excluded characters
			if char == '0' || char == '1' || char == 'O' || char == 'I' {
				t.Errorf("Slug contains excluded character %c: %s", char, slug)
			}
		}
	}
}
