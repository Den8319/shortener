package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid http", "http://example.com", true},
		{"valid https", "https://example.com/path?query=1", true},
		{"valid with port", "https://example.com:8080", true},
		{"invalid ftp", "ftp://example.com", false},
		{"invalid no scheme", "example.com", false},
		{"invalid no host", "http://", false},
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"invalid format", "not a url", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidURL(tt.input)
			assert.Equal(t, tt.want, got, "isValidURL(%q)", tt.input)
		})
	}
}
