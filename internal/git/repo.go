package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// Thanks https://git.icyphox.sh/legit/blob/master/git/git.go

var ErrEmptyRepo = errors.New("repository has no commits")

type Repo struct {
	path string
	r    *git.Repository
	h    plumbing.Hash
}

// Open opens a git repository at path. If ref is empty, HEAD is used.
func Open(path string, ref string) (*Repo, error) {
	var err error
	g := Repo{}
	g.path = path
	g.r, err = git.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}

	if ref == "" {
		head, err := g.r.Head()
		if err != nil {
			if errors.Is(err, plumbing.ErrReferenceNotFound) {
				return &g, nil
			}
			return nil, fmt.Errorf("getting head of %s: %w", path, err)
		}
		g.h = head.Hash()
	} else {
		hash, err := g.r.ResolveRevision(plumbing.Revision(ref))
		if err != nil {
			return nil, fmt.Errorf("resolving rev %s for %s: %w", ref, path, err)
		}
		g.h = *hash
	}
	return &g, nil
}

func (g *Repo) IsEmpty() bool {
	return g.h == plumbing.ZeroHash
}

// Init creates a bare repo.
func Init(path string) error {
	_, err := git.PlainInit(path, true)
	return err
}

func (g *Repo) Name() string {
	name := filepath.Base(g.path)
	return strings.TrimSuffix(name, ".git")
}

func (g *Repo) Commits() ([]*object.Commit, error) {
	if g.IsEmpty() {
		return []*object.Commit{}, nil
	}

	ci, err := g.r.Log(&git.LogOptions{
		From:  g.h,
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, fmt.Errorf("commits from ref: %w", err)
	}

	commits := []*object.Commit{}
	ci.ForEach(func(c *object.Commit) error {
		commits = append(commits, c)
		return nil
	})

	return commits, nil
}

func (g *Repo) LastCommit() (*object.Commit, error) {
	if g.IsEmpty() {
		return nil, ErrEmptyRepo
	}

	c, err := g.r.CommitObject(g.h)
	if err != nil {
		return nil, fmt.Errorf("last commit: %w", err)
	}
	return c, nil
}

func (g *Repo) FileContent(path string) (string, error) {
	c, err := g.r.CommitObject(g.h)
	if err != nil {
		return "", fmt.Errorf("commit object: %w", err)
	}

	tree, err := c.Tree()
	if err != nil {
		return "", fmt.Errorf("file tree: %w", err)
	}

	file, err := tree.File(path)
	if err != nil {
		return "", err
	}

	isbin, _ := file.IsBinary()
	if !isbin {
		return file.Contents()
	} else {
		return "Not displaying binary file", nil
	}
}

func (g *Repo) Tags() ([]*TagReference, error) {
	iter, err := g.r.Tags()
	if err != nil {
		return nil, fmt.Errorf("tag objects: %w", err)
	}

	tags := make([]*TagReference, 0)
	if err := iter.ForEach(func(ref *plumbing.Reference) error {
		obj, err := g.r.TagObject(ref.Hash())
		switch err {
		case nil:
			tags = append(tags, &TagReference{
				ref: ref,
				tag: obj,
			})
		case plumbing.ErrObjectNotFound:
			tags = append(tags, &TagReference{
				ref: ref,
			})
		default:
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	tagList := &TagList{r: g.r, refs: tags}
	sort.Sort(tagList)
	return tags, nil
}

func (g *Repo) Branches() ([]*plumbing.Reference, error) {
	bi, err := g.r.Branches()
	if err != nil {
		return nil, fmt.Errorf("branch: %w", err)
	}

	branches := []*plumbing.Reference{}
	err = bi.ForEach(func(ref *plumbing.Reference) error {
		branches = append(branches, ref)
		return nil
	})
	return branches, err
}

const defaultDescription = "Unnamed repository; edit this file 'description' to name the repository"

func (g *Repo) Description() (string, error) {
	// TODO: ??? Support both mugit.description and /description file
	path := filepath.Join(g.path, "description")
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("no description file found")
	}

	d, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read description: %w", err)
	}

	desc := string(d)
	if strings.Contains(desc, defaultDescription) {
		return "", nil
	}

	return desc, nil
}

func (g *Repo) IsPrivate() (bool, error) {
	c, err := g.r.Config()
	if err != nil {
		return false, fmt.Errorf("failed to read config: %w", err)
	}

	s := c.Raw.Section("mugit")
	return s.Options.Get("private") == "true", nil
}

func (g *Repo) IsGoMod() bool {
	_, err := g.FileContent("go.mod")
	return err == nil
}

func (g *Repo) FindMasterBranch(masters []string) (string, error) {
	if g.IsEmpty() {
		return "", ErrEmptyRepo
	}

	for _, b := range masters {
		if _, err := g.r.ResolveRevision(plumbing.Revision(b)); err == nil {
			return b, nil
		}
	}
	return "", fmt.Errorf("unable to find master branch")
}

type MirrorInfo struct {
	IsMirror  bool
	Remote    string
	RemoteURL string
}

func (g *Repo) MirrorInfo() (MirrorInfo, error) {
	c, err := g.r.Config()
	if err != nil {
		return MirrorInfo{}, fmt.Errorf("failed to read config: %w", err)
	}

	isMirror := c.Raw.Section("mugit").Options.Get("mirror") == "true"
	for _, remote := range c.Remotes {
		if len(remote.URLs) > 0 && (remote.Name == "upstream" || remote.Name == "origin") {
			return MirrorInfo{
				IsMirror:  isMirror,
				Remote:    remote.Name,
				RemoteURL: remote.URLs[0],
			}, nil
		}
	}
	// TODO: error if mirror opt is set, but there's no remotes
	return MirrorInfo{}, fmt.Errorf("no mirror remote found")
}

func (g *Repo) ReadLastSync() (time.Time, error) {
	c, err := g.r.Config()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to read config: %w", err)
	}

	raw := c.Raw.Section("mugit").Options.Get("last-sync")
	if raw == "" {
		return time.Time{}, fmt.Errorf("last-sync not set")
	}

	out, err := time.Parse(time.RFC3339, string(raw))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse time: %w", err)
	}
	return out, nil
}

func (g *Repo) SetLastSync(lastSync time.Time) error {
	c, err := g.r.Config()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	c.Raw.Section("mugit").
		SetOption("last-sync", lastSync.Format(time.RFC3339))
	return g.r.SetConfig(c)
}

func (g *Repo) Fetch(remote string) error {
	return g.FetchWithAuth(remote, "")
}

// FetchWithAuth fetches but with auth. Works only with github's auth
func (g *Repo) FetchWithAuth(remote string, token string) error {
	rmt, err := g.r.Remote(remote)
	if err != nil {
		return fmt.Errorf("failed to get upstream remote: %w", err)
	}

	opts := &git.FetchOptions{
		RefSpecs: []gitconfig.RefSpec{
			// fetch all branches
			"+refs/heads/*:refs/heads/*",
			"+refs/tags/*:refs/tags/*",
		},
		Tags:  git.AllTags,
		Prune: true,
		Force: true,
	}

	if token != "" {
		opts.Auth = &http.BasicAuth{
			Username: token,
			Password: "x-oauth-basic",
		}
	}

	if err := rmt.Fetch(opts); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fetch failed: %w", err)
	}
	return nil
}
