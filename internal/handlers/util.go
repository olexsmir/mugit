package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

func (h *handlers) templ(w http.ResponseWriter, name string, data any) {
	if err := h.t.ExecuteTemplate(w, name, data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("template", "name", name, "err", err)
	}
}

func (h *handlers) write404(w http.ResponseWriter, err error) {
	slog.Info("404", "err", err)
	w.WriteHeader(http.StatusNotFound)
	h.templ(w, "404", nil)
}

func (h *handlers) write500(w http.ResponseWriter, err error) {
	slog.Info("500", "err", err)
	w.WriteHeader(http.StatusInternalServerError)
	h.templ(w, "500", nil)
}

func (h *handlers) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				h.write500(w, fmt.Errorf("panic: %v", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (h *handlers) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := wrapResponseWriter(w)
		next.ServeHTTP(wrapped, r)

		slog.Info("http request",
			"method", r.Method,
			"status", wrapped.status,
			"path", r.URL.Path,
			"latency", time.Since(start).String(),
			"ua", r.UserAgent(),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w}
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}

	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
	rw.wroteHeader = true
}
