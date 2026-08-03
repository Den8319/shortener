// Package handler содержит HTTP-обработчики для API сокращения URL.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Den8319/shortener/internal/auth"
	"github.com/Den8319/shortener/internal/model"
	"github.com/rs/zerolog/log"
)

// GetUserURLsHandler возвращает все сокращённые пользователем URL в формате JSON.
// Возвращает HTTP 200 OK со списком URL, HTTP 204 No Content если URL нет,
// и HTTP 401 Unauthorized при отсутствии идентификатора пользователя.
func (h *Handler) GetUserURLsHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("GetUserURLsHandler")
	// Получаем или создаем идентификатор пользователя
	userID := getUser(r)
	if userID == "" {
		log.Warn().Msg("failed to get user ID")
		w.WriteHeader(http.StatusUnauthorized)
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

	fullURLs := make([]model.URL, len(urls))
	for i, u := range urls {
		fullURLs[i] = model.URL{
			ShortURL: fmt.Sprintf("%s/%s", h.baseURL, u.ShortURL),
			LongURL:  u.LongURL,
		}
	}

	// Возвращаем список URL
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(fullURLs); err != nil {
		log.Error().Err(err).Msg("failed to encode response")
	}
}

// DeleteURLsHandler асинхронно удаляет URL пользователя.
// Принимает массив идентификаторов коротких URL, возвращает HTTP 202 Accepted без ожидания завершения операции.
func (h *Handler) DeleteURLsHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем идентификатор пользователя
	userID := getUser(r)
	if userID == "" {
		log.Warn().Msg("failed to get user ID")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Читаем тело запроса
	var request []string
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&request); err != nil {
		log.Error().Err(err).Msg("Failed to parse JSON")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(request) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Асинхронно удаляем URL (gorutina запускается внутри DeleteURLs для каждой ссылки)
	// Возвращаем 202 Accepted сразу, без ожидания результата
	err := h.store.DeleteURLs(r.Context(), request, userID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("failed to delete URLs")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// getUser извлекает идентификатор пользователя из заголовка Auth HTTP-запроса.
// Использует токен аутентификации для получения UUID пользователя.
func getUser(r *http.Request) string {

	authCookie := r.Header.Get("Auth")

	return auth.GetUser(authCookie)
}
