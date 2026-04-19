package main

import (
	"github.com/Den8319/shortener/internal/compress"
	"github.com/Den8319/shortener/internal/config"
	"github.com/Den8319/shortener/internal/handler"
	"github.com/Den8319/shortener/internal/logger"
	"github.com/Den8319/shortener/internal/service/store"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.New()

	logger.InitLogger(cfg.LogLevel)

	s, err := store.NewFileStore(cfg.FileStoragePath)
	if err != nil {
		log.Fatal().Err(err).Str("path", cfg.FileStoragePath).Msg("failed to init file storage")
	}
	defer s.Close()
	log.Info().Str("path", cfg.FileStoragePath).Msg("using file storage")

	route := chi.NewRouter()
	h := handler.NewHandler(s, cfg.BaseURL)

	route.Use(logger.WithLogging)
	route.Use(compress.WithCompression)

	route.Post("/", h.ShortenTextHandler)
	route.Post("/api/shorten", h.ShortenJSONHandler)
	route.Get("/{id}", h.GetURLHandler)

	if err := http.ListenAndServe(cfg.ServerAddress, route); err != nil {
		log.Fatal().Err(err).Msg("server stopped")
	}
}