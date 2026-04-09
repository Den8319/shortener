package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHandler создает тестовый обработчик, который возвращает фиксированный ответ
func mockHandler(content string, contentType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(content))
	})
}

// createGzipReader создает сжатый reader с переданными данными
func createGzipReader(data string) (io.Reader, error) {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	_, err := gzWriter.Write([]byte(data))
	if err != nil {
		return nil, err
	}
	if err := gzWriter.Close(); err != nil {
		return nil, err
	}
	return bytes.NewBuffer(buf.Bytes()), nil}

func Test_WithCompression_RequestDecompression(t *testing.T) {
	// Создаем тело запроса
	originalBody := "https://example.com"
	
	// Сжимаем тело запроса
	compressedBody, err := createGzipReader(originalBody)
	require.NoError(t, err)
	
	// Создаем запрос с сжатым телом
	req := httptest.NewRequest(http.MethodPost, "/", compressedBody)
	req.Header.Set("Content-Encoding", "gzip")
	
	// Создаем ResponseRecorder для записи ответа
	w := httptest.NewRecorder()
	
	// Создаем цепочку middleware и обработчик
	handler := WithCompression(mockHandler("test","text/html"))
	
	// Вызываем обработчик
	handler.ServeHTTP(w, req)
	
	// Проверяем, что запрос был успешно обработан
	assert.Equal(t, http.StatusOK, w.Code)
}

func Test_WithCompression_ResponseCompression(t *testing.T) {
	// Оригинальный ответ
	originalResponse := "This is a test response"
	
	// Создаем запрос, который поддерживает gzip
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	
	// Создаем ResponseRecorder
	w := httptest.NewRecorder()
	
	// Создаем цепочку middleware и обработчик
	handler := WithCompression(mockHandler(originalResponse,"text/html"))
	
	// Вызываем обработчик
	handler.ServeHTTP(w, req)
	
	// Проверяем статус
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Проверяем заголовки
	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
	
	
	// Проверяем, что тело ответа сжато
	var buf bytes.Buffer
	_, err := io.Copy(&buf, w.Body)
	require.NoError(t, err)
	
	// Декомпрессируем ответ для проверки содержимого
	decompressor, err := gzip.NewReader(&buf)
	require.NoError(t, err)
	defer decompressor.Close()
	
	var decompressed bytes.Buffer
	_, err = io.Copy(&decompressed, decompressor)
	require.NoError(t, err)
	
	// Проверяем, что декомпрессированное содержимое совпадает с оригиналом
	assert.Equal(t, originalResponse, decompressed.String())
}

func Test_WithCompression_UncompressedRequestResponse(t *testing.T) {
	// Тест для случая, когда клиент не поддерживает сжатие
	originalResponse := "This is uncompressed"
	
	// Запрос без поддержки gzip
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Нет заголовка Accept-Encoding
	
	w := httptest.NewRecorder()
	
	// Создаем цепочку middleware и обработчик
	handler := WithCompression(mockHandler(originalResponse,"text/html"))
	
	// Вызываем обработчик
	handler.ServeHTTP(w, req)
	
	// Проверяем, что ответ не сжат
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Content-Encoding"))
	assert.Empty(t, w.Header().Get("Vary"))
	
	// Проверяем содержимое
	assert.Equal(t, originalResponse, w.Body.String())
}
