package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Den8319/shortener/internal/service/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-chi/chi/v5"
)

func newTestStore(t *testing.T) *store.FileStore {
	t.Helper()
	s, err := store.NewFileStore(t.TempDir() + "/store.json")
	require.NoError(t, err)
	return s
}

func Test_ShortenTextHandler(t *testing.T) {

	// Создаем тестовые данные
	longURL := "https://sberbank.ru"
	baseUrl := "https://sberbank.ru"

	// Создаем мок хранилища
	s := newTestStore(t)
	h := NewHandler(s, baseUrl)

	route := chi.NewRouter()
	route.Post("/", h.ShortenTextHandler)

	type want struct {
		Status int
		Header string
		Type   string
	}

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   want
	}{
		{
			name:   "positive",
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
			name:   "negative wrong method",
			method: http.MethodPut,
			path:   "/",
			body:   longURL,
			want: want{
				Status: http.StatusMethodNotAllowed,
			},
		},
		{
			name:   "negative wrong body",
			method: http.MethodPost,
			path:   "/",
			body:   "http/sber",
			want: want{
				Status: http.StatusBadRequest,
			},
		},
		{
			name:   "negative empty body",
			method: http.MethodPost,
			path:   "/",
			body:   "",
			want: want{
				Status: http.StatusBadRequest,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bodyReader := strings.NewReader(test.body)

			req := httptest.NewRequest(test.method, test.path, bodyReader)
			w := httptest.NewRecorder()

			route.ServeHTTP(w, req)

			assert.Equal(t, test.want.Status, w.Code)

			if test.want.Header != "" {
				assert.Equal(t, test.want.Type, w.Header().Get(test.want.Header))
			}

			if test.want.Status == http.StatusCreated {
				shortURL, err := s.GetShortURL(longURL)
				assert.NoError(t, err)
				assert.NotEmpty(t, shortURL)
			}
		})
	}
}

func Test_ShortenJSONHandler(t *testing.T) {

	// Создаем тестовые данные
	longURL := "https://sberbank.ru"
	baseUrl := "https://short.ru"

	// Создаем мок хранилища
	s := newTestStore(t)
	h := NewHandler(s, baseUrl)

	route := chi.NewRouter()
	route.Post("/api/shorten", h.ShortenJSONHandler)

	// Вспомогательная функция для создания JSON-тела
	jsonBody := func(url string) *bytes.Buffer {
		body, _ := json.Marshal(map[string]string{"url": url})
		return bytes.NewBuffer(body)
	}

	type want struct {
		status       int
		contentType  string
		location     string
		responseHas  string
		headerExists string
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
				status:       http.StatusCreated,
				contentType:  "application/json",
				responseHas:  "https://short.ru",
				headerExists: "Content-Type",
			},
		},
		{
			name:   "negative - wrong method",
			method: http.MethodGet,
			path:   "/api/shorten",
			body:   jsonBody(longURL),
			want: want{
				status: http.StatusMethodNotAllowed,
			},
		},
		{
			name:   "negative - invalid json",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   bytes.NewBuffer([]byte("{invalid json}")),
			want: want{
				status: http.StatusBadRequest,
			},
		},
		{
			name:   "negative - empty url in json",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   jsonBody(""),
			want: want{
				status: http.StatusBadRequest,
			},
		},
		{
			name:   "negative - invalid url format",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   jsonBody("ftp://example.com"),
			want: want{
				status: http.StatusBadRequest,
			},
		},
		{
			name:   "negative - missing url field",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   bytes.NewBuffer([]byte("{}")),
			want: want{
				status: http.StatusBadRequest,
			},
		},
		{
			name:   "negative - wrong content-type",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   jsonBody(longURL),
			want: want{
				status: http.StatusBadRequest,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, tt.body)
			w := httptest.NewRecorder()

			if !strings.Contains(tt.name, "wrong content-type") {
				req.Header.Set("Content-Type", "application/json")
			}

			route.ServeHTTP(w, req)

			assert.Equal(t, tt.want.status, w.Code)

			if tt.want.headerExists == "Content-Type" {
				assert.Equal(t, tt.want.contentType, w.Header().Get("Content-Type"))
			}

			if tt.want.responseHas != "" {
				response := w.Body.String()
				assert.Contains(t, response, tt.want.responseHas)
			}
		})
	}
}

func Test_GetURLHandler(t *testing.T) {

	longURL := "https://sberbank.ru"
	baseUrl := "https://sberbank.ru"

	s := newTestStore(t)
	h := NewHandler(s, baseUrl)

	shortURL, err := s.GetShortURL(longURL)
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
