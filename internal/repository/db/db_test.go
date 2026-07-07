package db

import (
	"context"
	"testing"

	"github.com/Den8319/shortener/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDB_GetURL(t *testing.T) {
	dbInstance := NewMockDB()
	ctx := context.Background()

	testURL := &model.URL{
		ShortURL: "abc12345",
		LongURL:  "https://example.com",
	}
	err := dbInstance.Save(ctx, testURL, "user-1")  
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		url, err := dbInstance.GetLongURL(ctx, testURL.ShortURL)
		require.NoError(t, err)
		assert.Equal(t, testURL.LongURL, url)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := dbInstance.GetLongURL(ctx, "nonexistent")
		assert.Error(t, err)
	})
}

func TestDB_Load(t *testing.T) {
	dbInstance := NewMockDB()
	ctx := context.Background()

	testData := []struct {
		short string
		long  string
	}{
		{"abc1", "https://yandex.ru"},
		{"abc2", "https://google.com"},
	}

	for _, d := range testData {
		err := dbInstance.Save(ctx, &model.URL{
			ShortURL: d.short,
			LongURL:  d.long,
		}, "user-1") // ← добавлен userUUID
		require.NoError(t, err)
	}

	urls, err := dbInstance.Load(ctx)
	require.NoError(t, err)
	assert.Len(t, urls, len(testData))

	for _, d := range testData {
		assert.Equal(t, d.long, urls[d.short])
	}
}

func TestMockDB_GetLongURL(t *testing.T) {
	mock := NewMockDB()
	ctx := context.Background()

	err := mock.Save(ctx, &model.URL{
		ShortURL: "abc123",
		LongURL:  "https://example.com",
		UserUUID: "user-1",
	}, "user-1") // ← добавлен userUUID
	require.NoError(t, err)

	t.Run("found", func(t *testing.T) {
		longURL, err := mock.GetLongURL(ctx, "abc123")
		require.NoError(t, err)
		assert.Equal(t, "https://example.com", longURL)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := mock.GetLongURL(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "url not found")
	})

	t.Run("deleted URL", func(t *testing.T) {
		err := mock.Delete(ctx, []string{"abc123"}, "user-1")
		require.NoError(t, err)

		_, err = mock.GetLongURL(ctx, "abc123")
		assert.ErrorIs(t, err, model.ErrURLDeleted)
	})
}