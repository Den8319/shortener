// Package handler содержит HTTP-обработчики для API сокращения URL.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Den8319/shortener/internal/model"
	"github.com/Den8319/shortener/internal/service"
	"github.com/rs/zerolog/log"
)

// DBHandler обрабатывает пакетное сокращение URL.
// Предназначен для обработки множества URLs в одном запросе.
type DBHandler struct {
	service *service.Service
	baseURL string
}

// NewDBHandler создаёт новый DBHandler для пакетного сокращения URL.
func NewDBHandler(service *service.Service, baseURL string) *DBHandler {
	return &DBHandler{
		service: service,
		baseURL: baseURL,
	}
}

// ShortenBatchHandler обрабатывает пакетное сокращение URL.
// Принимает массив объектов с длинными URL, возвращает массив с короткими URL в формате JSON.
// При успешном обрабатывании возвращает HTTP 201 Created.
func (h *DBHandler) ShortenBatchHandler(w http.ResponseWriter, r *http.Request) {
	userUUID, err := h.service.Authenticate(getAuthToken(r))
	if err != nil {
		log.Warn().Msg("failed to get user ID")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var req []model.BatchRequestItem
	if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req) == 0 {
		http.Error(w, "Empty batch", http.StatusBadRequest)
		return
	}

	results, err := h.service.ShortenBatch(r.Context(), req, userUUID)
	if err != nil {
		if errors.Is(err, model.ErrInvalidURL) {
			http.Error(w, "Invalid URL in batch", http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to process batch", http.StatusInternalServerError)
		return
	}

	for i := range results {
		results[i].ShortURL = h.baseURL + "/" + results[i].ShortURL
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err = json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
