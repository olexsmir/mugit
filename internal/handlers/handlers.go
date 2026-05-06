package handlers

import (
	"errors"
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
	mux.HandleFunc("GET /{name}/{$}", h.repoIndexHandler)
	mux.HandleFunc("GET /{name}/info/refs", h.infoRefsHandler)
	mux.HandleFunc("POST /{name}/git-upload-pack", h.uploadPackHandler)
	mux.HandleFunc("POST /{name}/git-receive-pack", h.receivePackHandler)
	mux.HandleFunc("GET /{name}/feed/{$}", h.repoFeedHandler)
	mux.HandleFunc("GET /{name}/tree/{ref}/{rest...}", h.repoTreeHandler)
	mux.HandleFunc("GET /{name}/blob/{ref}/{rest...}", h.fileContentsHandler)
	mux.HandleFunc("GET /{name}/raw/{ref}/{rest...}", h.rawFileContentsHandler)
	mux.HandleFunc("GET /{name}/log/{ref}", h.logHandler)
	mux.HandleFunc("GET /{name}/commit/{ref}", h.commitHandler)
	mux.HandleFunc("GET /{name}/compare/{ref1}/{ref2}", h.compareHandler)
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
	"inc":             func(n int) int { return n + 1 },
	"inc64":           func(n int64) int64 { return n + 1 },
	"humanizeTime":    func(t time.Time) string { return t.Format("2006-01-02 15:04:05 MST") },
	"humanizeRelTime": humanize.Time,
	"urlencode":       url.PathEscape,
	"commitSummary":   commitSummary,
	"dict":            dict,
}

func commitSummary(commitMsg string) string {
	before, after, found := strings.Cut(commitMsg, "\n")
	first := strings.TrimSuffix(before, "\r")
	if !found {
		return first
	}

	// if there is any content after the first newline, indicate it with "..."
	after = strings.TrimLeft(after, "\r\n")
	if after != "" {
		return first + "..."
	}

	return first
}

func dict(v ...any) (map[string]any, error) {
	if len(v)%2 != 0 {
		return nil, errors.New("dict requires an even number of arguments")
	}

	out := make(map[string]any, len(v)/2)
	for i := 0; i < len(v); i += 2 {
		key, ok := v[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		out[key] = v[i+1]
	}
	return out, nil
}
