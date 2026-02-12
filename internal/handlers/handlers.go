package handlers

import (
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

func InitRoutes(cfg *config.Config) http.Handler {
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
	mux.HandleFunc("GET /{name}/archive/{ref}", h.archiveHandler)


	handler := h.recoverMiddleware(mux)
	return h.loggingMiddleware(handler)
}

func (h *handlers) serveStatic(w http.ResponseWriter, r *http.Request) {
	f := filepath.Clean(r.PathValue("file"))
	// TODO: check if files exists
	http.ServeFileFS(w, r, web.StaticFS, f)
}

func repoNameToPath(name string) string { return name + ".git" }
func getNormalizedName(name string) string {
	return strings.TrimSuffix(name, ".git")
}

var templateFuncs = template.FuncMap{
	"humanizeTime": func(t time.Time) string { return humanize.Time(t) },
	"commitSummary": func(s string) string {
		before, after, found := strings.Cut(s, "\n")
		first := strings.TrimSuffix(before, "\r")
		if !found {
			return first
		}

		if strings.Contains(after, "\n") {
			return first + "..."
		}

		return first
	},
}
