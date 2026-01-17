package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"

	"olexsmir.xyz/mugit/internal/config"
)

type handlers struct {
	c *config.Config
	t *template.Template
}

func InitRoutes(cfg *config.Config) *http.ServeMux {
	tmpls := template.Must(template.ParseGlob(
		filepath.Join(cfg.Meta.TemplatesDir, "*"),
	))
	h := handlers{cfg, tmpls}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.index)

	return mux
}
