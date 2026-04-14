package compress

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

type compressResponseWriter struct {
	http.ResponseWriter
	gzWriter *gzip.Writer
}

func (w compressResponseWriter) Write(b []byte) (int, error) {
	return w.gzWriter.Write(b)
}

// decompressReader декомпрессирует тело запроса
func decompressReader(r *http.Request) (io.ReadCloser, error) {
	if r.Header.Get("Content-Encoding") == "gzip" {
		log.Info().Msg("Decompressing request body")
		return gzip.NewReader(r.Body)
	}
	return r.Body, nil
}

// newCompressWriter создает сжатый ResponseWriter
func newCompressWriter(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *gzip.Writer, bool) {

	ct := r.Header.Get("Content-Type")

	if !strings.Contains(ct, "application/json") && !strings.Contains(ct, "text/html") {
		return w, nil, false
	}

	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		// если gzip не поддерживается, передаём управление
		// дальше без изменений
		return w, nil, false
	}

	log.Info().Msg("Compressing response")
	gzWriter := gzip.NewWriter(w)

	w.Header().Set("Content-Encoding", "gzip")

	compressWriter := compressResponseWriter{
		ResponseWriter: w,
		gzWriter:       gzWriter,
	}

	return compressWriter, gzWriter, true
}

// WithCompression - middleware для сжатия HTTP-ответов
func WithCompression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info().
			Str("method", r.Method).
			Str("uri", r.URL.RequestURI()).
			Msg("gzip middleware processing request")

		bodyReader, err := decompressReader(r)
		if err != nil {
			log.Error().
				Str("method", r.Method).
				Str("uri", r.URL.RequestURI()).
				Msg("Failed to decompress request body")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer bodyReader.Close()
		r.Body = bodyReader

		responseWriter, gzWriter, compressed := newCompressWriter(w, r)
		if compressed {
			log.Info().
				Str("method", r.Method).
				Str("uri", r.URL.RequestURI()).
				Msg("Response will be compressed")

			defer gzWriter.Close()
			next.ServeHTTP(responseWriter, r)
			return
		}

		log.Info().
			Str("method", r.Method).
			Str("uri", r.URL.RequestURI()).
			Msg("Response will not be compressed")
		next.ServeHTTP(w, r)
	})
}
