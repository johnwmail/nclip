package utils

import (
	"errors"
	"strings"
	"testing"
)

func TestSecureRandomSlug(t *testing.T) {
	for _, length := range []int{5, 6, 7, 10, 32} {
		slug, err := SecureRandomSlug(length)
		if err != nil {
			t.Errorf("SecureRandomSlug(%d) error: %v", length, err)
		}
		if len(slug) != length {
			t.Errorf("SecureRandomSlug(%d) length = %d, want %d", length, len(slug), length)
		}
		// Should be custom charset, uppercase, no O/I/0/1
		allowed := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
		for _, c := range slug {
			if !strings.ContainsRune(allowed, c) {
				t.Errorf("SecureRandomSlug contains invalid char: %c", c)
			}
		}
	}
}

func TestGenerateSlugBatch(t *testing.T) {
	batch, err := GenerateSlugBatch(5, 8)
	if err != nil {
		t.Fatalf("GenerateSlugBatch error: %v", err)
	}
	if len(batch) != 5 {
		t.Errorf("GenerateSlugBatch size = %d, want 5", len(batch))
	}
	for _, slug := range batch {
		if len(slug) != 8 {
			t.Errorf("Batch slug length = %d, want 8", len(slug))
		}
	}
}

// Mock store for collision simulation
type mockStore struct {
	taken   map[string]bool
	expired map[string]bool
}

func (m *mockStore) Get(slug string) (*struct{}, error) {
	if m.taken[slug] {
		if m.expired[slug] {
			return &struct{}{}, errors.New("expired")
		}
		return &struct{}{}, nil
	}
	return nil, errors.New("not found")
}

func TestBatchCollisionRetryLogic(t *testing.T) {
	// Simulate all slugs in first batch taken, second batch partially free
	store := &mockStore{
		taken:   make(map[string]bool),
		expired: make(map[string]bool),
	}
	// Take all first batch slugs
	firstBatch, _ := GenerateSlugBatch(5, 5)
	for _, slug := range firstBatch {
		store.taken[slug] = true
	}
	// Second batch: only first slug is free
	secondBatch, _ := GenerateSlugBatch(5, 6)
	for i, slug := range secondBatch {
		if i > 0 {
			store.taken[slug] = true
		}
	}
	// Third batch: all taken
	thirdBatch, _ := GenerateSlugBatch(5, 7)
	for _, slug := range thirdBatch {
		store.taken[slug] = true
	}
	// Simulate the logic
	batches := [][]string{firstBatch, secondBatch, thirdBatch}
	var found string
	for _, batch := range batches {
		for _, candidate := range batch {
			_, err := store.Get(candidate)
			if err != nil || store.expired[candidate] {
				found = candidate
				break
			}
		}
		if found != "" {
			break
		}
	}
	if found == "" {
		t.Errorf("No free slug found after 3 batches")
	} else if len(found) != 6 {
		t.Errorf("Expected free slug from second batch (length 6), got %s (len=%d)", found, len(found))
	}
}

// ...existing code...

func TestSecureRandomSlug_LengthAndCharset(t *testing.T) {
	tests := []struct {
		name   string
		length int
		want   int // expected length
	}{
		{"default length", 5, 5},
		{"custom length", 10, 10},
		{"min valid length", 3, 3},
		{"max valid length", 32, 32},
		{"below min length defaults to 5", 2, 5},
		{"above max length defaults to 5", 33, 5},
		{"zero length defaults to 5", 0, 5},
		{"negative length defaults to 5", -1, 5},
	}
	allowed := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug, err := SecureRandomSlug(tt.length)
			if err != nil {
				t.Errorf("SecureRandomSlug() error = %v", err)
				return
			}
			if len(slug) != tt.want {
				t.Errorf("SecureRandomSlug() length = %v, want %v", len(slug), tt.want)
			}
			for _, c := range slug {
				if !strings.ContainsRune(allowed, c) {
					t.Errorf("SecureRandomSlug() generated invalid char: %c", c)
				}
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
		{"too short", "AB", false},
		{"too long", strings.Repeat("A", 33), false},
		{"min valid length", "ABC", true},
		{"max valid length", strings.Repeat("A", 32), true},
		{"empty string", "", false},
		{"valid alphanumeric", "ABC234", true},
		{"contains invalid character", "ABC-234", false},
		{"contains space", "ABC 234", false},
		{"contains special characters", "ABC@234", false},
		{"contains lowercase", "abc234", false},
		{"contains confusing chars 0,1,O,I", "AB01OI", false},
		{"contains excluded O", "ABCO234", false},
		{"contains excluded I", "ABCI234", false},
		{"contains excluded 0", "ABC0234", false},
		{"contains excluded 1", "ABC1234", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSlug(tt.slug); got != tt.want {
				t.Errorf("IsValidSlug() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecureRandomSlug_CharacterSet(t *testing.T) {
	allowed := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	slugs := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		slug, err := SecureRandomSlug(10)
		if err != nil {
			t.Fatalf("SecureRandomSlug() error = %v", err)
		}
		slugs[i] = slug
	}
	charCount := make(map[rune]int)
	for _, slug := range slugs {
		for _, char := range slug {
			charCount[char]++
		}
	}
	for char := range charCount {
		if !strings.ContainsRune(allowed, char) {
			t.Errorf("Unexpected character found in generated slugs: %c", char)
		}
	}
}

func TestSecureRandomSlug_Uniqueness(t *testing.T) {
	slugs := make(map[string]bool)
	duplicates := 0
	for i := 0; i < 1000; i++ {
		slug, err := SecureRandomSlug(5)
		if err != nil {
			t.Fatalf("SecureRandomSlug() error = %v", err)
		}
		if slugs[slug] {
			duplicates++
		}
		slugs[slug] = true
	}
	if duplicates > 10 {
		t.Errorf("Too many duplicate slugs generated: %d out of 1000", duplicates)
	}
}

func TestSecureRandomSlug_DoesNotContainConfusingCharacters(t *testing.T) {
	confusingChars := "01OI"
	for i := 0; i < 1000; i++ {
		slug, err := SecureRandomSlug(10)
		if err != nil {
			t.Fatalf("SecureRandomSlug() error = %v", err)
		}
		for _, char := range slug {
			if strings.ContainsRune(confusingChars, char) {
				t.Errorf("Slug contains confusing character %c: %s", char, slug)
			}
		}
	}
}

func TestSlugOnlyContainsUppercaseAndNumbers(t *testing.T) {
	allowed := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	for i := 0; i < 1000; i++ {
		slug, err := SecureRandomSlug(10)
		if err != nil {
			t.Fatalf("GenerateSlug() error = %v", err)
		}
		for _, char := range slug {
			if char >= 'a' && char <= 'z' {
				t.Errorf("Slug contains lowercase character %c: %s", char, slug)
			}
			if !strings.ContainsRune(allowed, char) {
				t.Errorf("Slug contains invalid character %c: %s", char, slug)
			}
		}
	}
}
