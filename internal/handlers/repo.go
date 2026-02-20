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
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"olexsmir.xyz/mugit/internal/git"
	"olexsmir.xyz/mugit/internal/mdx"
)

type Meta struct {
	Title       string
	Description string
	Host        string
	IsEmpty     bool
	GoMod       bool
	SSHEnabled  bool
}

type RepoBase struct {
	Ref  string
	Desc string
}

type PageData[T any] struct {
	Meta     Meta
	RepoName string // empty for non-repo pages, needed for _head.html to  compile
	P        T
}

func (h *handlers) indexHandler(w http.ResponseWriter, r *http.Request) {
	repos, err := h.listPublicRepos()
	if err != nil {
		h.write500(w, err)
		return
	}
	h.templ(w, "index", h.pageData(nil, repos))
}

type RepoIndex struct {
	Desc           string
	IsEmpty        bool
	Readme         template.HTML
	Ref            string
	Commits        []*git.Commit
	IsMirror       bool
	MirrorURL      string
	MirrorLastSync time.Time
}

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

	p := RepoIndex{Desc: desc, IsEmpty: repo.IsEmpty()}
	if p.IsEmpty {
		h.templ(w, "repo_index", h.pageData(repo, p))
		return
	}

	p.Ref, err = repo.FindMasterBranch(h.c.Repo.Masters)
	if err != nil {
		h.write500(w, err)
		return
	}

	p.Readme, err = h.renderReadme(repo, p.Ref, "")
	if err != nil {
		h.write500(w, err)
		return
	}

	p.Commits, err = repo.Commits()
	if err != nil {
		h.write500(w, err)
		return
	}

	if len(p.Commits) >= 3 {
		p.Commits = p.Commits[:3]
	}

	if isMirror, err := repo.IsMirror(); isMirror && err == nil {
		p.IsMirror = true
		p.MirrorURL, _ = repo.RemoteURL()
		p.MirrorLastSync, _ = repo.LastSync()
	}

	h.templ(w, "repo_index", h.pageData(repo, p))
}

type RepoTree struct {
	Desc       string
	Ref        string
	Tree       []git.NiceTree
	ParentPath string
	DotDot     string
	Readme     template.HTML
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

	tree, err := repo.FileTree(treePath)
	if err != nil {
		h.write500(w, err)
		return
	}

	readme, err := h.renderReadme(repo, ref, treePath)
	if err != nil {
		h.write500(w, err)
		return
	}

	h.templ(w, "repo_tree", h.pageData(repo, RepoTree{
		Desc:       desc,
		Ref:        ref,
		Tree:       tree,
		ParentPath: treePath,
		DotDot:     filepath.Dir(treePath),
		Readme:     readme,
	}))
}

type RepoFile struct {
	Ref       string
	Desc      string
	LineCount []int
	Path      string
	IsImage   bool
	IsBinary  bool
	Content   string
	Mime      string
	Size      int64
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

	fc, err := repo.FileContent(treePath)
	if err != nil {
		if errors.Is(err, git.ErrFileNotFound) {
			h.write404(w, err)
			return
		}
		h.write500(w, err)
		return
	}

	if raw {
		w.Header().Set("Content-Type", fc.Mime)
		w.WriteHeader(http.StatusOK)
		w.Write(fc.Content)
		return
	}

	p := RepoFile{
		Ref:      ref,
		Path:     treePath,
		IsImage:  fc.IsImage(),
		IsBinary: fc.IsBinary,
		Mime:     fc.Mime,
		Size:     fc.Size,
	}

	p.Desc, err = repo.Description()
	if err != nil {
		h.write500(w, err)
		return
	}

	if !fc.IsImage() && !fc.IsBinary {
		contentStr := fc.String()
		lc, err := countLines(strings.NewReader(contentStr))
		if err != nil {
			slog.Error("failed to count line numbers", "err", err)
		}
		lines := make([]int, lc)
		for i := range lines {
			lines[i] = i + 1
		}
		p.Content = contentStr
		p.LineCount = lines
	}

	h.templ(w, "repo_file", h.pageData(repo, p))
}

type RepoLog struct {
	Desc    string
	Commits []*git.Commit
	Ref     string
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

	h.templ(w, "repo_log", h.pageData(repo, RepoLog{
		Desc:    desc,
		Commits: commits,
		Ref:     ref,
	}))
}

type RepoCommit struct {
	Diff *git.NiceDiff
	Ref  string
	Desc string
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

	h.templ(w, "repo_commit", h.pageData(repo, RepoCommit{
		Desc: desc,
		Ref:  ref,
		Diff: diff,
	}))
}

type RepoRefs struct {
	Desc     string
	Ref      string
	Branches []*git.Branch
	Tags     []*git.TagReference
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

	master, err := repo.FindMasterBranch(h.c.Repo.Masters)
	if err != nil {
		h.write500(w, err)
		return
	}

	branches, err := repo.Branches()
	if err != nil {
		h.write500(w, err)
		return
	}

	// repo should have at least one branch, tags are *optional*
	tags, _ := repo.Tags()

	h.templ(w, "repo_refs", h.pageData(repo, RepoRefs{
		Desc:     desc,
		Ref:      master,
		Tags:     tags,
		Branches: branches,
	}))
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
	if v, found := h.repoListCache.Get("repo_list"); found {
		return v, nil
	}

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

	h.repoListCache.Set("repo_list", repos)
	return repos, errors.Join(errs...)
}

var markdown = goldmark.New(
	goldmark.WithRendererOptions(html.WithUnsafe()),
	goldmark.WithExtensions(
		extension.GFM,
		extension.Linkify,
		emoji.Emoji,
		mdx.RelativeLink,
	))

func (h *handlers) renderReadme(r *git.Repo, ref, treePath string) (template.HTML, error) {
	name := r.Name()
	cacheKey := fmt.Sprintf("%s:%s:%s", name, ref, treePath)
	if v, found := h.readmeCache.Get(cacheKey); found {
		return v, nil
	}

	var readmeContents template.HTML
	for _, readme := range h.c.Repo.Readmes {
		fullPath := filepath.Join(treePath, readme)
		fc, ferr := r.FileContent(fullPath)
		if ferr != nil {
			continue
		}

		if fc.IsBinary {
			continue
		}

		ext := filepath.Ext(readme)
		content := fc.String()
		if len(content) > 0 {
			switch ext {
			case ".md", ".markdown", ".mkd":
				var buf bytes.Buffer
				if cerr := markdown.Convert([]byte(content), &buf,
					mdx.NewRelativeLinkCtx(name, fullPath)); cerr != nil {
					return "", cerr
				}
				readmeContents = template.HTML(buf.String())
			default:
				readmeContents = template.HTML(fmt.Sprintf(`<pre>%s</pre>`, content))
			}
			break
		}
	}

	h.readmeCache.Set(cacheKey, readmeContents)
	return readmeContents, nil
}

func (h handlers) pageData(repo *git.Repo, p any) PageData[any] {
	var name string
	var gomod, empty bool
	if repo != nil {
		gomod = repo.IsGoMod()
		empty = repo.IsEmpty()
		name = repo.Name()
	}

	return PageData[any]{
		P:        p,
		RepoName: name,
		Meta: Meta{
			Title:       h.c.Meta.Title,
			Description: h.c.Meta.Description,
			Host:        h.c.Meta.Host,
			GoMod:       gomod,
			SSHEnabled:  h.c.SSH.Enable,
			IsEmpty:     empty,
		},
	}
}
