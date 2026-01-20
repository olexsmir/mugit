package handlers

import (
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"

	"olexsmir.xyz/mugit/internal/git"
	"olexsmir.xyz/mugit/internal/git/gitservice"
)

// multiplex, check if the request smells like gitprotocol-http(5), if so, it
// passes it to git smart http, otherwise renders templates
func (h *handlers) multiplex(w http.ResponseWriter, r *http.Request) {
	if r.URL.RawQuery == "service=git-receive-pack" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("http pushing isn't supported"))
		return
	}

	path := r.PathValue("rest")
	if path == "info/refs" && r.Method == "GET" && r.URL.RawQuery == "service=git-upload-pack" {
		h.infoRefs(w, r)
	} else if path == "git-upload-pack" && r.Method == "POST" {
		h.uploadPack(w, r)
	} else if r.Method == "GET" {
		h.repoIndex(w, r)
	}
}

func (h *handlers) infoRefs(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	_, err := h.openPublicRepo(name, "")
	if err != nil {
		h.write404(w, err)
		return
	}

	w.Header().Set("content-type", "application/x-git-upload-pack-advertisement")
	w.WriteHeader(http.StatusOK)

	if err := gitservice.InfoRefs(
		filepath.Join(h.c.Repo.Dir, name), // FIXME: use securejoin
		w,
	); err != nil {
		slog.Error("git: info/refs", "err", err)
		return
	}
}

func (h *handlers) uploadPack(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	_, err := h.openPublicRepo(name, "")
	if err != nil {
		h.write404(w, err)
		return
	}

	w.Header().Set("content-type", "application/x-git-upload-pack-result")
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	reader := io.Reader(r.Body)
	if r.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			slog.Error("git: gzip reader", "err", err)
			return
		}
		defer gr.Close()
		reader = gr
	}

	if err := gitservice.UploadPack(
		filepath.Join(h.c.Repo.Dir, name),
		true,
		reader,
		newFlushWriter(w),
	); err != nil {
		slog.Error("git: upload-pack", "err", err)
		return
	}
}

func (h *handlers) openPublicRepo(name, ref string) (*git.Repo, error) {
	n := filepath.Clean(name)
	repo, err := git.Open(filepath.Join(h.c.Repo.Dir, n), ref)
	if err != nil {
		return nil, err
	}

	isPrivate, err := repo.IsPrivate()
	if err != nil {
		return nil, err
	}

	if isPrivate {
		return nil, fmt.Errorf("repo is private")
	}

	return repo, nil
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
