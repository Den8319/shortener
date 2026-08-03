package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Den8319/shortener/internal/auth"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecretKey = "test_secret_key_123"
const expire = time.Hour * 12

func init() {
	auth.Init(testSecretKey)
}

// ============================================================
// Тесты для GetUser
// ============================================================

func TestGetUser(t *testing.T) {
	tests := []struct {
		name string
		auth string
		want string
	}{
		{
			name: "валидный токен — должен вернуть userID",
			auth: func() string {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)),
					},
					User: "user-123",
				})
				signed, _ := token.SignedString([]byte(testSecretKey))
				return signed
			}(),
			want: "user-123",
		},
		{
			name: "невалидная подпись — должен вернуть пустую строку",
			auth: func() string {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)),
					},
					User: "user-123",
				})
				signed, _ := token.SignedString([]byte("another_secret"))
				return signed
			}(),
			want: "",
		},
		{
			name: "просроченный токен — должен вернуть пустую строку",
			auth: func() string {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
					},
					User: "user-123",
				})
				signed, _ := token.SignedString([]byte(testSecretKey))
				return signed
			}(),
			want: "",
		},
		{
			name: "пустой токен — должен вернуть пустую строку",
			auth: "",
			want: "",
		},
		{
			name: "битый формат JWT — должен вернуть пустую строку",
			auth: "not_jwt",
			want: "",
		},
		{
			name: "модифицированный токен — должен отклонить",
			auth: func() string {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)),
					},
					User: "user-123",
				})
				signed, _ := token.SignedString([]byte(testSecretKey))
				return signed + "подделка"
			}(),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := auth.GetUser(tt.auth)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================
// Тесты для Init
// ============================================================

func TestInit(t *testing.T) {
	t.Run("init with valid key", func(t *testing.T) {
		// Init не должна паниковать
		assert.NotPanics(t, func() {
			auth.Init("new-secret-key")
		})

		// Восстанавливаем тестовый ключ
		auth.Init(testSecretKey)
	})

	t.Run("init with empty key", func(t *testing.T) {
		assert.NotPanics(t, func() {
			auth.Init("")
		})

		// Восстанавливаем тестовый ключ
		auth.Init(testSecretKey)
	})
}

// ============================================================
// Тесты для GenerateToken
// ============================================================

func TestGenerateToken(t *testing.T) {
	t.Run("valid token generation", func(t *testing.T) {
		userID := "user-456"
		email := "test@example.com"

		token, err := auth.GenerateToken(userID, email)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Проверяем, что токен валидный
		extractedUser := auth.GetUser(token)
		assert.Equal(t, userID, extractedUser)
	})

	t.Run("different users get different tokens", func(t *testing.T) {
		token1, err := auth.GenerateToken("user-1", "user1@example.com")
		require.NoError(t, err)

		token2, err := auth.GenerateToken("user-2", "user2@example.com")
		require.NoError(t, err)

		assert.NotEqual(t, token1, token2)
		assert.Equal(t, "user-1", auth.GetUser(token1))
		assert.Equal(t, "user-2", auth.GetUser(token2))
	})

	t.Run("generated token can be parsed", func(t *testing.T) {
		userID := "user-789"
		token, err := auth.GenerateToken(userID, "test@example.com")
		require.NoError(t, err)

		// Парсим токен и проверяем claims
		claims := &auth.Claims{}
		parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (any, error) {
			return []byte(testSecretKey), nil
		})
		require.NoError(t, err)
		assert.True(t, parsedToken.Valid)
		assert.Equal(t, userID, claims.User)
		assert.NotNil(t, claims.ExpiresAt)
	})
}

// ============================================================
// Тесты для WithAuth middleware
// ============================================================

func TestWithAuth(t *testing.T) {
	// Простой handler для тестирования
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	t.Run("request without cookie gets new cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		handler := auth.WithAuth(testHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Проверяем, что установлена cookie Auth
		cookies := w.Result().Cookies()
		var authCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "Auth" {
				authCookie = c
				break
			}
		}
		assert.NotNil(t, authCookie, "Auth cookie должна быть установлена")
		assert.NotEmpty(t, authCookie.Value, "Auth cookie не должна быть пустой")
	})

	t.Run("request with valid cookie passes through", func(t *testing.T) {
		// Создаём валидный токен
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)),
			},
			User: "test-user",
		})
		signed, err := token.SignedString([]byte(testSecretKey))
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "Auth",
			Value: signed,
		})
		w := httptest.NewRecorder()

		handler := auth.WithAuth(testHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("request with invalid cookie gets new cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "Auth",
			Value: "invalid-token",
		})
		w := httptest.NewRecorder()

		handler := auth.WithAuth(testHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Должна быть установлена новая cookie
		cookies := w.Result().Cookies()
		var authCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "Auth" {
				authCookie = c
				break
			}
		}
		assert.NotNil(t, authCookie)
		assert.NotEqual(t, "invalid-token", authCookie.Value)
	})

	t.Run("request with expired cookie gets new cookie", func(t *testing.T) {
		// Создаём просроченный токен
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			},
			User: "expired-user",
		})
		signed, err := token.SignedString([]byte(testSecretKey))
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "Auth",
			Value: signed,
		})
		w := httptest.NewRecorder()

		handler := auth.WithAuth(testHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Должна быть установлена новая cookie
		cookies := w.Result().Cookies()
		var authCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "Auth" {
				authCookie = c
				break
			}
		}
		assert.NotNil(t, authCookie)
		assert.NotEqual(t, signed, authCookie.Value)
	})
}
