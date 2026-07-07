package auth_test

import (
	"testing"
	"time"

	"github.com/Den8319/shortener/internal/auth"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
)

const expire = time.Hour * 12

func TestGetUser(t *testing.T) {

	auth.Init("test_secret_key_123")

	tests := []struct {
		name string // описание теста
		auth string // значение токена (Auth)
		want string // ожидаемый userID
	}{
		{
			name: "валидный токен — должен вернуть userID",
			auth: func() string {
				// Создаём валидный JWT
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)),
					},
					User: "user-123",
				})
				signed, _ := token.SignedString([]byte("test_secret_key_123"))
				return signed
			}(),
			want: "user-123",
		},
		{
			name: "невалидная подпись — должен вернуть пустую строку",
			auth: func() string {
				// Подписан другим ключом
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
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // просрочен
					},
					User: "user-123",
				})
				signed, _ := token.SignedString([]byte("test_secret_key_123"))
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
			name: "битой формат JWT — должен вернуть пустую строку",
			auth: "not_jwt",
			want: "",
		},
		{
			name: "модифицированный токен — изменён payload — должен отклонить",
			auth: func() string {
				// Токен валиден по структуре, но мы его подделываем
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)),
					},
					User: "user-123",
				})
				signed, _ := token.SignedString([]byte("test_secret_key_123"))
				return signed + "подделка"
			}(),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := auth.GetUser(tt.auth)
			assert.Equal(t, tt.want, got, "GetUser() = %v, want %v", got, tt.want)
		})
	}
}
