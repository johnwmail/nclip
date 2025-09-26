package models

import (
	"testing"
	"time"
)

func TestPaste_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		expiresAt *time.Time
		want      bool
	}{
		{
			name:      "not expired - future date",
			expiresAt: timePtr(now.Add(1 * time.Hour)),
			want:      false,
		},
		{
			name:      "expired - past date",
			expiresAt: timePtr(now.Add(-1 * time.Hour)),
			want:      true,
		},
		{
			name:      "no expiration - nil",
			expiresAt: nil,
			want:      false,
		},
		{
			name:      "just expired - 1 second ago",
			expiresAt: timePtr(now.Add(-1 * time.Second)),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Paste{
				ExpiresAt: tt.expiresAt,
			}
			got := p.IsExpired()
			if got != tt.want {
				t.Errorf("Paste.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPaste_ShouldBurn(t *testing.T) {
	tests := []struct {
		name          string
		burnAfterRead bool
		readCount     int
		want          bool
	}{
		{
			name:          "should burn - burn enabled and read",
			burnAfterRead: true,
			readCount:     1,
			want:          true,
		},
		{
			name:          "should burn - burn enabled and multiple reads",
			burnAfterRead: true,
			readCount:     5,
			want:          true,
		},
		{
			name:          "should not burn - burn disabled",
			burnAfterRead: false,
			readCount:     1,
			want:          false,
		},
		{
			name:          "should not burn - burn enabled but not read",
			burnAfterRead: true,
			readCount:     0,
			want:          false,
		},
		{
			name:          "should not burn - both false",
			burnAfterRead: false,
			readCount:     0,
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Paste{
				BurnAfterRead: tt.burnAfterRead,
				ReadCount:     tt.readCount,
			}
			got := p.ShouldBurn()
			if got != tt.want {
				t.Errorf("Paste.ShouldBurn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPaste_StructFields(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)
	content := []byte("test content")

	paste := &Paste{
		ID:            "ABC123",
		CreatedAt:     now,
		ExpiresAt:     &expiresAt,
		Size:          int64(len(content)),
		ContentType:   "text/plain",
		BurnAfterRead: true,
		ReadCount:     0,
		//		Content:       content,
	}

	// Test all fields are set correctly
	if paste.ID != "ABC123" {
		t.Errorf("Expected ID to be 'ABC123', got %s", paste.ID)
	}

	if paste.CreatedAt != now {
		t.Errorf("Expected CreatedAt to be %v, got %v", now, paste.CreatedAt)
	}

	if paste.ExpiresAt == nil || !paste.ExpiresAt.Equal(expiresAt) {
		t.Errorf("Expected ExpiresAt to be %v, got %v", expiresAt, paste.ExpiresAt)
	}

	if paste.Size != int64(len(content)) {
		t.Errorf("Expected Size to be %d, got %d", len(content), paste.Size)
	}

	if paste.ContentType != "text/plain" {
		t.Errorf("Expected ContentType to be 'text/plain', got %s", paste.ContentType)
	}

	if !paste.BurnAfterRead {
		t.Errorf("Expected BurnAfterRead to be true, got false")
	}

	if paste.ReadCount != 0 {
		t.Errorf("Expected ReadCount to be 0, got %d", paste.ReadCount)
	}

	//	if string(paste.Content) != "test content" {
	//		t.Errorf("Expected Content to be 'test content', got %s", string(paste.Content))
	//	}
}

// timePtr is a helper function to create a time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
