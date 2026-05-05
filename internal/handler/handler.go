package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Den8319/shortener/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Handler обрабатывает запросы на сокращение URL
type Handler struct {
	store   model.Storage
	baseURL string
}

type PingHandler struct {
	model.Pinger
}



func NewHandler(store model.Storage, baseURL string) *Handler {
	return &Handler{store: store, baseURL: baseURL}
}

func NewPingHandler(p model.Pinger) *PingHandler {
	return &PingHandler{Pinger: p}
}

func (h *PingHandler) HandlerGetDbPing(w http.ResponseWriter, r *http.Request) {
	if h.Pinger == nil {
		log.Warn().Msg("no pinger configured")
		http.Error(w, "Database not configured", http.StatusServiceUnavailable)
		return
	}
	if err := h.Ping(r.Context()); err != nil {
		log.Error().Err(err).Msg("db ping failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}


func (h *Handler) GetURLHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	longURL, err := h.store.GetLongURL(context.TODO(), id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Location", longURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *Handler) ShortenTextHandler(w http.ResponseWriter, r *http.Request) {

	log.Info().Msg("ShortenTextHandler")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	longURL := strings.TrimSpace(string(body))
	if !isValidURL(longURL) {
		log.Warn().Str("bad_url", longURL).Msg("Invalid URL in plain text request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortURL, err := h.store.GetShortURL(context.TODO(), longURL)
	if err != nil {
		if errors.Is(err, model.ErrURLAlreadyExists) {
			fullShortURL, buildErr := url.JoinPath(h.baseURL, shortURL)
			if buildErr != nil {
				log.Error().Err(buildErr).Msg("Failed to build full short URL")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusConflict) // 409
			if _, err := w.Write([]byte(fullShortURL)); err != nil {
				log.Error().Err(err).Msg("Failed to write response")
			}
			return
		}
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
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		log.Warn().Msg("content type not allowed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Debug().Msg("decoding request")
	var req model.Request
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		log.Error().Err(err).Msg("Failed to decoding request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !isValidURL(req.LongURL) {
		log.Warn().Str("bad_url", req.LongURL).Msg("Invalid URL in json request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	shortURL, err := h.store.GetShortURL(context.TODO(), req.LongURL)
	if err != nil {
		if errors.Is(err, model.ErrURLAlreadyExists) {
			fullShortURL, buildErr := url.JoinPath(h.baseURL, shortURL)
			if buildErr != nil {
				log.Error().Err(buildErr).Msg("Failed to build full short URL")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			response := model.Response{
				ShortURL: fullShortURL,
			}

			enc, marshalErr := json.Marshal(response)
			if marshalErr != nil {
				log.Error().Err(marshalErr).Msg("error encoding conflict response")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict) // 409
			if _, err := w.Write(enc); err != nil {
				log.Error().Err(err).Msg("Failed to write response")
			}
			return
		}
		log.Error().Err(err).Str("long_url", req.LongURL).Msg("Failed to generate short URL")
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
		ShortURL: fullShortURL,
	}

	enc, err := json.Marshal(response)
	if err != nil {
		log.Error().Err(err).Msg("error encoding response")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(enc); err != nil {
		log.Error().Err(err).Msg("Failed to write response")
	}
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

	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	if u.Host == "" {
		return false
	}

	return true
}
 