// Package handler содержит HTTP-обработчики для API сокращения URL.
package handler

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/Den8319/shortener/internal/config"
	"github.com/Den8319/shortener/internal/service"
	"github.com/rs/zerolog/log"
)

// StatsResponse представляет ответ эндпоинта статистики.
type StatsResponse struct {
	URLs  int `json:"urls"`
	Users int `json:"users"`
}

// StatsHandler содержит зависимости для обработчика статистики.
type StatsHandler struct {
	service    *service.Service
	trustedNet *net.IPNet
}

// NewStatsHandler создаёт новый StatsHandler.
// trustedNet — распарсенная при загрузке конфигурации доверенная подсеть (может быть nil).
func NewStatsHandler(service *service.Service, cfg *config.Config) *StatsHandler {
	return &StatsHandler{
		service:    service,
		trustedNet: cfg.TrustedNet,
	}
}

// GetStatsHandler возвращает статистику по URL и пользователям.
func (h *StatsHandler) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("GetStatsHandler")

	if h.trustedNet == nil {
		http.Error(w, "Forbidden: trusted subnet not configured", http.StatusForbidden)
		return
	}

	ipStr := r.Header.Get("X-Real-IP")
	if ipStr == "" {
		log.Error().Msg("IP not found in X-Real-IP header")
		http.Error(w, "Forbidden: IP header missing", http.StatusForbidden)
		return
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		log.Error().Str("ip", ipStr).Msg("failed to parse IP")
		http.Error(w, "Forbidden: invalid IP address", http.StatusForbidden)
		return
	}

	if !h.trustedNet.Contains(ip) {
		log.Error().Str("ip", ipStr).Msg("IP not in trusted subnet")
		http.Error(w, "Forbidden: IP not in trusted subnet", http.StatusForbidden)
		return
	}

	stats, err := h.service.Stats(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get stats")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp := StatsResponse{
		URLs:  stats.URLs,
		Users: stats.Users,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error().Err(err).Msg("failed to encode stats response")
	}
}
