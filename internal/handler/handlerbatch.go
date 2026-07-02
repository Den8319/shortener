package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	

	"github.com/Den8319/shortener/internal/model"
	"github.com/rs/zerolog/log"
)

type DBHandler struct {
	store   model.Storage
	baseURL string
}

func NewDBHandler(store model.Storage, baseURL string) *DBHandler {
	return &DBHandler{
		store:   store,
		baseURL: baseURL,
	}
}

func (h *DBHandler) ShortenBatchHandler(w http.ResponseWriter, r *http.Request) {
	// Тело запроса уже распаковано middleware

	userUUID := getUser(r)
	log.Info().Str("userUUID", userUUID).Msg("ShortenBatchHandler")
	if userUUID == "" {
		log.Warn().Msg("failed to get user ID")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var req []model.BatchRequestItem
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req) == 0 {
		http.Error(w, "Empty batch", http.StatusBadRequest)
		return
	}

	for _, item := range req {
		if _, err := url.ParseRequestURI(item.LongURL); err != nil {
			http.Error(w, "Invalid URL: "+item.LongURL, http.StatusBadRequest)
			return
		}
	}

	results, err := h.store.GetShortList(r.Context(), req, userUUID)
	if err != nil {
		http.Error(w, "Failed to process batch", http.StatusInternalServerError)
		return
	}

	for i := range results {
		results[i].ShortURL = h.baseURL + "/" + results[i].ShortURL
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
