package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

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

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {

	fmt.Println("Build version:", buildVersion)
	fmt.Println("Build date:", buildDate)
	fmt.Println("Build commit:", buildCommit)

	// 1. Контекст с сигналами для graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	// 2. Загрузка конфигурации
	cfg := config.New()

	// 3. Инициализация логгера и аутентификации
	logger.InitLogger(cfg.LogLevel)
	auth.Init(cfg.SecretKey)

	// 4. Создание загрузчика данных (БД или файл)
	loader, database := createLoader(ctx, cfg)

	// 5. Создание сервиса хранения
	s, err := store.New(loader)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init store")
	}

	// 6. Создание аудитора
	auditor := initAuditor(cfg)

	// 7. Запуск воркеров для асинхронного удаления URL
	//    Воркеры завершаются через CloseDeleter() после остановки HTTP-сервера.
	s.StartDeleter(4)

	// 8. Создание обработчиков и настройка маршрутов
	h := handler.NewHandler(s, cfg.BaseURL, cfg.SecretKey, auditor)
	route := setupRoutes(h, database, cfg, s)

	// 9. Создание HTTP-сервера
	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: route,
	}

	if cfg.EnableHTTPS {
		log.Info().Str("addr", cfg.ServerAddress).Msg("server started with TLS")

		certificates, err := makeCertificate()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create TLS certificate")
		}
		server.TLSConfig = &tls.Config{
			Certificates: certificates,
		}

		go func() {
			if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Fatal().Err(err).Msg("server stopped unexpectedly")
			}
		}()
	} else {
		log.Info().Str("addr", cfg.ServerAddress).Msg("server started")

		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal().Err(err).Msg("server stopped unexpectedly")
			}
		}()
	}

	// 10. Ожидание сигнала завершения
	<-ctx.Done()
	log.Info().Msg("shutting down server (signal received)")

	// 11. Graceful shutdown: не принимаем новые запросы, ждём завершения активных
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}

	// 12. Останавливаем воркеры удаления.

	s.CloseDeleter()

	// 13. Закрываем хранилище
	if err := s.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close store")
	}

	// 14. Закрываем аудитора
	if err := auditor.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close auditor")
	}

	log.Info().Msg("server stopped gracefully")
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

		// Статистика
		sh := handler.NewStatsHandler(s, cfg)
		route.Get("/api/internal/stats", sh.GetStatsHandler)

		log.Info().Msg("connected to database")
	}

	return route
}

// makeCertificate создаёт self-signed TLS-сертификат для localhost.
func makeCertificate() ([]tls.Certificate, error) {
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization: []string{"Yandex.Praktikum"},
			Country:      []string{"RU"},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	var certPEM bytes.Buffer
	if err = pem.Encode(&certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}); err != nil {
		return nil, err
	}

	var privateKeyPEM bytes.Buffer
	if err = pem.Encode(&privateKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}); err != nil {
		return nil, err
	}

	certPair, err := tls.X509KeyPair(certPEM.Bytes(), privateKeyPEM.Bytes())
	if err != nil {
		return nil, err
	}

	return []tls.Certificate{certPair}, nil
}
