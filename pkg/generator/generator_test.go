package generator

import (
	"strings"
	"testing"
)

func TestGenerateShort(t *testing.T) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"

	tests := []struct {
		name   string
		length int
	}{
		{name: "standard length 8", length: 8},
		{name: "zero length", length: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateShort(tt.length)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) != tt.length {
				t.Errorf("expected length %d, got %d", tt.length, len(result))
			}

			// Проверяем, что все символы из допустимого алфавита
			for _, ch := range result {
				if !strings.ContainsRune(alphabet, ch) {
					t.Errorf("character %q is not in alphabet", ch)
				}
			}
		})
	}
}