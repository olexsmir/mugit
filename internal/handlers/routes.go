package handlers

import (
	"log/slog"
	"net/http"
)

func (h *handlers) index(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]any)
	data["meta"] = h.c.Meta

	w.WriteHeader(http.StatusOK)
	if err := h.t.ExecuteTemplate(w, "index", nil); err != nil {
		slog.Error("index template", "err", err)
	}
}
