package rest

import (
	"log"
	"net/http"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			lrw := NewLoggingResponseWriter(w)
			next.ServeHTTP(lrw, r)

			log.Printf("%s %s - %d (%s)", r.Method, r.URL.Path, lrw.statusCode, http.StatusText(lrw.statusCode))
		},
	)
}
