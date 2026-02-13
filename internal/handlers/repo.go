package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"olexsmir.xyz/mugit/internal/git"
)

func (h *handlers) indexHandler(w http.ResponseWriter, r *http.Request) {
	repos, err := h.listPublicRepos()
	if err != nil {
		h.write500(w, err)
		return
	}

	data := make(map[string]any)
	data["meta"] = h.c.Meta
	data["repos"] = repos
	h.templ(w, "index", data)
}

var markdown = goldmark.New(
	goldmark.WithRendererOptions(html.WithUnsafe()),
	goldmark.WithExtensions(
		extension.GFM,
		extension.Linkify,
	))

func (h *handlers) repoIndex(w http.ResponseWriter, r *http.Request) {
	repo, err := h.openPublicRepo(r.PathValue("name"), "")
	if err != nil {
		h.write404(w, err)
		return
	}

	desc, err := repo.Description()
	if err != nil {
		h.write500(w, err)
		return
	}

	data := make(map[string]any)
	data["name"] = repo.Name()
	data["desc"] = desc
	data["servername"] = h.c.Meta.Host
	data["meta"] = h.c.Meta

	if repo.IsEmpty() {
		data["empty"] = true
		h.templ(w, "repo_index", data)
		return
	}

	var readmeContents template.HTML
	for _, readme := range h.c.Repo.Readmes {
		ext := filepath.Ext(readme)
		content, _ := repo.FileContent(readme)
		if len(content) > 0 {
			switch ext {
			case ".md", ".markdown", ".mkd":
				var buf bytes.Buffer
				if cerr := markdown.Convert([]byte(content), &buf); cerr != nil {
					h.write500(w, cerr)
					return
				}
				readmeContents = template.HTML(buf.String())
			default:
				readmeContents = template.HTML(fmt.Sprintf(`<pre>%s</pre>`, content))
			}
			break
		}
	}

	masterBranch, err := repo.FindMasterBranch(h.c.Repo.Masters)
	if err != nil {
		h.write500(w, err)
		return
	}

	commits, err := repo.Commits()
	if err != nil {
		h.write500(w, err)
		return
	}

	if len(commits) >= 4 {
		commits = commits[:3]
	}

	data["ref"] = masterBranch
	data["readme"] = readmeContents
	data["commits"] = commits
	data["gomod"] = repo.IsGoMod()

	if isMirror, err := repo.IsMirror(); err == nil && isMirror {
		lastSync, _ := repo.LastSync()
		remoteURL, _ := repo.RemoteURL()
		data["mirrorinfo"] = map[string]any{
			"isMirror": true,
			"url":      remoteURL,
			"lastSync": lastSync,
		}
	}

	h.templ(w, "repo_index", data)
}

func (h *handlers) repoTreeHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ref := r.PathValue("ref")
	treePath := r.PathValue("rest")

	repo, err := h.openPublicRepo(name, ref)
	if err != nil {
		h.write404(w, err)
		return
	}

	desc, err := repo.Description()
	if err != nil {
		h.write500(w, err)
		return
	}

	files, err := repo.FileTree(treePath)
	if err != nil {
		h.write500(w, err)
		return
	}

	data := make(map[string]any)
	data["name"] = name
	data["ref"] = ref
	data["parent"] = treePath
	data["dotdot"] = filepath.Dir(treePath)
	data["desc"] = desc
	data["meta"] = h.c.Meta
	data["files"] = files

	h.templ(w, "repo_tree", data)
}

func (h *handlers) fileContentsHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ref := r.PathValue("ref")
	treePath := r.PathValue("rest")

	var raw bool
	if rawParam, err := strconv.ParseBool(r.URL.Query().Get("raw")); err == nil {
		raw = rawParam
	}

	repo, err := h.openPublicRepo(name, ref)
	if err != nil {
		h.write404(w, err)
		return
	}

	desc, err := repo.Description()
	if err != nil {
		h.write500(w, err)
		return
	}

	contents, err := repo.FileContent(treePath)
	if err != nil {
		if errors.Is(err, git.ErrFileNotFound) {
			h.write404(w, err)
			return
		}
		h.write500(w, err)
		return
	}

	if raw {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(contents))
		return
	}

	data := make(map[string]any)
	data["name"] = name
	data["ref"] = ref
	data["desc"] = desc
	data["path"] = treePath

	lc, err := countLines(strings.NewReader(contents))
	if err != nil {
		slog.Error("failed to count line numbers", "err", err)
	}

	lines := make([]int, lc)
	if lc > 0 {
		for i := range lines {
			lines[i] = i + 1
		}
	}

	data["linecount"] = lines
	data["content"] = contents
	data["meta"] = h.c.Meta

	h.templ(w, "repo_file", data)
}

