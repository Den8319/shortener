package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Den8319/shortener/internal/auth"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

func Test_getUser_Security_Fixed(t *testing.T) {

	auth.Init("secret-key-test-12345")

	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectedUserID string
	}{
		{
			name: "нет куки Auth - должен вернуть пустую строку",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				return req
			},
			expectedUserID: "",
		},
		{
			name: "недействительный токен в куке Auth - должен вернуть пустую строку",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.AddCookie(&http.Cookie{
					Name:  "Auth",
					Value: "invalid-token",
				})
				return req
			},
			expectedUserID: "",
		},
		{
			name: "поддельная кука User без валидного Auth - должен игнорировать User и вернуть пустую строку",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)

				req.AddCookie(&http.Cookie{
					Name:  "User",
					Value: "hacker-user-id", //  подделка
				})
				return req
			},
			expectedUserID: "", // Кука User должна игнорироваться
		},

		{
			name: "просроченный токен в куке Auth - должен вернуть пустую строку",
			setupRequest: func() *http.Request {

				claims := auth.Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
					},
					User: "user-456",
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				signedToken, _ := token.SignedString([]byte("secret-key-test-12345"))

				req := httptest.NewRequest("GET", "/", nil)
				req.AddCookie(&http.Cookie{
					Name:  "Auth",
					Value: signedToken,
				})
				return req
			},
			expectedUserID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			got := getUser(req)
			assert.Equal(t, tt.expectedUserID, got, "getUser() = %v, want %v", got, tt.expectedUserID)
		})
	}
}
