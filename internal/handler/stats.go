// Package handler содержит HTTP-обработчики для API сокращения URL.
package handler

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/Den8319/shortener/internal/config"
	"github.com/Den8319/shortener/internal/service/store"
	"github.com/rs/zerolog/log"
)

// StatsResponse представляет ответ эндпоинта статистики.
type StatsResponse struct {
	URLs  int `json:"urls"`
	Users int `json:"users"`
}

// StatsHandler содержит зависимости для обработчика статистики.
type StatsHandler struct {
	store *store.Store
	cfg   *config.Config
}

// NewStatsHandler создаёт новый StatsHandler.
func NewStatsHandler(store *store.Store, cfg *config.Config) *StatsHandler {
	return &StatsHandler{
		store: store,
		cfg:   cfg,
	}
}

// GetStatsHandler возвращает статистику по URL и пользователям.
func (h *StatsHandler) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug().Msg("GetStatsHandler")

	
	if h.cfg.TrustedSubnet == "" {
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

	
	_, cidrNet, err := net.ParseCIDR(h.cfg.TrustedSubnet)
	if err != nil {
		log.Error().Err(err).Str("subnet", h.cfg.TrustedSubnet).Msg("failed to parse trusted subnet")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	
	if !cidrNet.Contains(ip) {
		log.Error().Str("ip", ipStr).Str("subnet", h.cfg.TrustedSubnet).Msg("IP not in trusted subnet")
		http.Error(w, "Forbidden: IP not in trusted subnet", http.StatusForbidden)
		return
	}

	
	urls, users, err := h.store.Stats(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get stats")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	
	resp := StatsResponse{
		URLs:  urls,
		Users: users,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Error().Err(err).Msg("failed to encode stats response")
	}
}
