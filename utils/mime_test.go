package utils

import (
	"testing"
)

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  []byte
		want     string
	}{
		{
			name:     "detect from filename - text",
			filename: "test.txt",
			content:  []byte("hello world"),
			want:     "text/plain; charset=utf-8",
		},
		{
			name:     "detect from filename - json",
			filename: "config.json",
			content:  []byte(`{"key": "value"}`),
			want:     "application/json",
		},
		{
			name:     "detect from filename - zip",
			filename: "archive.zip",
			content:  []byte("PK\x03\x04"), // ZIP header
			want:     "application/zip",
		},
		{
			name:     "detect from content - html",
			filename: "",
			content:  []byte("<html><body>test</body></html>"),
			want:     "text/html; charset=utf-8",
		},
		{
			name:     "detect from content - binary",
			filename: "",
			content:  []byte{0x00, 0x01, 0x02, 0x03},
			want:     "application/octet-stream",
		},
		{
			name:     "empty content and filename",
			filename: "",
			content:  []byte{},
			want:     "application/octet-stream",
		},
		{
			name:     "filename without extension",
			filename: "README",
			content:  []byte("This is a readme file"),
			want:     "text/plain; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectContentType(tt.filename, tt.content)
			if got != tt.want {
				t.Errorf("DetectContentType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTextContent(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{
			name:        "plain text",
			contentType: "text/plain",
			want:        true,
		},
		{
			name:        "html text",
			contentType: "text/html",
			want:        true,
		},
		{
			name:        "json",
			contentType: "application/json",
			want:        true,
		},
		{
			name:        "xml",
			contentType: "application/xml",
			want:        true,
		},
		{
			name:        "javascript",
			contentType: "application/javascript",
			want:        true,
		},
		{
			name:        "shell script",
			contentType: "application/x-sh",
			want:        true,
		},
		{
			name:        "yaml",
			contentType: "application/x-yaml",
			want:        true,
		},
		{
			name:        "binary - image",
			contentType: "image/png",
			want:        false,
		},
		{
			name:        "binary - zip",
			contentType: "application/zip",
			want:        false,
		},
		{
			name:        "binary - pdf",
			contentType: "application/pdf",
			want:        false,
		},
		{
			name:        "case insensitive",
			contentType: "TEXT/PLAIN",
			want:        true,
		},
		{
			name:        "with charset",
			contentType: "text/plain; charset=utf-8",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTextContent(tt.contentType)
			if got != tt.want {
				t.Errorf("IsTextContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtensionByMime(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		want     string
	}{
		{
			name:     "text plain",
			mimeType: "text/plain",
			want:     ".asc", // mime.ExtensionsByType returns .asc first
		},
		{
			name:     "json",
			mimeType: "application/json",
			want:     ".json",
		},
		{
			name:     "zip",
			mimeType: "application/zip",
			want:     ".zip",
		},
		{
			name:     "png image",
			mimeType: "image/png",
			want:     ".png",
		},
		{
			name:     "jpeg image",
			mimeType: "image/jpeg",
			want:     ".jfif", // mime.ExtensionsByType returns .jfif first
		},
		{
			name:     "pdf",
			mimeType: "application/pdf",
			want:     ".pdf",
		},
		{
			name:     "html",
			mimeType: "text/html",
			want:     ".htm", // mime.ExtensionsByType returns .htm first
		},
		{
			name:     "css",
			mimeType: "text/css",
			want:     ".css",
		},
		{
			name:     "javascript",
			mimeType: "application/javascript",
			want:     "", // No extension returned by mime.ExtensionsByType
		},
		{
			name:     "empty mime type",
			mimeType: "",
			want:     "",
		},
		{
			name:     "unknown mime type",
			mimeType: "application/x-unknown-type",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtensionByMime(tt.mimeType)
			if got != tt.want {
				t.Errorf("ExtensionByMime() = %v, want %v", got, tt.want)
			}
		})
	}
}
