package handler

import (
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"strings"
)

type Storage interface {
	GetShortURL(longURL string) (string, error)
	GetLongURL(shortURL string) (string, error)
}

type Handler struct {
	store   Storage
	baseUrl string
}

func NewHandler(store Storage, baseUrl string) *Handler {
	return &Handler{store: store, baseUrl: baseUrl}
}

func (h *Handler) HandGetURL(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) HandPostFullURL(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	longURL := strings.TrimSpace(string(body))
	if longURL == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(longURL, "http://") && !strings.HasPrefix(longURL, "https://") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	shortURL, err := h.store.GetShortURL(longURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	fullShortURL := h.baseUrl + "/" + shortURL
	_, _ = w.Write([]byte(fullShortURL))
}
