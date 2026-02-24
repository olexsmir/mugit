package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"olexsmir.xyz/mugit/internal/git"
	"olexsmir.xyz/mugit/internal/git/gitx"
)

// multiplex, check if the request smells like gitprotocol-http(5), if so, it
// passes it to git smart http, otherwise renders templates
func (h *handlers) multiplex(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery == "service=git-receive-pack" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("http pushing is not supported"))
		return
	}

	path := r.PathValue("rest")
	if path == "info/refs" && r.Method == "GET" && r.URL.RawQuery == "service=git-upload-pack" {
		h.infoRefs(w, r)
	} else if path == "git-upload-pack" && r.Method == "POST" {
		h.uploadPack(w, r)
	} else if r.Method == "GET" && path == "" {
		h.repoIndex(w, r)
	} else {
		h.write404(w, nil)
	}
}

func (h *handlers) infoRefs(w http.ResponseWriter, r *http.Request) {
	path, err := h.checkRepoPublicityAndGetPath(r.PathValue("name"), "")
	if err != nil {
		h.write404(w, err)
		return
	}

	w.Header().Set("content-type", "application/x-git-upload-pack-advertisement")
	w.WriteHeader(http.StatusOK)
	if err := gitx.InfoRefs(r.Context(), path, w); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("git: info/refs", "err", err)
		return
	}
}

func (h *handlers) uploadPack(w http.ResponseWriter, r *http.Request) {
	path, err := h.checkRepoPublicityAndGetPath(r.PathValue("name"), "")
	if err != nil {
		h.write404(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	if err := gitx.UploadPack(r.Context(), path, true, r.Body, newFlushWriter(w)); err != nil {
		slog.Error("git: upload-pack", "err", err)
		return
	}
}

func (h *handlers) archiveHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ref := h.parseRef(r.PathValue("ref"))

	path, err := h.checkRepoPublicityAndGetPath(name, ref)
	if err != nil {
		h.write404(w, err)
		return
	}

	filename := fmt.Sprintf("%s-%s.tar.gz", name, ref)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Content-Type", "application/gzip")
	w.WriteHeader(http.StatusOK)

	if err := gitx.ArchiveTar(r.Context(), path, ref, w); err != nil {
		slog.Error("git: archive", "ref", ref, "err", err)
		return
	}
}

func (h *handlers) checkRepoPublicityAndGetPath(name string, ref string) (string, error) {
	name = git.ResolveName(name)
	path, err := git.ResolvePath(h.c.Repo.Dir, name)
	if err != nil {
		return "", err
	}

	if _, oerr := git.OpenPublic(path, ref); oerr != nil {
		return "", oerr
	}

	return path, err
}

func (h *handlers) openPublicRepo(name, ref string) (*git.Repo, error) {
	name = git.ResolveName(name)
	path, err := git.ResolvePath(h.c.Repo.Dir, name)
	if err != nil {
		return nil, err
	}
	return git.OpenPublic(path, ref)
}

type flushWriter struct {
	w io.Writer
	f http.Flusher
}

func newFlushWriter(w http.ResponseWriter) io.Writer {
	f, _ := w.(http.Flusher)
	return &flushWriter{w: w, f: f}
}

func (fw *flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	if fw.f != nil {
		fw.f.Flush()
	}
	return n, err
}
