package storage

import (
	"testing"
)

func TestNewS3Store_EmptyBucket(t *testing.T) {
	_, err := NewS3Store("", "prefix")
	if err == nil {
		t.Error("expected error for empty bucket")
	}
}

func TestS3Store_Constructor(t *testing.T) {
	store, err := NewS3Store("bucket", "prefix")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.bucket != "bucket" {
		t.Errorf("expected bucket 'bucket', got %s", store.bucket)
	}
	// Accept both 'prefix' and 'prefix/' for flexibility
	if store.prefix != "prefix" && store.prefix != "prefix/" {
		t.Errorf("expected prefix 'prefix' or 'prefix/', got %s", store.prefix)
	}
}

// Do not call S3Store methods that require a real client
