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
	// 1. Загрузка конфигурации
	cfg := config.New()

	// 2. Инициализация логгера и аутентификации
	logger.InitLogger(cfg.LogLevel)
	auth.Init(cfg.SecretKey)

	// 3. Создание контекста (нужен для воркеров удаления)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 4. Создание загрузчика данных (БД или файл)
	loader, database := createLoader(ctx, cfg)

	// 5. Создание сервиса хранения
	s, err := store.New(loader)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init store")
	}
	defer s.Close()

	// 6. Запуск воркеров для асинхронного удаления URL
	s.StartDeleter(ctx, 4)

	// 7. Создание аудитора
	auditor := initAuditor(cfg)
	defer auditor.Close()

	// 8. Создание обработчиков и настройка маршрутов
	h := handler.NewHandler(s, cfg.BaseURL, cfg.SecretKey, auditor)
	route := setupRoutes(h, database, cfg, s)

	// 9. Запуск сервера
	log.Info().Str("addr", cfg.ServerAddress).Msg("server started")
	if err := http.ListenAndServe(cfg.ServerAddress, route); err != nil {
		log.Fatal().Err(err).Msg("server stopped")
	}
}

// createLoader создаёт загрузчик данных (БД или файл)
func createLoader(ctx context.Context, cfg *config.Config) (model.Loader, *db.DB) {
	var loader model.Loader
	var database *db.DB

	if cfg.DatabaseDSN != "" {
		dbInstance, err := db.New(ctx, cfg.DatabaseDSN)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to connect to database")
		}
		if err = dbInstance.Migrate(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to migrate database")
		}
		loader = dbInstance
		database = dbInstance
		log.Info().Msg("using database storage")
	}

	if loader == nil && cfg.FileStoragePath != "" {
		fileloader, err := file.New(cfg.FileStoragePath)
		if err != nil {
			log.Fatal().Err(err).Str("path", cfg.FileStoragePath).Msg("failed to init file storage")
		}
		loader = fileloader
		log.Info().Str("path", cfg.FileStoragePath).Msg("using file storage")
	}

	return loader, database
}

// initAuditor создаёт и настраивает аудитор
func initAuditor(cfg *config.Config) *audit.Auditor {
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

	return auditor
}

// setupRoutes настраивает маршруты роутера
func setupRoutes(h *handler.Handler, database *db.DB, cfg *config.Config, s *store.Store) *chi.Mux {
	route := chi.NewRouter()

	// Middleware
	route.Use(logger.WithLogging)
	route.Use(compress.WithCompression)
	route.Use(auth.WithAuth)
	route.Mount("/debug", middleware.Profiler())

	// Основные маршруты
	route.Post("/", h.ShortenTextHandler)
	route.Post("/api/shorten", h.ShortenJSONHandler)
	route.Get("/{id}", h.GetURLHandler)
	route.Get("/api/user/urls", h.GetUserURLsHandler)
	route.Delete("/api/user/urls", h.DeleteURLsHandler)

	// Маршруты только для БД
	if cfg.DatabaseDSN != "" && database != nil {
		hp := handler.NewPingHandler(database)
		route.Get("/ping", hp.HandlerGetDBPing)

		dbh := handler.NewDBHandler(s, cfg.BaseURL)
		route.Post("/api/shorten/batch", dbh.ShortenBatchHandler)

		log.Info().Msg("connected to database")
	}

	return route
}
