package handlers

import (
	"log/slog"
	"net/http"
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
