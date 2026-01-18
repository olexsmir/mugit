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

// multiplex if request smells like gitprotocol-http(5) passes it  to the git
// http service renders templates.
func (h *handlers) multiplex(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery == "service=git-receive-pack" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("http pushing isn't supported"))
		return
	}

	path := r.PathValue("rest")
	if path == "info/refs" &&
		r.URL.RawQuery == "service=git-upload-pack" &&
		r.Method == "GET" {
		h.infoRefs(w, r)
	} else if path == "git-upload-pack" && r.Method == "POST" {
		h.uploadPack(w, r)
	} else if r.Method == "GET" {
		h.repoIndex(w, r)
	}
}
