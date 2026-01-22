package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"olexsmir.xyz/mugit/internal/config"
	"olexsmir.xyz/mugit/internal/humanize"
	"olexsmir.xyz/mugit/web"
)

type handlers struct {
	c *config.Config
	t *template.Template
}

func InitRoutes(cfg *config.Config) *http.ServeMux {
	tmpls := template.Must(template.New("").
		Funcs(templateFuncs).
		ParseFS(web.TemplatesFS, "*"))
	h := handlers{cfg, tmpls}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.indexHandler)
	mux.HandleFunc("GET /static/{file}", h.serveStatic)
	mux.HandleFunc("GET /{name}", h.multiplex)
	mux.HandleFunc("POST /{name}", h.multiplex)
	mux.HandleFunc("GET /{name}/{rest...}", h.multiplex)
	mux.HandleFunc("POST /{name}/{rest...}", h.multiplex)
	mux.HandleFunc("GET /{name}/tree/{ref}/{rest...}", h.repoTreeHandler)
	mux.HandleFunc("GET /{name}/blob/{ref}/{rest...}", h.fileContentsHandler)
	mux.HandleFunc("GET /{name}/log/{ref}", h.logHandler)
	mux.HandleFunc("GET /{name}/commit/{ref}", h.commitHandler)
	mux.HandleFunc("GET /{name}/refs/{$}", h.refsHandler)
	return mux
}

func (h *handlers) serveStatic(w http.ResponseWriter, r *http.Request) {
	f := filepath.Clean(r.PathValue("file"))
	// TODO: check if files exists
	http.ServeFileFS(w, r, web.StaticFS, f)
}

var templateFuncs = template.FuncMap{
	"commitSummary": func(v any) string {
		s := fmt.Sprint(v)
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = strings.TrimSuffix(s[:i], "\r")
			return s + "..."
		}
		return strings.TrimSuffix(s, "\r")
	},
	"humanTime": func(t time.Time) string {
		return humanize.Time(t)
	},
}
