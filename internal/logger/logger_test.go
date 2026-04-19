package logger

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestWithLogging(t *testing.T) {
	// 1. Подготавливаем буфер для перехвата логов
	buf := &bytes.Buffer{}

	// Сохраняем старый логгер и восстанавливаем его после теста
	oldLogger := log.Logger
	defer func() { log.Logger = oldLogger }()

	// Настраиваем глобальный логгер на запись в наш буфер (в формате JSON для парсинга)
	log.Logger = zerolog.New(buf)

	// 2. Создаем тестовый обработчик, который имитирует логику приложения
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated) // Код 201
		w.Write([]byte("hello"))          // Тело длиной 5 байт
	})

	// Оборачиваем его в ваше middleware
	handlerToTest := WithLogging(testHandler)

	// 3. Выполняем фиктивный запрос
	req := httptest.NewRequest(http.MethodGet, "/test-uri", nil)
	rec := httptest.NewRecorder()
	handlerToTest.ServeHTTP(rec, req)

	// 4. Проверяем результаты
	output := buf.String()

	// Проверяем логи входящего запроса
	assert.Contains(t, output, `"method":"GET"`)
	assert.Contains(t, output, `"uri":"/test-uri"`)
	assert.Contains(t, output, "incoming request")

	// Проверяем логи завершения запроса (захваченные данные)
	assert.Contains(t, output, `"status":201`)
	assert.Contains(t, output, `"size":5`)
	assert.Contains(t, output, "request completed")
}

func TestInitLogger(t *testing.T) {
	// Проверяем, что некорректный уровень не ломает приложение и ставит Info
	InitLogger("invalid-level")
	assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())

	// Проверяем установку конкретного уровня
	InitLogger("debug")
	assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
}