func (h *handlers) logHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ref := r.PathValue("ref")

	repo, err := h.openPublicRepo(name, ref)
	if err != nil {
		h.write404(w, err)
		return
	}

	commits, err := repo.Commits()
	if err != nil {
		h.write500(w, err)
		return
	}

	desc, err := repo.Description()
	if err != nil {
		h.write500(w, err)
		return
	}

	data := make(map[string]any)
	data["name"] = name
	data["ref"] = ref
	data["desc"] = desc
	data["meta"] = h.c.Meta
	data["log"] = true
	data["commits"] = commits
	h.templ(w, "repo_log", data)
}

func (h *handlers) commitHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ref := r.PathValue("ref")
	repo, err := h.openPublicRepo(name, ref)
	if err != nil {
		h.write404(w, err)
		return
	}

	diff, err := repo.Diff()
	if err != nil {
		h.write500(w, err)
		return
	}

	desc, err := repo.Description()
	if err != nil {
		h.write500(w, err)
		return
	}

	data := make(map[string]any)
	data["diff"] = diff.Diff
	data["commit"] = diff.Commit
	data["parents"] = diff.Parents
	data["stat"] = diff.Stat
	data["name"] = name
	data["ref"] = ref
	data["desc"] = desc
	h.templ(w, "repo_commit", data)
}

func (h *handlers) refsHandler(w http.ResponseWriter, r *http.Request) {
	repo, err := h.openPublicRepo(r.PathValue("name"), "")
	if err != nil {
		h.write404(w, err)
		return
	}

	desc, err := repo.Description()
	if err != nil {
		h.write500(w, err)
		return
	}

	masterBranch, err := repo.FindMasterBranch(h.c.Repo.Masters)
	if err != nil {
		h.write500(w, err)
		return
	}

	branches, err := repo.Branches()
	if err != nil {
		h.write500(w, err)
		return
	}

	tags, err := repo.Tags()
	if err != nil {
		// repo should have at least one branch, tags are *optional*
		slog.Error("couldn't fetch repo tags", "err", err)
	}

	data := make(map[string]any)
	data["meta"] = h.c.Meta
	data["name"] = repo.Name()
	data["desc"] = desc
	data["ref"] = masterBranch
	data["branches"] = branches
	data["tags"] = tags
	h.templ(w, "repo_refs", data)
}

func countLines(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	bufLen := 0
	count := 0
	nl := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		if c > 0 {
			bufLen += c
		}
		count += bytes.Count(buf[:c], nl)

		switch {
		case err == io.EOF:
			// handle last line not having a newline at the end
			if bufLen >= 1 && buf[(bufLen-1)%(32*1024)] != '\n' {
				count++
			}
			return count, nil
		case err != nil:
			return 0, err
		}
	}
}

type repoList struct {
	Name       string
	Desc       string
	LastCommit time.Time
}

func (h *handlers) listPublicRepos() ([]repoList, error) {
	dirs, err := os.ReadDir(h.c.Repo.Dir)
	if err != nil {
		return nil, err
	}

	var repos []repoList
	var errs []error
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		name := dir.Name()
		repo, err := h.openPublicRepo(name, "")
		if err != nil {
			// if it's not git repo, just ignore it
			continue
		}

		desc, err := repo.Description()
		if err != nil {
			errs = append(errs, err)
			continue
		}

		lastCommit, err := repo.LastCommit()
		if err != nil {
			errs = append(errs, err)
			continue
		}

		repos = append(repos, repoList{
			Name:       repo.Name(),
			Desc:       desc,
			LastCommit: lastCommit.Committed,
		})
	}

	sort.Slice(repos, func(i, j int) bool {
		return repos[j].LastCommit.Before(repos[i].LastCommit)
	})

	return repos, errors.Join(errs...)
}
