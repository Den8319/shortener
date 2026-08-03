package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Den8319/shortener/internal/auth"
	"github.com/Den8319/shortener/internal/model"
	fileRepo "github.com/Den8319/shortener/internal/repository/file"
	"github.com/Den8319/shortener/internal/service/store"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	auth.Init("test-secret-key")
}

// createTestToken создает валидный JWT токен для тестов
func createTestToken(t *testing.T, userID string) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 12)),
		},
		User: userID,
	})
	signed, err := token.SignedString([]byte("test-secret-key"))
	require.NoError(t, err)
	return signed
}

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	repo, err := fileRepo.New(t.TempDir() + "/store.json")
	if err != nil {
		t.Fatalf("failed to create file repository: %v", err)
	}
	s, err := store.New(repo)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// ============================================================
// Тесты для isValidURL
// ============================================================

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

// ============================================================
// Тесты для ShortenTextHandler
// ============================================================

func Test_ShortenTextHandler(t *testing.T) {
	longURL := "https://sberbank.ru"
	baseURL := "https://short.ru"
	s := newTestStore(t)
	h := NewHandler(s, baseURL, "test-secret-key", nil)

	route := chi.NewRouter()
	route.Post("/", h.ShortenTextHandler)

	type want struct {
		Status int
		Header string
		Type   string
	}

	checkResponse := func(t *testing.T, w *httptest.ResponseRecorder, want want) {
		t.Helper()
		assert.Equal(t, want.Status, w.Code, "ожидаемый статус код не совпадает")
		if want.Header != "" {
			assert.Equal(t, want.Type, w.Header().Get(want.Header), "тип содержимого не совпадает")
		}
		if want.Status == http.StatusCreated {
			response := strings.TrimSpace(w.Body.String())
			assert.NotEmpty(t, response, "тело ответа не должно быть пустым")
			assert.True(t, strings.HasPrefix(response, baseURL), "ответ должен быть полным коротким URL")
		}
	}

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   want
	}{
		{
			name:   "positive - первый запрос, должен создать URL",
			method: http.MethodPost,
			path:   "/",
			body:   longURL,
			want: want{
				Status: http.StatusCreated,
				Header: "Content-Type",
				Type:   "text/plain",
			},
		},
		{
			name:   "positive conflict - второй запрос, тот же URL",
			method: http.MethodPost,
			path:   "/",
			body:   longURL,
			want: want{
				Status: http.StatusConflict,
				Header: "Content-Type",
				Type:   "text/plain",
			},
		},
		{
			name:   "negative - wrong method (PUT)",
			method: http.MethodPut,
			path:   "/",
			body:   longURL,
			want: want{
				Status: http.StatusMethodNotAllowed,
			},
		},
		{
			name:   "negative - invalid URL format",
			method: http.MethodPost,
			path:   "/",
			body:   "http/sber",
			want: want{
				Status: http.StatusBadRequest,
			},
		},
		{
			name:   "negative - empty body",
			method: http.MethodPost,
			path:   "/",
			body:   "",
			want: want{
				Status: http.StatusBadRequest,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyReader := strings.NewReader(tt.body)
			req := httptest.NewRequest(tt.method, tt.path, bodyReader)
			token := createTestToken(t, "test-user-uuid")
			req.Header.Set("Auth", token)
			req.AddCookie(&http.Cookie{
				Name:  "Auth",
				Value: token,
			})
			w := httptest.NewRecorder()
			route.ServeHTTP(w, req)
			checkResponse(t, w, tt.want)
		})
	}
}

// ============================================================
// Тесты для ShortenJSONHandler
// ============================================================

