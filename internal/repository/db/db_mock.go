package db

import (
	"context"
	"fmt"

	"github.com/Den8319/shortener/internal/model"
)

// MockDB is a mock implementation of the model.Loader interface
type MockDB struct {
	// Storage map to simulate database
	storage map[string]model.URL
}

// NewMockDB creates a new mock database instance
func NewMockDB() *MockDB {
	return &MockDB{
		storage: make(map[string]model.URL),
	}
}

// Save saves a URL to the mock storage
func (m *MockDB) Save(ctx context.Context, url *model.URL) error {
	m.storage[url.ShortURL] = *url
	return nil
}

// GetLongURL retrieves a long URL by its short version from mock storage
func (m *MockDB) GetLongURL(ctx context.Context, shortURL string) (string, error) {
	if url, exists := m.storage[shortURL]; exists {
		if url.IsDeleted {
			return "", model.ErrURLDeleted
		}
		return url.LongURL, nil
	}
	return "", fmt.Errorf("url not found")
}

// Load returns all stored URLs from mock storage (excluding deleted ones)
func (m *MockDB) Load(ctx context.Context) (map[string]string, error) {
	// Create a copy of the storage to prevent external modifications
	result := make(map[string]string)
	for k, v := range m.storage {
		if !v.IsDeleted {
			result[k] = v.LongURL
		}
	}
	return result, nil
}

// SoftDeleteBatch marks URLs as deleted for the specified user
func (m *MockDB) SoftDeleteBatch(ctx context.Context, shortURLs []string, userUUID string) error {
	for _, shortURL := range shortURLs {
		if url, exists := m.storage[shortURL]; exists {
			if url.UserUUID == userUUID {
				url.IsDeleted = true
				m.storage[shortURL] = url
			}
			// If userUUID doesn't match, silently skip (user has no right to delete)
		}
		// If URL doesn't exist, silently skip
	}
	return nil
}

// DeleteURLs is a wrapper for SoftDeleteBatch (for backward compatibility)
func (m *MockDB) DeleteURLs(ctx context.Context, shortURLs []string, userUUID string) error {
	return m.SoftDeleteBatch(ctx, shortURLs, userUUID)
}

// Close implements the Loader interface (no-op for mock)
func (m *MockDB) Close() error {
	return nil
}
