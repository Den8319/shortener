package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getAuthToken(t *testing.T) {

	tests := []struct {
		name          string
		setupRequest  func() *http.Request
		expectedToken string
	}{
		{
			name: "нет заголовка Auth — пустая строка",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				return req
			},
			expectedToken: "",
		},
		{
			name: "токен в заголовке Auth — возвращается как есть",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("Auth", "some-token-value")
				return req
			},
			expectedToken: "some-token-value",
		},
		{
			name: "кука User без заголовка Auth — игнорируется",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.AddCookie(&http.Cookie{
					Name:  "User",
					Value: "hacker-user-id",
				})
				return req
			},
			expectedToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			got := getAuthToken(req)
			assert.Equal(t, tt.expectedToken, got, "getAuthToken() = %v, want %v", got, tt.expectedToken)
		})
	}
}