func Test_ShortenJSONHandler(t *testing.T) {
	longURL := "https://sberbank.ru"
	baseURL := "https://short.ru"
	s := newTestStore(t)
	h := NewHandler(s, baseURL, "test-secret-key", nil)

	route := chi.NewRouter()
	route.Post("/api/shorten", h.ShortenJSONHandler)

	jsonBody := func(url string) *bytes.Buffer {
		body, _ := json.Marshal(map[string]string{"url": url})
		return bytes.NewBuffer(body)
	}

	type want struct {
		Status       int
		ContentType  string
		ResponseHas  string
		HeaderExists string
	}

	tests := []struct {
		name   string
		method string
		path   string
		body   *bytes.Buffer
		want   want
	}{
		{
			name:   "positive - valid json and url",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   jsonBody(longURL),
			want: want{
				Status:       http.StatusCreated,
				ContentType:  "application/json",
				ResponseHas:  "https://short.ru",
				HeaderExists: "Content-Type",
			},
		},
		{
			name:   "negative - wrong method",
			method: http.MethodGet,
			path:   "/api/shorten",
			body:   jsonBody(longURL),
			want: want{
				Status: http.StatusMethodNotAllowed,
			},
		},
		{
			name:   "negative - invalid json",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   bytes.NewBuffer([]byte("{invalid json}")),
			want: want{
				Status: http.StatusBadRequest,
			},
		},
		{
			name:   "negative - empty url in json",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   jsonBody(""),
			want: want{
				Status: http.StatusBadRequest,
			},
		},
		{
			name:   "negative - invalid url format",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   jsonBody("ftp://example.com"),
			want: want{
				Status: http.StatusBadRequest,
			},
		},
		{
			name:   "negative - missing url field",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   bytes.NewBuffer([]byte("{}")),
			want: want{
				Status: http.StatusBadRequest,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, tt.body)
			req.Header.Set("Content-Type", "application/json")
			token := createTestToken(t, "test-user-uuid")
			req.Header.Set("Auth", token)
			req.AddCookie(&http.Cookie{
				Name:  "Auth",
				Value: token,
			})
			w := httptest.NewRecorder()
			route.ServeHTTP(w, req)
			assert.Equal(t, tt.want.Status, w.Code)
			if tt.want.HeaderExists == "Content-Type" {
				assert.Equal(t, tt.want.ContentType, w.Header().Get("Content-Type"))
			}
			if tt.want.ResponseHas != "" {
				response := w.Body.String()
				assert.Contains(t, response, tt.want.ResponseHas)
			}
		})
	}
}

// ============================================================
// Тесты для GetURLHandler
// ============================================================

func Test_GetURLHandler(t *testing.T) {
	longURL := "https://sberbank.ru"
	baseURL := "https://short.ru"
	s := newTestStore(t)
	h := NewHandler(s, baseURL, "test-secret-key", nil)

	shortURL, err := s.GetShortURL(context.Background(), longURL, "test-user-uuid")
	require.NoError(t, err)
	assert.NotEmpty(t, shortURL)

	type want struct {
		Status int
		Header string
		Value  string
	}

	tests := []struct {
		name string
		path string
		want want
	}{
		{
			name: "positive",
			path: "/" + shortURL,
			want: want{
				Status: http.StatusTemporaryRedirect,
				Header: "Location",
				Value:  longURL,
			},
		},
		{
			name: "non-existent URL",
			path: "/nonexistent",
			want: want{
				Status: http.StatusNotFound,
			},
		},
		{
			name: "empty short URL",
			path: "/",
			want: want{
				Status: http.StatusNotFound,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			w := httptest.NewRecorder()
			ctx := chi.NewRouteContext()
			id := strings.TrimPrefix(test.path, "/")
			ctx.URLParams.Add("id", id)
			r := req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
			h.GetURLHandler(w, r)
			assert.Equal(t, test.want.Status, w.Code)
			if test.want.Header != "" {
				assert.Equal(t, test.want.Value, w.Header().Get(test.want.Header))
			}
		})
	}
}

// ============================================================
// Тесты для GetUserURLsHandler (НОВЫЕ)
// ============================================================

