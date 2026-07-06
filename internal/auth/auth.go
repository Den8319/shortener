package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	uuid "github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const expire = time.Hour * 12

var secretKey []byte // Хранит ключ из конфига

func Init(key string) {
	secretKey = []byte(key)
}

type Claims struct {
	jwt.RegisteredClaims
	User string
}

func WithAuth(h http.Handler) http.Handler {
	log.Info().Msg("WithAuth started")
	authFn := func(w http.ResponseWriter, r *http.Request) {

		auth, err := r.Cookie("Auth")
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				newCookie(w, r)
				h.ServeHTTP(w, r)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		r.Header.Set("Auth", auth.Value)

		user := GetUser(auth.Value)
		log.Info().Str("user", user).Msg("WithAuth get user")
		if user == "" {
			newCookie(w, r)
		} else {
			http.SetCookie(w, &http.Cookie{
				Name:    "User",
				Value:   user,
				Path:    "/",
				Expires: time.Now().Add(expire),
			})
		}
		h.ServeHTTP(w, r)

	}
	return http.HandlerFunc(authFn)
}

func newCookie(w http.ResponseWriter, r *http.Request) {
	user := uuid.New().String()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expire)),
		},
		User: user,
	})

	auth, err := token.SignedString(secretKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "Auth",
		Value:    auth,
		Path:     "/",
		Expires:  time.Now().Add(expire),
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:    "User",
		Value:   user,
		Path:    "/",
		Expires: time.Now().Add(expire),
	})
	r.Header.Set("Auth", auth)
}

func GetUser(auth string) string {
	claims := &Claims{}
	if token, err := jwt.ParseWithClaims(auth, claims, func(token *jwt.Token) (any, error) {
		return secretKey, nil
	}); err != nil || !token.Valid {
		return ""
	}
	log.Info().Str("claims.User", claims.User).Msg("WithAuth get user")
	return claims.User
}

const TokenExpire = 24 * time.Hour

// GenerateToken создаёт JWT-токен для пользователя
func GenerateToken(userID, email string) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExpire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
		User: userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
