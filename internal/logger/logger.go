package logger

import (
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log" // импортируем глобальный логгер
)

type responseData struct {
	status int
	size   int
}

type loggingResponseWriter struct {
	http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
	responseData        *responseData
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	// записываем ответ, используя оригинальный http.ResponseWriter
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size // захватываем размер
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	// записываем код статуса, используя оригинальный http.ResponseWriter
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode // захватываем код статуса
}

// InitLogger настраивает глобальный логгер zerolog.
// Принимает строку уровня (например, "info", "debug").
// Если уровень невалиден, используется уровень по умолчанию (InfoLevel).
func InitLogger(levelStr string) {
	// Парсим уровень
	level, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		level = zerolog.InfoLevel
	}

	logger := zerolog.New(os.Stderr).Level(level).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(level)
	log.Logger = logger

	if err != nil {
		log.Warn().Str("invalid_level", levelStr).Msg("Invalid log level provided, using info level")
	}
}

// WithLogging - middleware для логирования HTTP-запросов и ответов.
// Использует глобальный логгер из пакета `log`.
func WithLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Создаем обертку для ResponseWriter для захвата статуса и размера
		writer := &loggingResponseWriter{
			ResponseWriter: w,
			responseData:   &responseData{status: http.StatusOK},
		}

		// Логируем информацию о входящем запросе
		log.Info().
			Str("method", r.Method).
			Str("uri", r.URL.RequestURI()).
			Msg("incoming request")

		// Передаем управление следующему обработчику
		next.ServeHTTP(writer, r)

		// Логируем информацию об ответе
		log.Info().
			Int("status", writer.responseData.status).
			Int("size", writer.responseData.size).
			Dur("duration", time.Since(start)).
			Msg("request completed")
	})
}
