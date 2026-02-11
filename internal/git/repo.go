package git

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// Thanks https://git.icyphox.sh/legit/blob/master/git/git.go

var (
	ErrEmptyRepo    = errors.New("repository has no commits")
	ErrFileNotFound = errors.New("file not found")
)

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

// Init initializes a bare repo in path.
func Init(path string) error {
	if _, err := git.PlainInit(path, true); err != nil {
		return fmt.Errorf("failed to initialize repo: %w", err)
	}
	return nil
}

func (g *Repo) Name() string {
	name := filepath.Base(g.path)
	return strings.TrimSuffix(name, ".git")
}

type Commit struct {
	Message     string
	AuthorEmail string
	AuthorName  string
	ChangeID    string
	Committed   time.Time
	Hash        string
	HashShort   string
}

func newShortHash(h plumbing.Hash) string { return h.String()[:7] }
func newCommit(c *object.Commit) *Commit {
	var changeID string
	for _, header := range c.ExtraHeaders {
		if header.Key == "change-id" {
			changeID = header.Value
			break
		}
	}

	return &Commit{
		Message:     c.Message,
		AuthorEmail: c.Author.Email,
		AuthorName:  c.Author.Name,
		ChangeID:    changeID,
		Committed:   c.Committer.When,
		Hash:        c.Hash.String(),
		HashShort:   newShortHash(c.Hash),
	}
}

func (g *Repo) Commits() ([]*Commit, error) {
	if g.IsEmpty() {
		return []*Commit{}, nil
	}

	ci, err := g.r.Log(&git.LogOptions{
		From:  g.h,
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, fmt.Errorf("commits from ref: %w", err)
	}

	var commits []*Commit
	ci.ForEach(func(c *object.Commit) error {
		commits = append(commits, newCommit(c))
		return nil
	})

	return commits, nil
}

func (g *Repo) LastCommit() (*Commit, error) {
	if g.IsEmpty() {
		return &Commit{}, nil
	}

	c, err := g.r.CommitObject(g.h)
	if err != nil {
		return nil, fmt.Errorf("last commit: %w", err)
	}

	return newCommit(c), nil
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
		if errors.Is(err, object.ErrFileNotFound) {
			return "", ErrFileNotFound
		}
		return "", err
	}

	isbin, _ := file.IsBinary()
	if !isbin {
		return file.Contents()
	} else {
		return "Not displaying binary file", nil
	}
}

type Branch struct{ Name string }

func (g *Repo) Branches() ([]*Branch, error) {
	bi, err := g.r.Branches()
	if err != nil {
		return nil, fmt.Errorf("branch: %w", err)
	}

	var branches []*Branch
	err = bi.ForEach(func(r *plumbing.Reference) error {
		branches = append(branches, &Branch{
			Name: r.Name().Short(),
		})
		return nil
	})
	return branches, err
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

func (g *Repo) Fetch() error { return g.fetch(nil) }

func (g *Repo) FetchFromGithubWithToken(token string) error {
	return g.fetch(&http.BasicAuth{
		Username: "x-access-token", // this can be anything but empty
		Password: token,
	})
}

func (g *Repo) fetch(auth transport.AuthMethod) error {
	rmt, err := g.r.Remote(originRemote)
	if err != nil {
		return fmt.Errorf("failed to get remote: %w", err)
	}

	if err = rmt.Fetch(&git.FetchOptions{
		Auth:  auth,
		Tags:  git.AllTags,
		Prune: true,
		Force: true,
	}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	// for some reason fetch doesn't change head for empty repos
	if !g.IsEmpty() {
		return nil
	}

	refs, err := rmt.List(&git.ListOptions{Auth: auth})
	if err != nil {
		return fmt.Errorf("failed to list references: %w", err)
	}

	for _, ref := range refs {
		if ref.Name() == plumbing.HEAD {
			if err := g.r.Storer.SetReference(
				plumbing.NewSymbolicReference(plumbing.HEAD, ref.Target()),
			); err != nil {
				return fmt.Errorf("failed to set HEAD: %w", err)
			}
			break
		}
	}

	return nil
}
