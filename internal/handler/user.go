package handler

import (
	"strings"
	"encoding/json"
	"github.com/Den8319/shortener/internal/auth"
	"net/http"

	"github.com/rs/zerolog/log"
)

// GetUserURLsHandler возвращает все сокращённые пользователем URL
func (h *Handler) GetUserURLsHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем или создаем идентификатор пользователя
	userID := getUser(w, r)
	if userID == "" {
		log.Warn().Msg("failed to get user ID")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Получаем URL из хранилища
	urls, err := h.store.GetUserURLs(r.Context(), userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("failed to get user URLs")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Если нет URL, возвращаем 204 No Content
	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	for i := range urls {
    if !strings.HasPrefix(urls[i].ShortURL, "http") {
        urls[i].ShortURL = h.baseURL + "/" + urls[i].ShortURL
    }
}

	// Возвращаем список URL
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(urls); err != nil {
		log.Error().Err(err).Msg("failed to encode response")
	}
}
func getUser(w http.ResponseWriter, r *http.Request) string {
	token := r.Header.Get("Auth")
	return auth.GetUser(token)
}
