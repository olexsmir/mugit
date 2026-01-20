package handlers

import (
	"bytes"
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
	"olexsmir.xyz/mugit/internal/humanize"
)

func (h *handlers) index(w http.ResponseWriter, r *http.Request) {
	dirs, err := os.ReadDir(h.c.Repo.Dir)
	if err != nil {
		h.write500(w, err)
		return
	}

	type repoInfo struct {
		Name, Desc, Idle string
		t                time.Time
	}

	repoInfos := []repoInfo{}
	for _, dir := range dirs {
		name := dir.Name()
		repo, err := h.openPublicRepo(name, "")
		if err != nil {
			slog.Error("", "name", name, "err", err)
			continue
		}

		desc, err := repo.Description()
		if err != nil {
			slog.Error("", "err", err)
			continue
		}

		lastComit, err := repo.LastCommit()
		if err != nil {
			slog.Error("", "err", err)
			continue
		}

		repoInfos = append(repoInfos, repoInfo{
			Name: name,
			Desc: desc,
			Idle: humanize.Time(lastComit.Author.When),
			t:    lastComit.Author.When,
		})
	}

	sort.Slice(repoInfos, func(i, j int) bool {
		return repoInfos[j].t.Before(repoInfos[i].t)
	})

	data := make(map[string]any)
	data["meta"] = h.c.Meta
	data["repos"] = repoInfos
	h.templ(w, "index", data)
}

var markdown = goldmark.New(
	goldmark.WithRendererOptions(html.WithUnsafe()),
	goldmark.WithExtensions(
		extension.GFM,
		extension.Linkify,
	))

func (h *handlers) repoIndex(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	repo, err := h.openPublicRepo(name, "")
	if err != nil {
		h.write404(w, err)
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

	desc, err := repo.Description()
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

	data := make(map[string]any)
	data["name"] = name
	data["ref"] = masterBranch
	data["desc"] = desc
	data["readme"] = readmeContents
	data["commits"] = commits
	data["servername"] = h.c.Meta.Host
	data["meta"] = h.c.Meta
	data["gomod"] = repo.IsGoMod()

	h.templ(w, "repo_index", data)
}

func (h *handlers) repoTree(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ref := r.PathValue("ref")
	treePath := r.PathValue("rest")

	repo, err := h.openPublicRepo(name, ref)
	if err != nil {
		h.write404(w, err)
		return
	}

	isPrivate, err := repo.IsPrivate()
	if isPrivate || err != nil {
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

func (h *handlers) fileContents(w http.ResponseWriter, r *http.Request) {
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
		h.write500(w, err)
		return
	}

	data := make(map[string]any)
	data["name"] = name
	data["ref"] = ref
	data["desc"] = desc
	data["path"] = treePath

	if raw {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(contents))
		return
	}

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

	h.templ(w, "file", data)
}

func (h *handlers) log(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ref := r.PathValue("ref")

	repo, err := h.openPublicRepo(name, ref)
	if err != nil {
		h.write404(w, err)
		return
	}

	isPrivate, err := repo.IsPrivate()
	if isPrivate || err != nil {
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

func (h *handlers) commit(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ref := r.PathValue("ref")
	repo, err := h.openPublicRepo(name, ref)
	if err != nil {
		h.write404(w, err)
		return
	}

	isPrivate, err := repo.IsPrivate()
	if isPrivate || err != nil {
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
	data["stat"] = diff.Stat
	data["diff"] = diff.Diff
	data["commit"] = diff.Commit
	data["name"] = name
	data["ref"] = ref
	data["desc"] = desc
	h.templ(w, "commit", data)
}

func (h *handlers) refs(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	repo, err := h.openPublicRepo(name, "")
	if err != nil {
		h.write404(w, err)
		return
	}

	isPrivate, err := repo.IsPrivate()
	if isPrivate || err != nil {
		h.write404(w, err)
		return
	}

	desc, err := repo.Description()
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
	data["name"] = name
	data["desc"] = desc
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
