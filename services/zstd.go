package services

import (
	"net/http"
	"strings"

	"github.com/klauspost/compress/zstd"
)

type zstdResponseWriter struct {
	http.ResponseWriter
	encoder *zstd.Encoder
}

func (w *zstdResponseWriter) Write(b []byte) (int, error) {
	return w.encoder.Write(b)
}

func ZstdMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Only compress if client explicitly accepts zstd
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "zstd") {
			next.ServeHTTP(w, r)
			return
		}

		encoder, err := zstd.NewWriter(w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer encoder.Close()

		w.Header().Set("Content-Encoding", "zstd")

		zw := &zstdResponseWriter{
			ResponseWriter: w,
			encoder:        encoder,
		}

		next.ServeHTTP(zw, r)
	})
}
