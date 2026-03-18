package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Den8319/shortener/internal/service/store"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-chi/chi/v5"
)

func TestHandler_HandPostFullURL(t *testing.T) {
	// Создаем мок хранилища
	s := store.New()
	h := NewHandler(s)

	// Создаем тестовые данные
	longURL := "https://sberbank.ru"

	type want struct {
		Status int
		Header string
		Type   string
		}

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		want           want	
		
	}{
		{
			name:           "positive",
			method:         http.MethodPost,
			path:           "/",
			body:           longURL,
			want : want{
			Status: http.StatusCreated,
			Header: "Content-Type",
			Type:   "text/plain",
			},
		},
		{
			name:           "negative wrong method",
			method:         http.MethodGet,
			path:           "/",
			body:           longURL,
			want : want{
			Status: http.StatusMethodNotAllowed,

			},
		},

		{
			name:           "negative wrong path",
			method:         http.MethodPost,
			path:           "/negative",
			body:           longURL,
			want : want{
			Status: http.StatusNotFound,
		},
		
		},
		{
			name:           "negative wrong body",
			method:         http.MethodPost,
			path:           "/",
			body:           "http/sber",
			want : want{
			Status: http.StatusBadRequest,
			},
		},
		{
			name:           "negative empty body",
			method:         http.MethodPost,
			path:           "/",
			body:           "",
			want : want{
			Status: http.StatusBadRequest,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bodyReader := strings.NewReader(test.body)
			
			req := httptest.NewRequest(test.method, test.path, bodyReader)
			w := httptest.NewRecorder()

	
			h.HandPostFullURL(w, req)

	
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

func TestHandler_HandGetURL(t *testing.T) {
	
	route := chi.NewRouter()
	s := store.New()
	h := NewHandler(s)
	
	route.Post("/", h.HandPostFullURL) 
	route.Get("/{id}", h.HandGetURL) 

	
	longURL := "https://sberbank.ru"
	shortURL, err := s.GetShortURL(longURL)
	
	require.NoError(t, err)
	assert.NotEmpty(t, shortURL)

	type want struct {
		Status int
		Header string
		Value  string
	}

	tests := []struct {
		name           string
		path           string
		want           want	
	}{
		{
			name:           "positive",
			path:           "/" + shortURL,
			want : want{
			Status: http.StatusTemporaryRedirect,
			Header: "Location",
			Value:  longURL,
			},
			
		},
		{
			name:           "non-existent URL",
			path:           "/nonexistent",
			want : want{
			Status: http.StatusNotFound,			
			},
		},
		{
			name:           "empty short URL",
			path:           "/",
			want : want{
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

			
			h.HandGetURL(w, r)

	        assert.Equal(t, test.want.Status, w.Code)


			
			if test.want.Header != "" {
				assert.Equal(t, test.want.Value, w.Header().Get(test.want.Header))
			}
		})
	}
}