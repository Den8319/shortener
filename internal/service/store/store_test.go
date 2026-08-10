package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Den8319/shortener/internal/model"
	file "github.com/Den8319/shortener/internal/repository/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStoreB(b *testing.B) *Store {
	b.Helper()
	tmpFile, err := os.CreateTemp("", "store-*.json")
	require.NoError(b, err)
	_ = tmpFile.Close()

	repo, err := file.New(tmpFile.Name())
	require.NoError(b, err)
	s, err := New(repo)
	require.NoError(b, err)

	for i := range 100 {
		url := fmt.Sprintf("https://benchmark-test-url-%d.com", i)
		_, err := s.GetShortURL(context.Background(), url, "benchmark-user")
		require.NoError(b, err)
	}

	b.Cleanup(func() {
		os.Remove(tmpFile.Name())
		s.Close()
	})

	return s
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	fileloader, err := file.New(filepath.Join(t.TempDir(), "store.json"))
	require.NoError(t, err)
	s, err := New(fileloader)
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
				short, err := s.GetShortURL(context.Background(), u, "dfsfsdfsdfv")
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
				short, _ := s.GetShortURL(context.Background(), "https://example.com", "dfsfsdfsdfv")
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
			long, err := s.GetLongURL(context.Background(), short)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantLong, long)
		})
	}
}

func BenchmarkGetShortURL(b *testing.B) {
	s := newTestStoreB(b)
	baseURL := "https://example.com/test"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		url := baseURL + "-url-" + fmt.Sprintf("%d-%d", i, time.Now().UnixNano())
		_, err := s.GetShortURL(context.Background(), url, "test-user")
		if err != nil {
			b.Errorf("GetShortURL() error = %v", err)
		}
	}
}

func BenchmarkGetLongURL(b *testing.B) {
	s := newTestStoreB(b)
	url := "https://example.com/test"
	short, _ := s.GetShortURL(context.Background(), url, "test-user")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.GetLongURL(context.Background(), short)
		if err != nil {
			b.Errorf("GetLongURL() error = %v", err)
		}
	}
}

func BenchmarkFindOrGenerate(b *testing.B) {
	s := newTestStoreB(b)
	url := "https://example.com/test"

	for i := range 100 {
		_, _ = s.GetShortURL(context.Background(), url+"-"+string(rune('a'+i%26)), "test-user")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = s.findOrGenerate(url)
	}
}

func TestGetShortList(t *testing.T) {
	s := newTestStore(t)

	items := []model.BatchRequestItem{
		{Corr: "1", LongURL: "https://example.com"},
		{Corr: "2", LongURL: "https://google.com"},
	}

	results, err := s.GetShortList(context.Background(), items, "user-1")
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "1", results[0].Corr)
	assert.Equal(t, "2", results[1].Corr)
	assert.NotEmpty(t, results[0].ShortURL)
	assert.NotEmpty(t, results[1].ShortURL)

	// Повторный вызов — существующие URL возвращают те же короткие
	results2, err := s.GetShortList(context.Background(), items, "user-1")
	require.NoError(t, err)
	assert.Equal(t, results[0].ShortURL, results2[0].ShortURL)
	assert.Equal(t, results[1].ShortURL, results2[1].ShortURL)
}

// mockLoader — простая реализация model.Loader для тестов
type mockLoader struct {
	storage map[string]model.URL
}

func newMockLoader() *mockLoader {
	return &mockLoader{storage: make(map[string]model.URL)}
}

func (m *mockLoader) Save(_ context.Context, url *model.URL, userUUID string) error {
	url.UserUUID = userUUID
	m.storage[url.ShortURL] = *url
	return nil
}

func (m *mockLoader) GetLongURL(_ context.Context, shortURL string) (string, error) {
	if u, ok := m.storage[shortURL]; ok {
		if u.IsDeleted {
			return "", model.ErrURLDeleted
		}
		return u.LongURL, nil
	}
	return "", fmt.Errorf("not found")
}

func (m *mockLoader) Load(_ context.Context) (map[string]string, error) {
	result := make(map[string]string)
	for k, v := range m.storage {
		if !v.IsDeleted {
			result[k] = v.LongURL
		}
	}
	return result, nil
}

func (m *mockLoader) Close() error { return nil }

func (m *mockLoader) GetUserURLs(_ context.Context, userUUID string) ([]model.URL, error) {
	var result []model.URL
	for _, u := range m.storage {
		if u.UserUUID == userUUID && !u.IsDeleted {
			result = append(result, u)
		}
	}
	return result, nil
}

func (m *mockLoader) Delete(_ context.Context, shortURLs []string, userUUID string) error {
	for _, s := range shortURLs {
		if u, ok := m.storage[s]; ok && u.UserUUID == userUUID {
			u.IsDeleted = true
			m.storage[s] = u
		}
	}
	return nil
}

func TestDeleteURLs(t *testing.T) {
	loader := newMockLoader()
	s, err := New(loader)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	short1, err := s.GetShortURL(context.Background(), "https://example.com", "user-1")
	require.NoError(t, err)
	short2, err := s.GetShortURL(context.Background(), "https://google.com", "user-1")
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, s.StartDeleter(2))
	defer s.CloseDeleter()

	err = s.DeleteURLs(ctx, []string{short1, short2}, "user-1")
	require.NoError(t, err)

	// Даём время воркеру обработать
	time.Sleep(100 * time.Millisecond)

	// Удалённые URL должны возвращать ошибку
	_, err = s.GetLongURL(context.Background(), short1)
	assert.Error(t, err)
	_, err = s.GetLongURL(context.Background(), short2)
	assert.Error(t, err)
}

func TestStartDeleter_DoubleStart(t *testing.T) {
	loader := newMockLoader()
	s, err := New(loader)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })

	require.NoError(t, s.StartDeleter(2))
	defer s.CloseDeleter()

	// Повторный вызов должен вернуть ошибку
	err = s.StartDeleter(2)
	assert.Error(t, err)

	// После CloseDeleter можно запустить заново
	s.CloseDeleter()
	require.NoError(t, s.StartDeleter(2))
	s.CloseDeleter()
}
