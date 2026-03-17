package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Den8319/shortener/internal/service/store"
)



type Handler struct {
	store store.Storage
}

func NewHandler(store store.Storage) *Handler {
	return &Handler{store: store}
}

// formatShortURL собирает полный короткий URL с учётом схемы и хоста.
func formatShortURL(r *http.Request, id string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s", scheme, r.Host, id)
}


func (h *Handler) HandGetURL(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/")
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
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

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
	fullShortURL := formatShortURL(r, shortURL)
	_, _ = w.Write([]byte(fullShortURL))
}