func Test_GetUserURLsHandler(t *testing.T) {
	baseURL := "https://short.ru"
	s := newTestStore(t)
	h := NewHandler(s, baseURL, "test-secret-key", nil)

	route := chi.NewRouter()
	route.Get("/api/user/urls", h.GetUserURLsHandler)

	// Создаём URL для пользователя
	userID := "test-user-123"
	_, err := s.GetShortURL(context.Background(), "https://example1.com", userID)
	require.NoError(t, err)
	_, err = s.GetShortURL(context.Background(), "https://example2.com", userID)
	require.NoError(t, err)

	t.Run("success - user has URLs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
		token := createTestToken(t, userID)
		req.Header.Set("Auth", token)

		w := httptest.NewRecorder()
		route.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var urls []model.URL
		err := json.Unmarshal(w.Body.Bytes(), &urls)
		require.NoError(t, err)
		assert.Len(t, urls, 2)
	})

	t.Run("no content - user has no URLs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
		token := createTestToken(t, "empty-user")
		req.Header.Set("Auth", token)

		w := httptest.NewRecorder()
		route.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("unauthorized - no token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
		w := httptest.NewRecorder()
		route.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// ============================================================
// Тесты для DeleteURLsHandler (НОВЫЕ)
// ============================================================

func Test_DeleteURLsHandler(t *testing.T) {
	baseURL := "https://short.ru"
	s := newTestStore(t)
	h := NewHandler(s, baseURL, "test-secret-key", nil)

	// Запускаем воркеры для удаления
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.StartDeleter(ctx, 1)

	route := chi.NewRouter()
	route.Delete("/api/user/urls", h.DeleteURLsHandler)

	userID := "test-user-123"
	shortURL, err := s.GetShortURL(context.Background(), "https://delete-me.com", userID)
	require.NoError(t, err)

	t.Run("success - delete URLs", func(t *testing.T) {
		body, _ := json.Marshal([]string{shortURL})
		req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		token := createTestToken(t, userID)
		req.Header.Set("Auth", token)

		w := httptest.NewRecorder()
		route.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)
	})

	t.Run("bad request - empty body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader("[]"))
		req.Header.Set("Content-Type", "application/json")
		token := createTestToken(t, userID)
		req.Header.Set("Auth", token)

		w := httptest.NewRecorder()
		route.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad request - invalid json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader("invalid"))
		req.Header.Set("Content-Type", "application/json")
		token := createTestToken(t, userID)
		req.Header.Set("Auth", token)

		w := httptest.NewRecorder()
		route.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unauthorized - no token", func(t *testing.T) {
		body, _ := json.Marshal([]string{shortURL})
		req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		route.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// ============================================================
// Тесты для ShortenBatchHandler (НОВЫЕ)
// ============================================================

func Test_ShortenBatchHandler(t *testing.T) {
	baseURL := "https://short.ru"
	s := newTestStore(t)
	h := NewDBHandler(s, baseURL)

	route := chi.NewRouter()
	route.Post("/api/shorten/batch", h.ShortenBatchHandler)

	t.Run("success - valid batch", func(t *testing.T) {
		batch := []model.BatchRequestItem{
			{Corr: "1", LongURL: "https://example1.com"},
			{Corr: "2", LongURL: "https://example2.com"},
		}
		body, _ := json.Marshal(batch)

		req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		token := createTestToken(t, "test-user")
		req.Header.Set("Auth", token)

		w := httptest.NewRecorder()
		route.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var results []model.BatchResponseItem
		err := json.Unmarshal(w.Body.Bytes(), &results)
		require.NoError(t, err)
		assert.Len(t, results, 2)
		assert.Equal(t, "1", results[0].Corr)
		assert.Equal(t, "2", results[1].Corr)
		assert.Contains(t, results[0].ShortURL, baseURL)
	})

	t.Run("bad request - empty batch", func(t *testing.T) {
		body, _ := json.Marshal([]model.BatchRequestItem{})
		req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		token := createTestToken(t, "test-user")
		req.Header.Set("Auth", token)

		w := httptest.NewRecorder()
		route.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("bad request - invalid URL in batch", func(t *testing.T) {
		batch := []model.BatchRequestItem{
			{Corr: "1", LongURL: "not-a-valid-url"},
		}
		body, _ := json.Marshal(batch)

		req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		token := createTestToken(t, "test-user")
		req.Header.Set("Auth", token)

		w := httptest.NewRecorder()
		route.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unauthorized - no token", func(t *testing.T) {
		batch := []model.BatchRequestItem{
			{Corr: "1", LongURL: "https://example.com"},
		}
		body, _ := json.Marshal(batch)

		req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		route.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// ============================================================
// Тесты для HandlerGetDbPing (НОВЫЕ)
// ============================================================

// mockPinger - мок для тестирования PingHandler
type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.err
}

func Test_HandlerGetDbPing(t *testing.T) {
	t.Run("success - ping ok", func(t *testing.T) {
		h := NewPingHandler(&mockPinger{err: nil})

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()

		h.HandlerGetDbPing(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("internal error - ping failed", func(t *testing.T) {
		h := NewPingHandler(&mockPinger{err: assert.AnError})

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()

		h.HandlerGetDbPing(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("service unavailable - no pinger", func(t *testing.T) {
		h := NewPingHandler(nil)

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()

		h.HandlerGetDbPing(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}
