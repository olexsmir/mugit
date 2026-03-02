package handlers

import (
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"olexsmir.xyz/mugit/internal/git"
	"olexsmir.xyz/mugit/internal/git/gitx"
)

func (h *handlers) infoRefsHandler(w http.ResponseWriter, r *http.Request) {
	path, err := h.checkRepoPublicityAndGetPath(r.PathValue("name"), "")
	if err != nil {
		h.gitError(w, http.StatusNotFound, "repository not found")
		return
	}

	service := r.URL.Query().Get("service")
	switch service {
	case "git-upload-pack":
		w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
		w.Header().Set("Connection", "Keep-Alive")
		w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")

		w.WriteHeader(http.StatusOK)
		if err := gitx.InfoRefs(r.Context(), path, w); err != nil {
			h.gitError(w, http.StatusInternalServerError, err.Error())
			slog.Error("git: info/refs", "err", err)
			return
		}

	case "git-receive-pack":
		h.receivePackHandler(w, r)

	default:
		h.gitError(w, http.StatusBadRequest, "service unsupported")
	}
}

const uploadPackExpectedContentType = "application/x-git-upload-pack-request"

func (h *handlers) uploadPackHandler(w http.ResponseWriter, r *http.Request) {
	path, err := h.checkRepoPublicityAndGetPath(r.PathValue("name"), "")
	if err != nil {
		h.gitError(w, http.StatusNotFound, "repository not found")
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != uploadPackExpectedContentType {
		h.gitError(w, http.StatusUnsupportedMediaType, "provided content type is not supported")
		return
	}

	bodyReader := r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(r.Body)
		if err != nil {
			h.gitError(w, http.StatusInternalServerError, err.Error())
			slog.Error("git: failed to create gzip reader", "err", err)
			return
		}
		defer gzipReader.Close()
		bodyReader = gzipReader
	}

	w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("Cache-Control", "no-cache, max-age=0, must-revalidate")

	w.WriteHeader(http.StatusOK)
	if err := gitx.UploadPack(r.Context(), path, true, bodyReader, newFlushWriter(w)); err != nil {
		h.gitError(w, http.StatusInternalServerError, err.Error())
		slog.Error("git: upload-pack", "err", err)
		return

	}
}

func (h *handlers) receivePackHandler(w http.ResponseWriter, _ *http.Request) {
	h.gitError(w, http.StatusForbidden, "pushes are only supported over ssh")
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

func (h *handlers) gitError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("content-type", "text/plain; charset=UTF-8")
	w.WriteHeader(code)
	fmt.Fprintf(w, "%s\n", msg)
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
