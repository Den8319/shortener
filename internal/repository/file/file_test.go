package file_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Den8319/shortener/internal/model"
	"github.com/Den8319/shortener/internal/repository/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLoader(t *testing.T, filePath string) *file.Fileloader {
	t.Helper()
	fl, err := file.New(filePath)
	require.NoError(t, err)
	t.Cleanup(func() { fl.Close() })
	return fl
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		path    func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "creates new file if not exists",
			path: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "store.json")
			},
			wantErr: false,
		},
		{
			name: "opens existing file",
			path: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "store.json")
				f, err := os.Create(p)
				require.NoError(t, err)
				f.Close()
				return p
			},
			wantErr: false,
		},
		{
			name: "error on inaccessible path",
			path: func(t *testing.T) string {
				return "/nonexistent/dir/store.json"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fl, err := file.New(tt.path(t))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			defer fl.Close()
			assert.NotNil(t, fl)
		})
	}
}

func TestSaveAndLoad(t *testing.T) {
	tests := []struct {
		name    string
		records []struct{ short, long string }
		want    map[string]string
	}{
		{
			name:    "empty file",
			records: nil,
			want:    map[string]string{},
		},
		{
			name: "single record saved and loaded",
			records: []struct{ short, long string }{
				{"abc12345", "https://yandex.ru"},
			},
			want: map[string]string{
				"abc12345": "https://yandex.ru",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "store.json")
			fl := newTestLoader(t, path)

			for _, rec := range tt.records {
				err := fl.Save(context.Background(), &model.URL{ShortURL: rec.short, LongURL: rec.long}, "test-user-uuid")
				require.NoError(t, err)
			}
			fl.Close()

			fl2 := newTestLoader(t, path)
			got, err := fl2.Load(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoad_CreatesFileIfNotExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new-store.json")
	fl := newTestLoader(t, path)

	got, err := fl.Load(context.Background())
	require.NoError(t, err)
	assert.Empty(t, got)

	_, err = os.Stat(path)
	assert.NoError(t, err, "file should exist after Load")
}

func TestGetLongURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	fl := newTestLoader(t, path)

	err := fl.Save(context.Background(), &model.URL{ShortURL: "abc12345", LongURL: "https://example.com"}, "user-1")
	require.NoError(t, err)

	got, err := fl.GetLongURL(context.Background(), "abc12345")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", got)

	_, err = fl.GetLongURL(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestGetUserURLs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	fl := newTestLoader(t, path)

	err := fl.Save(context.Background(), &model.URL{ShortURL: "short1", LongURL: "https://a.com"}, "user-1")
	require.NoError(t, err)
	err = fl.Save(context.Background(), &model.URL{ShortURL: "short2", LongURL: "https://b.com"}, "user-1")
	require.NoError(t, err)
	err = fl.Save(context.Background(), &model.URL{ShortURL: "short3", LongURL: "https://c.com"}, "user-2")
	require.NoError(t, err)

	urls, err := fl.GetUserURLs(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Len(t, urls, 2)

	urls2, err := fl.GetUserURLs(context.Background(), "user-2")
	require.NoError(t, err)
	assert.Len(t, urls2, 1)
	assert.Equal(t, "https://c.com", urls2[0].LongURL)

	urls3, err := fl.GetUserURLs(context.Background(), "nonexistent-user")
	require.NoError(t, err)
	assert.Empty(t, urls3)
}
