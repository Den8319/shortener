package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Den8319/shortener/internal/audit"
	"github.com/Den8319/shortener/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Handler обрабатывает HTTP-запросы на сокращение URL, получение длинных URL по коротким и аудит операций.
// Содержит обработчики для получения длинных URL по коротким и для сокращения URL в текстовом и JSON форматах.
type Handler struct {
	store     model.Storage
	baseURL   string
	secretKey string
	auditor   *audit.Auditor
}

type PingHandler struct {
	model.Pinger
}

func NewHandler(store model.Storage, baseURL string, secretKey string, auditor *audit.Auditor) *Handler {
	return &Handler{store: store, baseURL: baseURL, secretKey: secretKey, auditor: auditor}
}

func NewPingHandler(p model.Pinger) *PingHandler {
	return &PingHandler{Pinger: p}
}

// HandlerGetDbPing проверяет доступность базы данных, выполняя ping-запрос.
// Возвращает HTTP 200 OK при успешном подключении, HTTP 503 Service Unavailable если база не настроена,
// и HTTP 500 Internal Server Error при ошибке подключения.
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

// GetURLHandler обрабатывает запросы на получение длинного URL по его короткому идентификатору.
// Возвращает HTTP 307 Temporary Redirect с заголовком Location, указывающим на исходный URL.
// При удалённой ссылке возвращает HTTP 410 Gone, а при отсутствии — HTTP 404 Not Found.
func (h *Handler) GetURLHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	longURL, err := h.store.GetLongURL(r.Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrURLDeleted) {
			w.WriteHeader(http.StatusGone)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Location", longURL)
	w.WriteHeader(http.StatusTemporaryRedirect)

	h.logAudit("follow", getUser(r), longURL)
}

// ShortenTextHandler обрабатывает сокращение URL из текстового запроса.
// Принимает длинный URL в теле запроса, возвращает полный короткий URL в текстовом формате.
// При существующей ссылке возвращает HTTP 409 Conflict с уже существующим коротким URL,
// при успешном сокращении — HTTP 201 Created.
func (h *Handler) ShortenTextHandler(w http.ResponseWriter, r *http.Request) {
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

	userUUID := getUser(r)
	if userUUID == "" {
		log.Warn().Msg("failed to get user ID")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	shortURL, err := h.store.GetShortURL(r.Context(), longURL, userUUID)
	if err != nil {
		if errors.Is(err, model.ErrURLAlreadyExists) {
			fullShortURL, buildErr := url.JoinPath(h.baseURL, shortURL)
			if buildErr != nil {
				log.Error().Err(buildErr).Msg("Failed to build full short URL")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(fullShortURL))

			h.logAudit("shorten", userUUID, longURL)
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
	w.Write([]byte(fullShortURL))

	h.logAudit("shorten", userUUID, longURL)
}

// ShortenJSONHandler обрабатывает сокращение URL из JSON-запроса.
// Принимает объект с полем LongURL в JSON-формате, возвращает объект с полем ShortURL в JSON-формате.
// При существующей ссылке возвращает HTTP 409 Conflict с уже существующим коротким URL,
// при успешном сокращении — HTTP 201 Created.
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

	userUUID := getUser(r)
	log.Info().Str("userUUID", userUUID).Msg("ShortenJSONHandler")
	if userUUID == "" {
		log.Warn().Msg("failed to get user ID")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	shortURL, err := h.store.GetShortURL(r.Context(), req.LongURL, userUUID)
	if err != nil {
		if errors.Is(err, model.ErrURLAlreadyExists) {
			fullShortURL, err := url.JoinPath(h.baseURL, shortURL)
			if err != nil {
				log.Error().Err(err).Msg("Failed to build full short URL")
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

			h.logAudit("shorten", userUUID, req.LongURL)
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

	h.logAudit("shorten", userUUID, req.LongURL)
	log.Debug().Msg("sending HTTP 201 response")
}

// isValidURL проверяет, является ли строка корректным HTTP или HTTPS URL.
// Возвращает true, если URL валиден (имеет схему http/https и хост), иначе false.
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

// logAudit асинхронно отправляет событие аудита о действии пользователя (сокращение URL или переход по короткой ссылке).
// Выполняется в отдельной горутине, чтобы не блокировать основной поток обработки запроса.
func (h *Handler) logAudit(action, userID, url string) {
	if h.auditor == nil {
		return
	}

	go func(act, usr, u string) {
		event := audit.AuditEvent{
			Ts:     time.Now().Unix(),
			Action: act,
			UserID: usr,
			URL:    u,
		}
		h.auditor.Notify(event)
	}(action, userID, url)
}
