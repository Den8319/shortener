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
	"github.com/Den8319/shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// Handler обрабатывает HTTP-запросы на сокращение URL, получение длинных URL по коротким и аудит операций.
// Содержит обработчики для получения длинных URL по коротким и для сокращения URL в текстовом и JSON форматах.
type Handler struct {
	service *service.Service
	baseURL string
	auditor *audit.Auditor
}

// PingHandler — обработчик для проверки доступности базы данных.
type PingHandler struct {
	model.Pinger
}

// NewHandler создаёт новый Handler с указанным сервисом, базовым URL и аудитором.
func NewHandler(service *service.Service, baseURL string, auditor *audit.Auditor) *Handler {
	return &Handler{service: service, baseURL: baseURL, auditor: auditor}
}

// NewPingHandler создаёт новый PingHandler для проверки доступности базы данных.
func NewPingHandler(p model.Pinger) *PingHandler {
	return &PingHandler{Pinger: p}
}

// HandlerGetDBPing проверяет доступность базы данных, выполняя ping-запрос.
// Возвращает HTTP 200 OK при успешном подключении, HTTP 503 Service Unavailable если база не настроена,
// и HTTP 500 Internal Server Error при ошибке подключения.
func (h *PingHandler) HandlerGetDBPing(w http.ResponseWriter, r *http.Request) {
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

	longURL, err := h.service.Expand(r.Context(), id)
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

	// Аудит опционален — пользователь не обязан быть авторизован для перехода.
	userUUID, _ := h.service.Authenticate(getAuthToken(r))
	h.logAudit("follow", userUUID, longURL)
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

	userUUID, err := h.service.Authenticate(getAuthToken(r))
	if err != nil {
		log.Warn().Msg("failed to get user ID")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	result, err := h.service.Shorten(r.Context(), longURL, userUUID)
	if err != nil {
		if errors.Is(err, model.ErrInvalidURL) {
			log.Warn().Str("bad_url", longURL).Msg("Invalid URL in plain text request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Error().Err(err).Str("url", longURL).Msg("Failed to generate short URL")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fullShortURL, err := url.JoinPath(h.baseURL, result.ShortURL)
	if err != nil {
		log.Error().Err(err).Msg("Failed to build full short URL")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	statusCode := http.StatusCreated
	if !result.Created {
		statusCode = http.StatusConflict
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(statusCode)
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

	userUUID, err := h.service.Authenticate(getAuthToken(r))
	if err != nil {
		log.Warn().Msg("failed to get user ID")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	result, err := h.service.Shorten(r.Context(), req.LongURL, userUUID)
	if err != nil {
		if errors.Is(err, model.ErrInvalidURL) {
			log.Warn().Str("bad_url", req.LongURL).Msg("Invalid URL in json request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.Error().Err(err).Str("long_url", req.LongURL).Msg("Failed to generate short URL")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fullShortURL, err := url.JoinPath(h.baseURL, result.ShortURL)
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

	statusCode := http.StatusCreated
	if !result.Created {
		statusCode = http.StatusConflict
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if _, err := w.Write(enc); err != nil {
		log.Error().Err(err).Msg("Failed to write response")
	}

	h.logAudit("shorten", userUUID, req.LongURL)
	log.Debug().Msg("sending HTTP response")
}

// logAudit асинхронно отправляет событие аудита о действии пользователя (сокращение URL или переход по короткой ссылке).
// Выполняется в отдельной горутине, чтобы не блокировать основной поток обработки запроса.
func (h *Handler) logAudit(action, userID, url string) {
	if h.auditor == nil {
		return
	}

	go func(act, usr, u string) {
		event := audit.AuditEvent{
			Timestamp: time.Now().Unix(),
			Action:    act,
			UserID:    usr,
			URL:       u,
		}
		h.auditor.Notify(event)
	}(action, userID, url)
}
