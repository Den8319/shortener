package db

import (
	"context"
	"testing"

	"github.com/Den8319/shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDB_GetURL(t *testing.T) {
	// Create mock database instance instead of real DB connection
	dbInstance := NewMockDB()

	// Вставляем тестовые данные
	testURL := &model.URL{
		ShortURL: "abc12345",
		LongURL:  "https://example.com",
	}

	err := dbInstance.Save(context.Background(), testURL)
	require.NoError(t, err)

	// Тестируем GetURL
	t.Run("found", func(t *testing.T) {
		url, err := dbInstance.GetLongURL(context.Background(), testURL.ShortURL)
		require.NoError(t, err)
		assert.Equal(t, testURL.LongURL, url)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := dbInstance.GetLongURL(context.Background(), "nonexistent")
		assert.Error(t, err)
	})
}

func TestDB_Load(t *testing.T) {
	dbInstance := NewMockDB()

	testData := []struct {
		short string
		long  string
	}{
		{"abc1", "https://yandex.ru"},
		{"abc2", "https://google.com"},
	}

	for _, d := range testData {
		err := dbInstance.Save(context.Background(), &model.URL{
			ShortURL: d.short,
			LongURL:  d.long,
		})
		require.NoError(t, err)
	}

	urls, err := dbInstance.Load(context.Background())
	require.NoError(t, err)
	assert.Len(t, urls, len(testData))

	for _, d := range testData {
		assert.Equal(t, d.long, urls[d.short])
	}
}
