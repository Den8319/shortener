package db

import (
	"context"
	"fmt"

	"github.com/Den8319/shortener/internal/model"
)

// MockDB is a mock implementation of the model.Loader interface
type MockDB struct {
	// Storage map to simulate database
	storage map[string]string
}

// NewMockDB creates a new mock database instance
func NewMockDB() *MockDB {
	return &MockDB{
		storage: make(map[string]string),
	}
}

// Save saves a URL to the mock storage
func (m *MockDB) Save(ctx context.Context, url *model.URL) error {
	m.storage[url.ShortURL] = url.LongURL
	return nil
}

// GetLongURL retrieves a long URL by its short version from mock storage
func (m *MockDB) GetLongURL(ctx context.Context, shortURL string) (string, error) {
	if longURL, exists := m.storage[shortURL]; exists {
		return longURL, nil
	}
	return "", fmt.Errorf("url not found")
}

// Load returns all stored URLs from mock storage
func (m *MockDB) Load(ctx context.Context) (map[string]string, error) {
	// Create a copy of the storage to prevent external modifications
	result := make(map[string]string)
	for k, v := range m.storage {
		result[k] = v
	}
	return result, nil
}

// Close implements the Loader interface (no-op for mock)
func (m *MockDB) Close() error {
	return nil
}
