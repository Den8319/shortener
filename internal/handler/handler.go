package handler

import (
	"encoding/json"
	"github.com/Den8319/shortener/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Storage interface {
	GetShortURL(longURL string) (string, error)
	GetLongURL(shortURL string) (string, error)
}

type Handler struct {
	store   Storage
	baseURL string
}

func NewHandler(store Storage, baseURL string) *Handler {
	return &Handler{store: store, baseURL: baseURL}
}

func (h *Handler) GetURLHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	longURL, err := h.store.GetLongURL(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Location", longURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *Handler) ShortenTextHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body) 
	if err != nil {
		log.Error().Err(err).Msg("Failed to read request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodPost {
    log.Debug().Str("method", r.Method).Msg("Method not allowed")
    w.WriteHeader(http.StatusMethodNotAllowed)
    return
}

	longURL := strings.TrimSpace(string(body))
	if !isValidURL(longURL) {
		log.Warn().Str("bad_url", longURL).Msg("Invalid URL in plain text request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortURL, err := h.store.GetShortURL(longURL)
	if err != nil {
		log.Error().Err(err).Str("url", longURL).Msg("Failed to generate short URL")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fullShortURL, err := url.JoinPath(h.baseURL, shortURL)
	if err != nil {
		log.Error().Err(err).Msg("Failed to build full short URL")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write([]byte(fullShortURL)); err != nil {
		log.Error().Err(err).Msg("Failed to write response")
	}
}

func (h *Handler) ShortenJSONHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		log.Warn().Msg("method not allowed")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		log.Warn().Msg("content type not allowed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// десериализуем запрос в структуру модели
	log.Debug().Msg("decoding request")
	var req model.Request
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		log.Error().Err(err).Msg("Failed to decoding request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !isValidURL(req.LongUrl) {
		log.Warn().Str("bad_url", req.LongUrl).Msg("Invalid URL in json request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	shortURL, err := h.store.GetShortURL(req.LongUrl)
	if err != nil {
		log.Error().Err(err).Str("long_url", req.LongUrl).Msg("Failed to generate short URL")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fullShortURL, err := url.JoinPath(h.baseURL, shortURL)
	if err != nil {
		log.Error().Err(err).Msg("Failed to build full short URL")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	response := model.Response{
		ShortUrl: fullShortURL,
	}

	enc, err := json.Marshal(response)
	if err != nil {
		log.Error().Err(err).Msg("error encoding response")
		w.WriteHeader(http.StatusBadRequest)
		return

	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(enc)
	log.Debug().Msg(w.Header().Get("Content-Type"))
	log.Debug().Msg("sending HTTP 201 response")
}


func isValidURL(rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return false
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	// Проверяем схему
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// Хост обязателен
	if u.Host == "" {
		return false
	}

	return true
}
