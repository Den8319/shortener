package db

import (
	"context"
	"fmt"

	"github.com/Den8319/shortener/internal/model"
)

// MockDB is a mock implementation of the model.Loader interface
type MockDB struct {
	storage map[string]model.URL
}

// NewMockDB creates a new mock database instance
func NewMockDB() *MockDB {
	return &MockDB{
		storage: make(map[string]model.URL),
	}
}

// Save saves a URL to the mock storage
func (m *MockDB) Save(ctx context.Context, url *model.URL, userUUID string) error {
	url.UserUUID = userUUID
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
	result := make(map[string]string)
	for k, v := range m.storage {
		if !v.IsDeleted {
			result[k] = v.LongURL
		}
	}
	return result, nil
}

// Delete marks URLs as deleted for the specified user
func (m *MockDB) Delete(ctx context.Context, shortURLs []string, userUUID string) error {
	for _, shortURL := range shortURLs {
		if url, exists := m.storage[shortURL]; exists {
			if url.UserUUID == userUUID {
				url.IsDeleted = true
				m.storage[shortURL] = url
			}
		}
	}
	return nil
}

// GetUserURLs returns all URLs for the specified user
func (m *MockDB) GetUserURLs(ctx context.Context, userUUID string) ([]model.URL, error) {
	var result []model.URL
	for _, url := range m.storage {
		if url.UserUUID == userUUID && !url.IsDeleted {
			result = append(result, url)
		}
	}
	return result, nil
}

// Close implements the Loader interface (no-op for mock)
func (m *MockDB) Close() error {
	return nil
}
