package handlers

import (
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"

	"olexsmir.xyz/mugit/internal/git/gitservice"
)

func (h *handlers) infoRefs(w http.ResponseWriter, r *http.Request) {
	// TODO: 404 for private repos

	name := filepath.Clean(r.PathValue("name"))

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
	// TODO: 404 for private repos

	name := filepath.Clean(r.PathValue("name"))

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
