package handlers

import (
	"log/slog"
	"net/http"
)

func (h *handlers) write404(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	if err := h.t.ExecuteTemplate(w, "404", nil); err != nil {
		slog.Error("404 template", "err", err)
	}
}

func (h *handlers) write500(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	if err := h.t.ExecuteTemplate(w, "500", nil); err != nil {
		slog.Error("500 template", "err", err)
	}
}
