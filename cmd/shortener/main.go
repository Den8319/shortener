package main

import (
	"context"
	"net/http"

	"github.com/Den8319/shortener/internal/audit"
	"github.com/Den8319/shortener/internal/auth"
	"github.com/Den8319/shortener/internal/compress"
	"github.com/Den8319/shortener/internal/config"
	"github.com/Den8319/shortener/internal/handler"
	"github.com/Den8319/shortener/internal/logger"
	"github.com/Den8319/shortener/internal/model"
	"github.com/Den8319/shortener/internal/repository/db"
	"github.com/Den8319/shortener/internal/repository/file"
	"github.com/Den8319/shortener/internal/service/store"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.New()

	logger.InitLogger(cfg.LogLevel)
	auth.Init(cfg.SecretKey)

	var loader model.Loader
	var database *db.DB
	if cfg.DatabaseDSN != "" {
		dbInstance, err := db.New(context.Background(), cfg.DatabaseDSN)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect to database")
		}
		if err = dbInstance.Migrate(context.Background()); err != nil {
			log.Fatal().Err(err).Msg("failed to migrate database")
		}
		loader = dbInstance
		log.Info().Msg("using database storage")
		database = dbInstance
	}

	if loader == nil && cfg.FileStoragePath != "" {
		fileloader, err := file.New(cfg.FileStoragePath)
		if err != nil {
			log.Fatal().Err(err).Str("path", cfg.FileStoragePath).Msg("failed to init file storage")
		}
		loader = fileloader
		log.Info().Str("path", cfg.FileStoragePath).Msg("using file storage")
	}

	s, err := store.New(loader)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init store")
	}
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запуск worker'ов для асинхронного удаления URL
	s.StartDeleter(ctx, 4)

	// Создание auditor для аудита запросов
	auditor := audit.NewAuditor()
	if cfg.AuditFile != "" {
		fileObserver, err := audit.NewFileObserver(cfg.AuditFile)
		if err != nil {
			log.Fatal().Err(err).Str("audit_file", cfg.AuditFile).Msg("failed to init file audit observer")
		}
		auditor.Register(fileObserver)
		log.Info().Str("audit_file", cfg.AuditFile).Msg("file audit observer added")
	}
	if cfg.AuditURL != "" {
		httpObserver := audit.NewHTTPObserver(cfg.AuditURL)
		auditor.Register(httpObserver)
		log.Info().Str("audit_url", cfg.AuditURL).Msg("HTTP audit observer added")
	}

	h := handler.NewHandler(s, cfg.BaseURL, cfg.SecretKey, auditor)

	route := chi.NewRouter()
	route.Use(logger.WithLogging)
	route.Use(compress.WithCompression)
	route.Use(auth.WithAuth)

	route.Mount("/debug", middleware.Profiler())

	route.Post("/", h.ShortenTextHandler)
	route.Post("/api/shorten", h.ShortenJSONHandler)
	route.Get("/{id}", h.GetURLHandler)
	route.Get("/api/user/urls", h.GetUserURLsHandler)
	route.Delete("/api/user/urls", h.DeleteURLsHandler)

	if cfg.DatabaseDSN != "" {
		if database != nil {
			hp := handler.NewPingHandler(database)
			route.Get("/ping", hp.HandlerGetDbPing)
			dbh := handler.NewDBHandler(s, cfg.BaseURL)
			route.Post("/api/shorten/batch", dbh.ShortenBatchHandler)
			defer database.Close()
			log.Info().Msg("connected to database")
		}
	}

	if err := http.ListenAndServe(cfg.ServerAddress, route); err != nil {
		log.Fatal().Err(err).Msg("server stopped")
	}
}
