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
	gzWriter    *gzip.Writer
	wroteHeader bool
	compress    bool
}

func (w *compressResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	ct := w.Header().Get("Content-Type")
	w.compress = strings.Contains(ct, "application/json") || strings.Contains(ct, "text/html")
	if w.compress {
		w.Header().Set("Content-Encoding", "gzip")
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *compressResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.compress {
		return w.gzWriter.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// decompressReader декомпрессирует тело запроса
func decompressReader(r *http.Request) (io.ReadCloser, error) {
	if r.Header.Get("Content-Encoding") == "gzip" {
		log.Info().Msg("Decompressing request body")
		return gzip.NewReader(r.Body)
	}
	return r.Body, nil
}

// newCompressWriter создает сжатый ResponseWriter, если клиент поддерживает gzip
func newCompressWriter(w http.ResponseWriter, r *http.Request) (*compressResponseWriter, *gzip.Writer, bool) {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		return nil, nil, false
	}
	gzWriter := gzip.NewWriter(w)
	cw := &compressResponseWriter{
		ResponseWriter: w,
		gzWriter:       gzWriter,
	}
	return cw, gzWriter, true
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

		cw, gzWriter, ok := newCompressWriter(w, r)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		defer func() {
			if cw.compress {
				gzWriter.Close()
			}
		}()

		next.ServeHTTP(cw, r)
	})
}
