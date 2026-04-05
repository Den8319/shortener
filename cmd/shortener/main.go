package main

import (
	"github.com/Den8319/shortener/internal/config"
	"github.com/Den8319/shortener/internal/handler"
	"github.com/Den8319/shortener/internal/logger"
	"github.com/Den8319/shortener/internal/service/store"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := config.New()

	// Инициализируем глобальный логгер
	logger.InitLogger(cfg.LogLevel)

	route := chi.NewRouter()
	s := store.New()
	h := handler.NewHandler(s, cfg.BaseURL)

	// Используем middleware, который сам использует глобальный логгер
	route.Use(logger.WithLogging)

	route.Post("/", h.ShortenTextHandler)
	route.Post("/api/shorten", h.ShortenJSONHandler)
	route.Get("/{id}", h.GetURLHandler)

	err := http.ListenAndServe(cfg.ServerAddress, route)
	if err != nil {
		panic(err)
	}
}
