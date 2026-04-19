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
	return bytes.NewBuffer(buf.Bytes()), nil
}

func Test_WithCompression_RequestDecompression(t *testing.T) {

	originalBody := "https://example.com"

	compressedBody, err := createGzipReader(originalBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", compressedBody)
	req.Header.Set("Content-Encoding", "gzip")

	w := httptest.NewRecorder()

	handler := WithCompression(mockHandler("test", "text/html"))

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func Test_WithCompression_ResponseCompression(t *testing.T) {

	originalResponse := "This is a test response"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	w := httptest.NewRecorder()

	handler := WithCompression(mockHandler(originalResponse, "text/html"))

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))

	var buf bytes.Buffer
	_, err := io.Copy(&buf, w.Body)
	require.NoError(t, err)

	decompressor, err := gzip.NewReader(&buf)
	require.NoError(t, err)
	defer decompressor.Close()

	var decompressed bytes.Buffer
	_, err = io.Copy(&decompressed, decompressor)
	require.NoError(t, err)

	assert.Equal(t, originalResponse, decompressed.String())
}

func Test_WithCompression_UncompressedRequestResponse(t *testing.T) {
	// Тест для случая, когда клиент не поддерживает сжатие
	originalResponse := "This is uncompressed"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// Нет заголовка Accept-Encoding

	w := httptest.NewRecorder()

	handler := WithCompression(mockHandler(originalResponse, "text/html"))

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Content-Encoding"))

	assert.Equal(t, originalResponse, w.Body.String())
}
