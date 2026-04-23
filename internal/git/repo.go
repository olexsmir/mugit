package git

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// Thanks https://git.icyphox.sh/legit/blob/master/git/git.go

var (
	ErrEmptyRepo    = errors.New("repository has no commits")
	ErrFileNotFound = errors.New("file not found")
	ErrPrivate      = errors.New("repository is private")
	ErrRepoNotFound = errors.New("repository not found")
)

type Repo struct {
	path string
	r    *git.Repository
	h    plumbing.Hash
}

// Open opens a git repository at path. If ref is empty, HEAD is used.
func Open(path, ref string) (*Repo, error) {
	var err error
	g := Repo{}
	g.path = path
	g.r, err = git.PlainOpen(path)
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return nil, ErrRepoNotFound
		}
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

// OpenPublic opens a repository, returns [ErrPrivate] if it's private.
func OpenPublic(path, ref string) (*Repo, error) {
	r, err := Open(path, ref)
	if err != nil {
		return nil, err
	}

	isPrivate, err := r.IsPrivate()
	if err != nil {
		return nil, err
	}

	if isPrivate {
		return nil, ErrPrivate
	}

	return r, nil
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

func (g *Repo) DefaultBranch() (string, error) {
	out, err := g.runGitCmd("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get default branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (g *Repo) SetDefaultBranch(branch string) error {
	b := plumbing.NewBranchReferenceName(branch)
	_, err := g.r.Reference(b, true)
	if err != nil {
		return fmt.Errorf("branch %q not found: %w", branch, err)
	}
	head := plumbing.NewSymbolicReference(plumbing.HEAD, b)
	return g.r.Storer.SetReference(head)
}

type Commit struct {
	Message        string
	AuthorEmail    string
	AuthorName     string
	CommitterName  string
	CommitterEmail string
	Committed      time.Time
	ChangeID       string
	Hash           string
	HashShort      string
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
		Message:        c.Message,
		AuthorEmail:    c.Author.Email,
		AuthorName:     c.Author.Name,
		CommitterName:  c.Committer.Name,
		CommitterEmail: c.Committer.Email,
		Committed:      c.Committer.When,
		ChangeID:       changeID,
		Hash:           c.Hash.String(),
		HashShort:      newShortHash(c.Hash),
	}
}

const CommitsPage = 150

// Commits returns [CommitsPage] commits after the given commit hash cursor.
// If after is empty, starts from HEAD.
func (g *Repo) Commits(after string) ([]*Commit, error) {
	if g.IsEmpty() {
		return []*Commit{}, nil
	}

	from := g.h
	if after != "" {
		hash, err := g.r.ResolveRevision(plumbing.Revision(after))
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		from = *hash
	}

	ci, err := g.r.Log(&git.LogOptions{
		From:  from,
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, fmt.Errorf("commits from ref: %w", err)
	}

	// since after commit was shown on prev page, skip it
	if after != "" {
		ci.Next()
	}

	commits := make([]*Commit, 0, CommitsPage)
	ci.ForEach(func(c *object.Commit) error {
		if len(commits) == CommitsPage {
			return storer.ErrStop
		}
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

type Branch struct {
	Name       string
	LastUpdate time.Time
}

func (g *Repo) Branches() ([]*Branch, error) {
	bi, err := g.r.Branches()
	if err != nil {
		return nil, fmt.Errorf("branch: %w", err)
	}

	var branches []*Branch
	err = bi.ForEach(func(r *plumbing.Reference) error {
		cmt, cerr := g.r.CommitObject(r.Hash())
		if cerr != nil {
			return cerr
		}

		branches = append(branches, &Branch{
			Name:       r.Name().Short(),
			LastUpdate: cmt.Committer.When,
		})
		return nil
	})
	return branches, err
}

func (g *Repo) IsGoMod() bool {
	_, err := g.FileContent("go.mod")
	return err == nil
}

func (g *Repo) Fetch(ctx context.Context) (isUpdated bool, err error) {
	return g.fetch(ctx, nil)
}

func (g *Repo) FetchFromGithubWithToken(ctx context.Context, token string) (isUpdated bool, err error) {
	return g.fetch(ctx, &http.BasicAuth{
		Username: "x-access-token", // this can be anything but empty
		Password: token,
	})
}

func (g *Repo) fetch(ctx context.Context, auth transport.AuthMethod) (bool, error) {
	rmt, err := g.r.Remote(originRemote)
	if err != nil {
		return false, fmt.Errorf("failed to get remote: %w", err)
	}

	err = rmt.FetchContext(ctx, &git.FetchOptions{
		Auth:  auth,
		Tags:  git.AllTags,
		Prune: true,
		Force: true,
	})

	isUpdated := !errors.Is(err, git.NoErrAlreadyUpToDate)
	if err != nil && isUpdated {
		return false, fmt.Errorf("failed to fetch: %w", err)
	}

	if !g.IsEmpty() {
		return isUpdated, nil
	}

	refs, err := rmt.List(&git.ListOptions{Auth: auth})
	if err != nil {
		return false, fmt.Errorf("failed to list references: %w", err)
	}

	for _, ref := range refs {
		if ref.Name() == plumbing.HEAD {
			if err := g.r.Storer.SetReference(
				plumbing.NewSymbolicReference(plumbing.HEAD, ref.Target()),
			); err != nil {
				return false, fmt.Errorf("failed to set HEAD: %w", err)
			}
			break
		}
	}

	return isUpdated, nil
}
