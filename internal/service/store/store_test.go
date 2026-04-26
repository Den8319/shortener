package store

import (
	"path/filepath"
	"testing"

	fileRepo "github.com/Den8319/shortener/internal/repository/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	repo, err := fileRepo.New(filepath.Join(t.TempDir(), "store.json"))
	require.NoError(t, err)
	s, err := New(repo)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func TestGetShortURL(t *testing.T) {
	tests := []struct {
		name    string
		urls    []string
		wantErr bool
	}{
		{
			name:    "new URL returns non-empty short",
			urls:    []string{"https://yandex.ru"},
			wantErr: false,
		},
		{
			name:    "same URL returns same short",
			urls:    []string{"https://yandex.com", "https://yandex.com"},
			wantErr: false,
		},
		{
			name:    "different URLs return different shorts",
			urls:    []string{"https://yandex.com", "https://google.com"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)

			var shorts []string
			for _, u := range tt.urls {
				short, err := s.GetShortURL(u)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.NotEmpty(t, short)
				shorts = append(shorts, short)
			}

			if len(tt.urls) == 2 && tt.urls[0] == tt.urls[1] {
				assert.Equal(t, shorts[0], shorts[1], "same urls")
			}
			if len(tt.urls) == 2 && tt.urls[0] != tt.urls[1] {
				assert.NotEqual(t, shorts[0], shorts[1], "different urls")
			}
		})
	}
}

func TestGetLongURL(t *testing.T) {
	tests := []struct {
		name     string
		longURL  string
		lookupFn func(s *Store) string
		wantLong string
		wantErr  bool
	}{
		{
			name:    "existing short returns long",
			longURL: "https://example.com",
			lookupFn: func(s *Store) string {
				short, _ := s.GetShortURL("https://example.com")
				return short
			},
			wantLong: "https://example.com",
			wantErr:  false,
		},
		{
			name:    "unknown short returns error",
			longURL: "",
			lookupFn: func(_ *Store) string {
				return "nonexistent"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)

			short := tt.lookupFn(s)
			long, err := s.GetLongURL(short)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantLong, long)
		})
	}
}
