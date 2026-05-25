package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	fileRepo "github.com/Den8319/shortener/internal/repository/file"
	"github.com/Den8319/shortener/internal/service/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-chi/chi/v5"
)

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

func Test_ShortenTextHandler(t *testing.T) {
	// Тестовые данные
	longURL := "https://sberbank.ru"
	baseURL := "https://short.ru"

	// Создаём новое хранилище для каждого запуска
	s := newTestStore(t)
	h := NewHandler(s, baseURL, "test-secret-key")

	// Маршрутизатор
	route := chi.NewRouter()
	route.Post("/", h.ShortenTextHandler)

	type want  struct {
		Status int
		Header string
		Type   string
	}
	
	// Вспомогательная функция для проверки ответа
	checkResponse := func(t *testing.T, w *httptest.ResponseRecorder, want want ) {
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
			// Подготавливаем тело запроса
			bodyReader := strings.NewReader(tt.body)
			req := httptest.NewRequest(tt.method, tt.path, bodyReader)
			w := httptest.NewRecorder()

			// Выполняем запрос
			route.ServeHTTP(w, req)

			// Проверяем результат
			checkResponse(t, w, tt.want)
		})
	}
}
func Test_ShortenJSONHandler(t *testing.T) {

	// Создаем тестовые данные
	longURL := "https://sberbank.ru"
	baseUrl := "https://short.ru"

	// Создаем мок хранилища
	s := newTestStore(t)
	h := NewHandler(s, baseUrl, "test-secret-key")

	route := chi.NewRouter()
	route.Post("/api/shorten", h.ShortenJSONHandler)

	// Вспомогательная функция для создания JSON-тела
	jsonBody := func(url string) *bytes.Buffer {
		body, _ := json.Marshal(map[string]string{"url": url})
		return bytes.NewBuffer(body)
	}

	type want struct {
		Status       int
		ContentType  string
		Location     string
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
		{
			name:   "negative - wrong content-type",
			method: http.MethodPost,
			path:   "/api/shorten",
			body:   jsonBody(longURL),
			want: want{
				Status: http.StatusBadRequest,
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

func Test_GetURLHandler(t *testing.T) {

	longURL := "https://sberbank.ru"
	baseUrl := "https://short.ru"

	s := newTestStore(t)
	h := NewHandler(s, baseUrl, "test-secret-key")

	shortURL, err := s.GetShortURL(context.Background(), longURL)
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
