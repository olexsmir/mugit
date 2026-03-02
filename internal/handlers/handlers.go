package handlers

import (
	"html/template"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"olexsmir.xyz/mugit/internal/cache"
	"olexsmir.xyz/mugit/internal/config"
	"olexsmir.xyz/mugit/internal/git"
	"olexsmir.xyz/mugit/internal/humanize"
	"olexsmir.xyz/mugit/web"
)

type handlers struct {
	c *config.Config
	t *template.Template

	repoListCache cache.Cacher[[]repoList]
	readmeCache   cache.Cacher[template.HTML]
	diffCache     cache.Cacher[*git.NiceDiff]
}

func InitRoutes(cfg *config.Config) http.Handler {
	tmpls := template.Must(template.New("").
		Funcs(templateFuncs).
		ParseFS(web.TemplatesFS, "*"))
	h := handlers{
		cfg, tmpls,
		cache.NewInMemory[[]repoList](cfg.Cache.HomePage),
		cache.NewInMemory[template.HTML](cfg.Cache.Readme),
		cache.NewInMemory[*git.NiceDiff](cfg.Cache.Diff),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.indexHandler)
	mux.HandleFunc("GET /index.xml", h.indexFeedHandler)
	mux.HandleFunc("GET /static/{file}", h.serveStaticHandler)
	mux.HandleFunc("GET /{name}", h.repoIndexHandler)
	mux.HandleFunc("GET /{name}/info/refs", h.infoRefsHandler)
	mux.HandleFunc("POST /{name}/git-upload-pack", h.uploadPackHandler)
	mux.HandleFunc("POST /{name}/git-receive-pack", h.receivePackHandler)
	mux.HandleFunc("GET /{name}/feed/{$}", h.repoFeedHandler)
	mux.HandleFunc("GET /{name}/tree/{ref}/{rest...}", h.repoTreeHandler)
	mux.HandleFunc("GET /{name}/blob/{ref}/{rest...}", h.fileContentsHandler)
	mux.HandleFunc("GET /{name}/log/{ref}", h.logHandler)
	mux.HandleFunc("GET /{name}/commit/{ref}", h.commitHandler)
	mux.HandleFunc("GET /{name}/refs/{$}", h.refsHandler)
	mux.HandleFunc("GET /{name}/archive/{ref}", h.archiveHandler)

	handler := h.recoverMiddleware(mux)
	return h.loggingMiddleware(handler)
}

func (h *handlers) serveStaticHandler(w http.ResponseWriter, r *http.Request) {
	f := filepath.Clean(r.PathValue("file"))
	http.ServeFileFS(w, r, web.StaticFS, f)
}

// parseRef parses url encoded ref name.
// If it fails it falls back to raw provided value.
func (h handlers) parseRef(name string) string {
	ref, err := url.PathUnescape(name)
	if err != nil {
		return name
	}
	return ref
}

var templateFuncs = template.FuncMap{
	"humanizeRelTime": func(t time.Time) string { return humanize.Time(t) },
	"humanizeTime":    func(t time.Time) string { return t.Format("2006-01-02 15:04:05 MST") },
	"urlencode":       func(s string) string { return url.PathEscape(s) },
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
